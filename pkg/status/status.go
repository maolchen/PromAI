package status

import (
	"context"
	"log"
	"time"

	"PromAI/pkg/config"
	"PromAI/pkg/metrics"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// 添加配置相关的类型定义
type Config struct {
	PrometheusURL string       `yaml:"prometheus_url"`
	MetricTypes   []MetricType `yaml:"metric_types"`
}

type MetricType struct {
	Type    string         `yaml:"type"`
	Metrics []MetricConfig `yaml:"metrics"`
}

type MetricConfig struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	Query         string            `yaml:"query"`
	Threshold     float64           `yaml:"threshold"`
	Unit          string            `yaml:"unit"`
	Labels        map[string]string `yaml:"labels"`
	ThresholdType string            `yaml:"threshold_type"`
}

type StatusSummary struct {
	Normal       int
	Warning      int // 新增警告状态计数
	Abnormal     int
	TotalMetrics int            // 总指标数
	TypeCounts   map[string]int // 每种类型的指标数量
}

type MetricStatus struct {
	Name          string
	DailyStatus   map[string]string // key是日期，value是状态("normal"/"warning"/"abnormal")
	Threshold     float64
	Unit          string
	ThresholdType string
}

type StatusData struct {
	Summary StatusSummary
	Metrics []MetricStatus
	Dates   []string
}

func GenerateStatusData(days int) (*StatusData, error) {
	data := &StatusData{
		Summary: StatusSummary{
			TypeCounts: make(map[string]int), // 初始化类型计数map
		},
		Metrics: []MetricStatus{},
		Dates:   make([]string, days),
	}

	// 生成最近n天的日期
	now := time.Now()
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)
		data.Dates[days-1-i] = date.Format("01-02") // MM-DD格式
	}

	return data, nil
}

func CollectMetricStatus(client metrics.PrometheusAPI, config *config.Config) (*StatusData, error) {
	data, err := GenerateStatusData(7) // 显示最近7天的数据
	if err != nil {
		log.Printf("生成状态数据失败: %v", err)
		return nil, err
	}

	log.Printf("开始收集指标状态数据，时间范围: %v", data.Dates)

	// 遍历所有指标类型
	for _, metricType := range config.MetricTypes {
		log.Printf("处理指标类型: %s", metricType.Type)

		// 统计每种类型的指标数量
		data.Summary.TypeCounts[metricType.Type] = len(metricType.Metrics)
		// 累加总指标数
		data.Summary.TotalMetrics += len(metricType.Metrics)

		// 遍历每个指标
		for _, metric := range metricType.Metrics {
			log.Printf("处理指标: %s (阈值: %v %s, 阈值类型: %s)",
				metric.Name, metric.Threshold, metric.Unit, metric.ThresholdType)

			metricStatus := MetricStatus{
				Name:          metric.Name,
				DailyStatus:   make(map[string]string),
				Threshold:     metric.Threshold,
				Unit:          metric.Unit,
				ThresholdType: metric.ThresholdType,
			}

			// 查询每天的状态
			for _, date := range data.Dates {
				status, err := queryMetricStatus(client, metric, date)
				if err != nil {
					log.Printf("查询指标 [%s] 在 %s 的状态失败: %v", metric.Name, date, err)
					metricStatus.DailyStatus[date] = "abnormal"
					data.Summary.Abnormal++
				} else {
					metricStatus.DailyStatus[date] = status
					switch status {
					case "normal":
						log.Printf("指标 [%s] 在 %s 状态正常", metric.Name, date)
						data.Summary.Normal++
					case "warning":
						log.Printf("指标 [%s] 在 %s 状态警告", metric.Name, date)
						data.Summary.Warning++
					case "abnormal":
						log.Printf("指标 [%s] 在 %s 状态异常", metric.Name, date)
						data.Summary.Abnormal++
					}
				}
			}

			data.Metrics = append(data.Metrics, metricStatus)
		}
	}

	log.Printf("状态数据收集完成. 总指标数: %d, 正常: %d, 警告: %d, 异常: %d",
		data.Summary.TotalMetrics, data.Summary.Normal, data.Summary.Warning, data.Summary.Abnormal)

	// 打印每种类型的指标数量
	for typeName, count := range data.Summary.TypeCounts {
		log.Printf("指标类型 [%s] 包含 %d 个指标", typeName, count)
	}

	return data, nil
}

