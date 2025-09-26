package metrics

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"strings"
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

	// 添加数据质量统计
	totalMetrics := 0
	validMetrics := 0
	invalidMetrics := 0
	diskAnomalies := 0

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
				log.Printf("警告: 查询指标 %s 失败: %v, PromQL: %s", metric.Name, err, metric.Query)
				continue
			}
			log.Printf("指标 [%s] 查询结果: %+v", metric.Name, result)

			switch v := result.(type) {
			case model.Vector:
				metrics := make([]report.MetricData, 0, len(v))
				for _, sample := range v {
					totalMetrics++
					log.Printf("指标 [%s] 原始数据: %+v, 值: %+v", metric.Name, sample.Metric, sample.Value)

					// 数据验证：在处理数据前先验证数值的合理性
					if err := validateMetricValue(metric.Name, float64(sample.Value)); err != nil {
						invalidMetrics++
						if strings.Contains(metric.Name, "磁盘") {
							diskAnomalies++
						}
						log.Printf("警告: 指标 [%s] 数据验证失败: %v, 原始值: %f", metric.Name, err, float64(sample.Value))
						continue // 跳过异常数据
					}
					validMetrics++

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
						Status:      getStatus(float64(sample.Value), metric.Threshold, metric.ThresholdType, metric.Type),
						StatusText:  report.GetStatusText(getStatus(float64(sample.Value), metric.Threshold, metric.ThresholdType, metric.Type)),
						Timestamp:   time.Now(),
						Labels:      labels,
					}

					if err := validateMetricData(metricData, metric.Labels); err != nil {
						log.Printf("警告: 指标 [%s] 数据验证失败: %v", metric.Name, err)
						continue
					}

					metrics = append(metrics, metricData)
				}
				// 存储所有指标数据到MetricsByName（包括show_in_table=false的）
				group.MetricsByName[metric.Name] = metrics
				
				// 只有show_in_table为true或未设置的指标才添加到MetricOrder
				// 默认行为：如果show_in_table为nil（未设置），则显示；如果显式设置为true，则显示
				if metric.ShowInTable == nil || *metric.ShowInTable {
					group.MetricOrder = append(group.MetricOrder, metric.Name)
				}
			}
		}
	}
	
	// 输出数据质量统计报告
	log.Printf("数据收集统计 - 总指标数: %d, 有效数: %d, 无效数: %d, 磁盘异常: %d", 
		totalMetrics, validMetrics, invalidMetrics, diskAnomalies)
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
func getStatus(value, threshold float64, thresholdType, metricType string) string {
	// 展示类指标直接返回normal状态
	if metricType == "display" {
		return "normal"
	}
	
	// 监控类指标进行阈值判断
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

// validateDiskData 验证磁盘相关数据的合理性
func validateDiskData(metricName string, value float64, labels []report.LabelData) error {
	// 根据指标类型进行不同的验证
	switch metricName {
	case "磁盘总量":
		// 磁盘总量必须大于0
		if value <= 0 {
			return fmt.Errorf("磁盘总量不能为负数或零: %.2f", value)
		}
		// 合理性检查：磁盘总量不应该超过1PB (太大可能是数据异常)
		if value > 1024*1024*1024*1024*1024 { // 1PB in bytes
			return fmt.Errorf("磁盘总量异常过大: %.2f bytes", value)
		}
	
	case "磁盘可用量":
		// 磁盘可用量必须大于等于0
		if value < 0 {
			return fmt.Errorf("磁盘可用量不能为负数: %.2f", value)
		}
	
	case "磁盘使用率":
		// 磁盘使用率必须在0-100%范围内
		if value < 0 || value > 100 {
			return fmt.Errorf("磁盘使用率超出合理范围(0-100%%): %.2f%%", value)
		}
	}
	
	return nil
}

// validateMetricValue 验证指标数值的合理性
func validateMetricValue(metricName string, value float64) error {
	// TODO: 可以添加NaN和无穷大检查，需要导入math包
	
	// 磁盘相关指标的特殊验证
	if err := validateDiskData(metricName, value, nil); err != nil {
		return err
	}
	
	return nil
}
