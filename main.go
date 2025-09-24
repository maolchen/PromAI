package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"PromAI/pkg/config"
	"PromAI/pkg/metrics"
	"PromAI/pkg/notify"
	"PromAI/pkg/prometheus"
	"PromAI/pkg/report"
	"PromAI/pkg/status"
	"PromAI/pkg/utils"

	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v2"
)

// loadConfig 加载配置文件
func loadConfig(path string) (*config.Config, error) {
	data, err := os.ReadFile(path) // 读取配置文件
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config config.Config // 定义配置结构体
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	} // 解析配置文件
	// 从环境变量中获取 PrometheusURL
	if envPrometheusURL := os.Getenv("PROMETHEUS_URL"); envPrometheusURL != "" {
		log.Printf("使用环境变量中的 Prometheus URL: %s", envPrometheusURL)
		config.PrometheusURL = envPrometheusURL
	} else {
		log.Printf("使用配置文件中的 Prometheus URL: %s", config.PrometheusURL)
	}
	return &config, nil // 返回配置结构体
}

// setup 初始化应用程序
func setup(configPath string) (*prometheus.Client, *config.Config, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}

	client, err := prometheus.NewClient(config.PrometheusURL)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing Prometheus client: %w", err)
	}

	return client, config, nil
}

func main() {
	configPath := flag.String("config", "config/config.yaml", "Path to configuration file")
	port := flag.String("port", "8091", "Port to run the HTTP server on")
	flag.Parse()

	utils.SetGlobalPort(*port)

	client, config, err := setup(*configPath)
	if err != nil {
		log.Fatalf("Error setting up: %v", err)
	}

	collector := metrics.NewCollector(client.API, config)

	// 设置定时任务
	if config.CronSchedule != "" {
		c := cron.New()
		_, err := c.AddFunc(config.CronSchedule, func() {
			data, err := collector.CollectMetrics()
			if err != nil {
				log.Printf("定时任务收集指标失败: %v", err)
				return
			}

			reportFilePath, err := report.GenerateReport(*data)
			if err != nil {
				log.Printf("定时任务生成报告失败: %v", err)
				return
			}
			log.Printf("定时任务成功生成报告: %s", reportFilePath)

			if config.Notifications.Dingtalk.Enabled {
				log.Printf("发送钉钉消息")
				if err := notify.SendDingtalk(config.Notifications.Dingtalk, reportFilePath); err != nil {
					log.Printf("发送钉钉消息失败: %v", err)
				}
			}

			if config.Notifications.Email.Enabled {
				log.Printf("发送邮件")
				notify.SendEmail(config.Notifications.Email, reportFilePath)
			}

			// ✅ 新增：企业微信通知
			if config.Notifications.Wecom.Enabled {
				log.Printf("发送企业微信消息")
				if err := notify.SendWeCom(config.Notifications.Wecom, reportFilePath); err != nil {
					log.Printf("发送企业微信消息失败: %v", err)
				}
			}

		})

		if err != nil {
			log.Printf("设置定时任务失败: %v", err)
		} else {
			c.Start()
			log.Printf("已启动定时任务，执行计划: %s", config.CronSchedule)
		}
	} else {
		log.Printf("未配置定时任务，请手动触发生成报告")
	}
	if config.ReportCleanup.Enabled {
		// 确定使用哪个计划
		cleanupSchedule := config.ReportCleanup.CronSchedule
		if cleanupSchedule == "" {
			cleanupSchedule = config.CronSchedule
		}

		if cleanupSchedule != "" {
			c := cron.New()
			_, err := c.AddFunc(cleanupSchedule, func() {
				if err := report.CleanupReports(config.ReportCleanup.MaxAge); err != nil {
					log.Printf("报告清理失败: %v", err)
					return
				}
				log.Printf("报告清理成功")
			})

			if err != nil {
				log.Printf("设置清理定时任务失败: %v", err)
			} else {
				c.Start()
				log.Printf("已启动清理定时任务，执行计划: %s", cleanupSchedule)
			}
		} else {
			log.Printf("未配置任何定时任务计划，请手动清理报告")
		}
	}

	// 设置路由处理器
	setupRoutes(collector, config)

	// 启动服务器
	log.Printf("Starting server on port: %s with config: %s", *port, *configPath)
	log.Printf("Prometheus URL: %s", config.PrometheusURL)
	log.Printf("获取报告地址: http://localhost:%s/getreport", *port)
	log.Printf("健康看板地址: http://localhost:%s/status", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}

// setupRoutes 设置 HTTP 路由
func setupRoutes(collector *metrics.Collector, config *config.Config) {
	// 设置报告生成路由
	http.HandleFunc("/getreport", makeReportHandler(collector))

	// 设置静态文件服务
	http.Handle("/reports/", http.StripPrefix("/reports/", http.FileServer(http.Dir("reports"))))

	// 设置状态页面路由
	http.HandleFunc("/status", makeStatusHandler(collector.Client, config))

}

// makeReportHandler 创建报告处理器
func makeReportHandler(collector *metrics.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := collector.CollectMetrics()
		if err != nil {
			http.Error(w, "Failed to collect metrics", http.StatusInternalServerError)
			log.Printf("Error collecting metrics: %v", err)
			return
		}

		reportFilePath, err := report.GenerateReport(*data)
		if err != nil {
			http.Error(w, "Failed to generate report", http.StatusInternalServerError)
			log.Printf("Error generating report: %v", err)
			return
		}

		http.Redirect(w, r, "/"+reportFilePath, http.StatusSeeOther)
	}
}

// makeStatusHandler 创建状态页面处理器
func makeStatusHandler(client metrics.PrometheusAPI, config *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := status.CollectMetricStatus(client, config)
		if err != nil {
			http.Error(w, "Failed to collect status data", http.StatusInternalServerError)
			log.Printf("Error collecting status data: %v", err)
			return
		}

		// 创建模板函数映射
		funcMap := template.FuncMap{
			"now": time.Now,
			"date": func(format string, t time.Time) string {
				return t.Format(format)
			},
		}

		tmpl := template.New("status.html").Funcs(funcMap)
		tmpl, err = tmpl.ParseFiles("templates/status.html")
		if err != nil {
			http.Error(w, "Failed to parse template", http.StatusInternalServerError)
			log.Printf("Error parsing template: %v", err)
			return
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			log.Printf("Error rendering template: %v", err)
			return
		}
	}
}