func queryMetricStatus(client metrics.PrometheusAPI, metric config.MetricConfig, date string) (string, error) {
	ctx := context.Background()

	dateTime, err := time.Parse("01-02", date)
	if err != nil {
		return "abnormal", err
	}

	// 设置查询时间范围为那一天的0点到23:59:59
	startTime := time.Date(time.Now().Year(), dateTime.Month(), dateTime.Day(), 0, 0, 0, 0, time.Local)
	endTime := startTime.Add(24 * time.Hour).Add(-time.Second)

	log.Printf(`
查询指标: [%s]
时间范围: %s 到 %s
PromQL: %s
调试步骤:
1. 打开 Prometheus UI
2. 粘贴查询: %s
3. 设置时间范围为: %s 到 %s
-------------------`,
		metric.Name,
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"),
		metric.Query,
		metric.Query,
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"))

	// 直接使用原始查询语句
	result, _, err := client.QueryRange(ctx, metric.Query, v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  time.Hour, // 每小时一个采样点
	})

	if err != nil {
		log.Printf("执行查询失败 [%s]: %v", metric.Query, err)
		return "abnormal", err
	}

	switch v := result.(type) {
	case model.Matrix:
		if len(v) == 0 {
			log.Printf("指标 [%s] 查询结果为空", metric.Name)
			return "abnormal", nil
		}

		log.Printf("指标 [%s] 返回 %d 个时间序列", metric.Name, len(v))

		maxValue := float64(0)
		// 遍历每个时间序列
		for _, series := range v {
			// 遍历每个采样点，找出最大值
			for _, sample := range series.Values {
				value := float64(sample.Value)
				if value > maxValue {
					maxValue = value
				}
				log.Printf("指标 [%s] 时间: %v, 值: %v",
					metric.Name,
					sample.Timestamp.Time().Format("15:04:05"),
					value)
			}
		}

		// 使用最大值进行阈值判断
		status := checkThreshold(maxValue, metric.Threshold, metric.ThresholdType)
		log.Printf("指标 [%s] 最大值: %v, 阈值: %v, 阈值类型: %s, 状态: %s",
			metric.Name,
			maxValue,
			metric.Threshold,
			metric.ThresholdType,
			status)

		return status, nil

	default:
		log.Printf("指标 [%s] 返回了意外的结果类型: %T", metric.Name, result)
		return "abnormal", nil
	}
}

// 根据阈值类型判断状态
func checkThreshold(value, threshold float64, thresholdType string) string {
	if thresholdType == "" {
		thresholdType = "greater" // 默认值
	}

	// 警告阈值为正常阈值的90%
	warningFactor := 0.9

	switch thresholdType {
	case "greater":
		// 当值大于阈值时告警
		// 例如：CPU使用率 > 80% 告警
		if value > threshold {
			return "abnormal"
		} else if value > threshold*warningFactor {
			return "warning"
		}
		return "normal"
	case "greater_equal":
		// 当值大于等于阈值时告警
		if value >= threshold {
			return "abnormal"
		} else if value >= threshold*warningFactor {
			return "warning"
		}
		return "normal"
	case "less":
		// 当值小于阈值时告警
		// 例如：可用节点数 < 3 告警
		if value < threshold {
			return "abnormal"
		} else if value < threshold/warningFactor {
			return "warning"
		}
		return "normal"
	case "less_equal":
		// 当值小于等于阈值时告警
		if value <= threshold {
			return "abnormal"
		} else if value <= threshold/warningFactor {
			return "warning"
		}
		return "normal"
	case "equal":
		// 值必须等于阈值才正常
		if value == threshold {
			return "normal"
		}
		return "abnormal"
	case "not_equal":
		// 值不等于阈值才正常
		if value != threshold {
			return "normal"
		}
		return "abnormal"
	default:
		// 默认情况：大于阈值告警
		if value > threshold {
			return "abnormal"
		} else if value > threshold*warningFactor {
			return "warning"
		}
		return "normal"
	}
}
