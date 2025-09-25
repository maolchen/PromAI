package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"PromAI/pkg/config"
)

// boolPtr 返回bool指针的辅助函数
func boolPtr(b bool) *bool {
	return &b
}

// MockPrometheusAPI 模拟Prometheus API
type MockPrometheusAPI struct {
	responses map[string]model.Value
}

func (m *MockPrometheusAPI) Query(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error) {
	if response, exists := m.responses[query]; exists {
		return response, nil, nil
	}
	// 返回默认的Vector值
	return model.Vector{
		&model.Sample{
			Metric: model.Metric{
				"instance": "test-instance",
			},
			Value:     42.0,
			Timestamp: model.TimeFromUnix(ts.Unix()),
		},
	}, nil, nil
}

func (m *MockPrometheusAPI) QueryRange(ctx context.Context, query string, r v1.Range, opts ...v1.Option) (model.Value, v1.Warnings, error) {
	return nil, nil, nil
}

func TestCollectorShowInTableFiltering(t *testing.T) {
	// 创建测试配置
	testConfig := &config.Config{
		MetricTypes: []config.MetricType{
			{
				Type: "test-group",
				Metrics: []config.MetricConfig{
					{
						Name:        "monitoring-metric",
						Type:        "monitoring",
						ShowInTable: boolPtr(true), // 应该显示在表格中
						Query:       "test_monitoring_query",
						Description: "测试监控指标",
						Unit:        "%",
						Labels: map[string]string{
							"instance": "节点",
						},
					},
					{
						Name:        "display-metric-hidden",
						Type:        "display",
						ShowInTable: boolPtr(false), // 不应该显示在表格中
						Query:       "test_display_query_hidden",
						Description: "测试展示指标（隐藏）",
						Unit:        "B",
						Labels: map[string]string{
							"instance": "节点",
						},
					},
					{
						Name:        "display-metric-shown",
						Type:        "display",
						ShowInTable: boolPtr(true), // 应该显示在表格中
						Query:       "test_display_query_shown",
						Description: "测试展示指标（显示）",
						Unit:        "B",
						Labels: map[string]string{
							"instance": "节点",
						},
					},
					{
						Name: "metric-default-behavior",
						Type: "monitoring",
						// ShowInTable 未设置，应该默认为true
						Query:       "test_default_query",
						Description: "测试默认行为指标",
						Unit:        "%",
						Labels: map[string]string{
							"instance": "节点",
						},
					},
				},
			},
		},
	}

	// 创建Mock API
	mockAPI := &MockPrometheusAPI{
		responses: make(map[string]model.Value),
	}

	// 创建收集器
	collector := NewCollector(mockAPI, testConfig)

	// 执行收集
	reportData, err := collector.CollectMetrics()
	if err != nil {
		t.Fatalf("收集指标失败: %v", err)
	}

	// 验证结果
	if len(reportData.MetricGroups) != 1 {
		t.Fatalf("期望1个指标组，实际得到%d个", len(reportData.MetricGroups))
	}

	group := reportData.MetricGroups["test-group"]
	if group == nil {
		t.Fatal("未找到test-group指标组")
	}

	// 验证MetricsByName包含所有指标数据
	expectedMetricsInData := []string{
		"monitoring-metric",
		"display-metric-hidden",
		"display-metric-shown",
		"metric-default-behavior",
	}

	for _, metricName := range expectedMetricsInData {
		if _, exists := group.MetricsByName[metricName]; !exists {
			t.Errorf("MetricsByName中缺少指标: %s", metricName)
		}
	}

	// 验证MetricOrder只包含show_in_table为true的指标
	expectedMetricsInOrder := []string{
		"monitoring-metric",
		"display-metric-shown",
		"metric-default-behavior",
	}

	if len(group.MetricOrder) != len(expectedMetricsInOrder) {
		t.Errorf("MetricOrder长度不正确，期望%d，实际%d", len(expectedMetricsInOrder), len(group.MetricOrder))
	}

	// 验证MetricOrder中的指标
	for i, expectedMetric := range expectedMetricsInOrder {
		if i >= len(group.MetricOrder) {
			t.Errorf("MetricOrder中缺少指标: %s", expectedMetric)
			continue
		}
		if group.MetricOrder[i] != expectedMetric {
			t.Errorf("MetricOrder[%d]不正确，期望%s，实际%s", i, expectedMetric, group.MetricOrder[i])
		}
	}

	// 验证show_in_table=false的指标不在MetricOrder中
	for _, metricName := range group.MetricOrder {
		if metricName == "display-metric-hidden" {
			t.Error("show_in_table=false的指标不应该出现在MetricOrder中")
		}
	}

	t.Logf("测试通过：MetricOrder包含%d个指标，MetricsByName包含%d个指标", 
		len(group.MetricOrder), len(group.MetricsByName))
}

