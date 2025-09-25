package config

import (
	"testing"
	"gopkg.in/yaml.v2"
)

// boolPtr 返回bool指针的辅助函数
func boolPtr(b bool) *bool {
	return &b
}

func TestMetricConfigShowInTableParsing(t *testing.T) {
	// 测试YAML解析show_in_table字段
	yamlData := `
metric_types:
- type: "test-group"
  metrics:
  - name: "test-metric-true"
    type: "monitoring"
    show_in_table: true
    query: "test_query_1"
    description: "测试指标1"
    unit: "%"
    labels:
      instance: "节点"
      
  - name: "test-metric-false"
    type: "display"
    show_in_table: false
    query: "test_query_2" 
    description: "测试指标2"
    unit: "B"
    labels:
      instance: "节点"
      
  - name: "test-metric-default"
    type: "monitoring"
    # 没有show_in_table字段，应该使用默认值
    query: "test_query_3"
    description: "测试指标3"
    unit: "%"
    labels:
      instance: "节点"
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("YAML解析失败: %v", err)
	}

	if len(config.MetricTypes) != 1 {
		t.Fatalf("期望1个指标类型，实际%d个", len(config.MetricTypes))
	}

	metricType := config.MetricTypes[0]
	if len(metricType.Metrics) != 3 {
		t.Fatalf("期望3个指标，实际%d个", len(metricType.Metrics))
	}

	// 测试show_in_table: true
	metric1 := metricType.Metrics[0]
	if metric1.Name != "test-metric-true" {
		t.Errorf("第一个指标名称不正确")
	}
	if metric1.ShowInTable == nil || !*metric1.ShowInTable {
		t.Errorf("show_in_table: true的指标应该解析为true")
	}

	// 测试show_in_table: false
	metric2 := metricType.Metrics[1]
	if metric2.Name != "test-metric-false" {
		t.Errorf("第二个指标名称不正确")
	}
	if metric2.ShowInTable == nil || *metric2.ShowInTable {
		t.Errorf("show_in_table: false的指标应该解析为false")
	}

	// 测试默认值（没有show_in_table字段）
	metric3 := metricType.Metrics[2]
	if metric3.Name != "test-metric-default" {
		t.Errorf("第三个指标名称不正确")
	}
	if metric3.ShowInTable != nil {
		t.Logf("未设置show_in_table字段的指标，ShowInTable值为: %v", metric3.ShowInTable)
	}
}

func TestConfigStructureValidation(t *testing.T) {
	// 测试完整的配置结构
	config := Config{
		PrometheusURL: "http://localhost:9090",
		ProjectName:   "测试项目",
		CronSchedule:  "0 */1 * * *",
		MetricTypes: []MetricType{
			{
				Type: "基础资源",
				Metrics: []MetricConfig{
					{
						Name:          "CPU使用率",
						Type:          "monitoring",
						ShowInTable:   boolPtr(true),
						Description:   "CPU使用率",
						Query:         "test_cpu_query",
						Threshold:     80.0,
						ThresholdType: "greater",
						Unit:          "%",
						Labels: map[string]string{
							"instance": "节点",
						},
					},
					{
						Name:        "CPU核心数",
						Type:        "display",
						ShowInTable: boolPtr(false),
						Description: "CPU核心数",
						Query:       "test_cpu_cores_query",
						Unit:        "core",
						Labels: map[string]string{
							"instance": "节点",
						},
					},
				},
			},
		},
	}

	// 验证结构正确性
	if config.PrometheusURL != "http://localhost:9090" {
		t.Error("PrometheusURL设置不正确")
	}

	if len(config.MetricTypes) != 1 {
		t.Error("MetricTypes数量不正确")
	}

	metricType := config.MetricTypes[0]
	if metricType.Type != "基础资源" {
		t.Error("MetricType.Type设置不正确")
	}

	if len(metricType.Metrics) != 2 {
		t.Error("Metrics数量不正确")
	}

	// 验证监控类指标
	monitoringMetric := metricType.Metrics[0]
	if monitoringMetric.ShowInTable == nil || !*monitoringMetric.ShowInTable {
		t.Error("监控类指标ShowInTable应该为true")
	}
	if monitoringMetric.Type != "monitoring" {
		t.Error("监控类指标Type不正确")
	}

	// 验证展示类指标
	displayMetric := metricType.Metrics[1]
	if displayMetric.ShowInTable == nil || *displayMetric.ShowInTable {
		t.Error("展示类指标ShowInTable应该为false")
	}
	if displayMetric.Type != "display" {
		t.Error("展示类指标Type不正确")
	}
}