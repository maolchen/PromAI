package metrics

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"PromAI/pkg/config"
	"PromAI/pkg/report"
)

// Collector 处理指标收集
type Collector struct {
	Client PrometheusAPI
	config *config.Config
}

type PrometheusAPI interface {
	Query(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error)
	QueryRange(ctx context.Context, query string, r v1.Range, opts ...v1.Option) (model.Value, v1.Warnings, error)
}

// NewCollector 创建新的收集器
func NewCollector(client PrometheusAPI, config *config.Config) *Collector {
	return &Collector{
		Client: client,
		config: config,
	}
}

// CollectMetrics 收集指标数据
func (c *Collector) CollectMetrics() (*report.ReportData, error) {
	ctx := context.Background()

	data := &report.ReportData{
		Timestamp:    time.Now(),
		MetricGroups: make(map[string]*report.MetricGroup),
		GroupOrder:   make([]string, 0, len(c.config.MetricTypes)),
		ChartData:    make(map[string]template.JS),
		Project:      c.config.ProjectName,
	}

	for _, metricType := range c.config.MetricTypes {
		group := &report.MetricGroup{
			Type:          metricType.Type,
			MetricsByName: make(map[string][]report.MetricData),
			MetricOrder:   make([]string, 0, len(metricType.Metrics)),
		}
		data.MetricGroups[metricType.Type] = group
		data.GroupOrder = append(data.GroupOrder, metricType.Type)

		for _, metric := range metricType.Metrics {
			result, _, err := c.Client.Query(ctx, metric.Query, time.Now())
			if err != nil {
				log.Printf("警告: 查询指标 %s 失败: %v", metric.Name, err)
				continue
			}
			log.Printf("指标 [%s] 查询结果: %+v", metric.Name, result)

			switch v := result.(type) {
			case model.Vector:
				metrics := make([]report.MetricData, 0, len(v))
				for _, sample := range v {
					log.Printf("指标 [%s] 原始数据: %+v, 值: %+v", metric.Name, sample.Metric, sample.Value)

					availableLabels := make(map[string]string)
					for labelName, labelValue := range sample.Metric {
						availableLabels[string(labelName)] = string(labelValue)
					}

					labels := make([]report.LabelData, 0, len(metric.Labels))
					for configLabel, configAlias := range metric.Labels {
						labelValue := "-"
						if rawValue, exists := availableLabels[configLabel]; exists && rawValue != "" {
							labelValue = rawValue
						} else {
							log.Printf("警告: 指标 [%s] 标签 [%s] 缺失或为空", metric.Name, configLabel)
						}

						labels = append(labels, report.LabelData{
							Name:  configLabel,
							Alias: configAlias,
							Value: labelValue,
						})
					}

					if !validateLabels(labels) {
						log.Printf("警告: 指标 [%s] 标签数据不完整，跳过该条记录", metric.Name)
						continue
					}

					metricData := report.MetricData{
						Name:        metric.Name,
						Description: metric.Description,
						Value:       float64(sample.Value),
						Threshold:   metric.Threshold,
						Unit:        metric.Unit,
						Status:      getStatus(float64(sample.Value), metric.Threshold, metric.ThresholdType),
						StatusText:  report.GetStatusText(getStatus(float64(sample.Value), metric.Threshold, metric.ThresholdType)),
						Timestamp:   time.Now(),
						Labels:      labels,
					}

					if err := validateMetricData(metricData, metric.Labels); err != nil {
						log.Printf("警告: 指标 [%s] 数据验证失败: %v", metric.Name, err)
						continue
					}

					metrics = append(metrics, metricData)
				}
				group.MetricsByName[metric.Name] = metrics
				group.MetricOrder = append(group.MetricOrder, metric.Name)
			}
		}
	}
	return data, nil
}

// validateMetricData 验证指标数据的完整性
func validateMetricData(data report.MetricData, configLabels map[string]string) error {
	if len(data.Labels) != len(configLabels) {
		return fmt.Errorf("标签数量不匹配: 期望 %d, 实际 %d",
			len(configLabels), len(data.Labels))
	}

	labelMap := make(map[string]bool)
	for _, label := range data.Labels {
		if _, exists := configLabels[label.Name]; !exists {
			return fmt.Errorf("发现未配置的标签: %s", label.Name)
		}
		if label.Value == "" || label.Value == "-" {
			return fmt.Errorf("标签 %s 值为空", label.Name)
		}
		labelMap[label.Name] = true
	}

	return nil
}

// getStatus 获取状态
func getStatus(value, threshold float64, thresholdType string) string {
	if thresholdType == "" {
		thresholdType = "greater"
	}
	switch thresholdType {
	case "greater":
		if value > threshold {
			return "critical"
		} else if value >= threshold*0.8 {
			return "warning"
		}
	case "greater_equal":
		if value >= threshold {
			return "critical"
		} else if value >= threshold*0.8 {
			return "warning"
		}
	case "less":
		if value < threshold {
			return "normal"
		} else if value <= threshold*1.2 {
			return "warning"
		}
	case "less_equal":
		if value <= threshold {
			return "normal"
		} else if value <= threshold*1.2 {
			return "warning"
		}
	case "equal":
		if value == threshold {
			return "normal"
		} else if value > threshold {
			return "critical"
		}
		return "critical"
	}
	return "normal"
}

// validateLabels 验证标签数据的完整性
func validateLabels(labels []report.LabelData) bool {
	for _, label := range labels {
		if label.Value == "" || label.Value == "-" {
			return false
		}
	}
	return true
}