func TestCollectorBackwardCompatibility(t *testing.T) {
	// 测试向后兼容性：旧配置文件（没有show_in_table字段）应该正常工作
	testConfig := &config.Config{
		MetricTypes: []config.MetricType{
			{
				Type: "backward-compatibility-test",
				Metrics: []config.MetricConfig{
					{
						Name:        "old-config-metric",
						Type:        "monitoring",
						// 没有ShowInTable字段
						Query:       "test_old_config_query",
						Description: "测试旧配置兼容性",
						Unit:        "%",
						Labels: map[string]string{
							"instance": "节点",
						},
					},
				},
			},
		},
	}

	mockAPI := &MockPrometheusAPI{
		responses: make(map[string]model.Value),
	}

	collector := NewCollector(mockAPI, testConfig)
	reportData, err := collector.CollectMetrics()
	if err != nil {
		t.Fatalf("收集指标失败: %v", err)
	}

	group := reportData.MetricGroups["backward-compatibility-test"]
	if group == nil {
		t.Fatal("未找到backward-compatibility-test指标组")
	}

	// 旧配置应该默认显示所有指标
	if len(group.MetricOrder) != 1 {
		t.Errorf("向后兼容性测试失败：MetricOrder应该包含1个指标，实际%d个", len(group.MetricOrder))
	}

	if group.MetricOrder[0] != "old-config-metric" {
		t.Errorf("向后兼容性测试失败：MetricOrder[0]应该是'old-config-metric'，实际是'%s'", group.MetricOrder[0])
	}
}

func TestDisplayMetricStatusBehavior(t *testing.T) {
	// 测试展示类指标的状态行为
	testConfig := &config.Config{
		MetricTypes: []config.MetricType{
			{
				Type: "status-test",
				Metrics: []config.MetricConfig{
					{
						Name:        "display-metric-status",
						Type:        "display",
						ShowInTable: boolPtr(true),
						Query:       "test_display_status",
						Description: "测试展示指标状态",
						Unit:        "B",
						Threshold:   100, // 设置阈值
						ThresholdType: "greater",
						Labels: map[string]string{
							"instance": "节点",
						},
					},
				},
			},
		},
	}

	// 设置高于阈值的值
	mockAPI := &MockPrometheusAPI{
		responses: map[string]model.Value{
			"test_display_status": model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"instance": "test-instance",
					},
					Value:     150.0, // 高于阈值100
					Timestamp: model.TimeFromUnix(time.Now().Unix()),
				},
			},
		},
	}

	collector := NewCollector(mockAPI, testConfig)
	reportData, err := collector.CollectMetrics()
	if err != nil {
		t.Fatalf("收集指标失败: %v", err)
	}

	group := reportData.MetricGroups["status-test"]
	metrics := group.MetricsByName["display-metric-status"]
	
	if len(metrics) == 0 {
		t.Fatal("未找到展示指标数据")
	}

	// 展示类指标应该始终返回normal状态，不受阈值影响
	if metrics[0].Status != "normal" {
		t.Errorf("展示类指标状态应该是'normal'，实际是'%s'", metrics[0].Status)
	}
}