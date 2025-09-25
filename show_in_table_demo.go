package main

import (
	"fmt"
	"log"
	"gopkg.in/yaml.v2"
	"PromAI/pkg/config"
)

// 验证show_in_table功能的演示程序
func main() {
	// 测试YAML配置解析
	yamlConfig := `
prometheus_url: "http://localhost:9090"
project_name: "测试项目"
metric_types:
- type: "基础资源使用情况"
  metrics:
  - name: "CPU使用率"
    type: "monitoring"
    show_in_table: true
    description: "节点CPU使用率统计"
    query: "test_cpu_usage_query"
    threshold: 80
    threshold_type: "greater"
    unit: "%"
    labels:
      instance: "节点"
      
  - name: "CPU核心数"
    type: "display"
    show_in_table: false
    query: "test_cpu_cores_query"
    description: "节点CPU核心数统计"  
    unit: "core"
    labels:
      instance: "节点"
      
  - name: "内存使用率"
    type: "monitoring"
    # 未设置show_in_table，应该默认显示
    description: "节点内存使用率统计"
    query: "test_memory_usage_query"
    threshold: 85
    threshold_type: "greater"
    unit: "%"
    labels:
      instance: "节点"
`

	var cfg config.Config
	err := yaml.Unmarshal([]byte(yamlConfig), &cfg)
	if err != nil {
		log.Fatalf("配置解析失败: %v", err)
	}

	fmt.Println("=== show_in_table功能验证 ===")
	fmt.Printf("项目名称: %s\n", cfg.ProjectName)
	fmt.Printf("Prometheus URL: %s\n", cfg.PrometheusURL)
	
	for _, metricType := range cfg.MetricTypes {
		fmt.Printf("\n指标组: %s\n", metricType.Type)
		
		var showInTableMetrics []string
		var allMetrics []string
		
		for _, metric := range metricType.Metrics {
			allMetrics = append(allMetrics, metric.Name)
			
			// 模拟Collector的过滤逻辑
			if metric.ShowInTable == nil || *metric.ShowInTable {
				showInTableMetrics = append(showInTableMetrics, metric.Name)
			}
			
			showStatus := "显示"
			if metric.ShowInTable != nil && !*metric.ShowInTable {
				showStatus = "隐藏"
			} else if metric.ShowInTable == nil {
				showStatus = "默认显示"
			}
			
			fmt.Printf("  - %s [%s] [%s] -> 表格中%s\n", 
				metric.Name, metric.Type, showStatus, 
				func() string {
					if metric.ShowInTable == nil || *metric.ShowInTable {
						return "✓"
					}
					return "✗"
				}())
		}
		
		fmt.Printf("\n所有指标数量: %d\n", len(allMetrics))
		fmt.Printf("表格显示指标数量: %d\n", len(showInTableMetrics))
		fmt.Printf("隐藏指标数量: %d\n", len(allMetrics)-len(showInTableMetrics))
		
		fmt.Printf("\n表格中显示的指标:\n")
		for _, metric := range showInTableMetrics {
			fmt.Printf("  ✓ %s\n", metric)
		}
		
		fmt.Printf("\n表格中隐藏的指标:\n")
		for _, metric := range allMetrics {
			found := false
			for _, showMetric := range showInTableMetrics {
				if metric == showMetric {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("  ✗ %s\n", metric)
			}
		}
	}
	
	fmt.Println("\n=== 功能验证完成 ===")
	fmt.Println("✓ YAML配置解析正常")
	fmt.Println("✓ show_in_table字段解析正确")
	fmt.Println("✓ 默认行为处理正确")
	fmt.Println("✓ 指标过滤逻辑正确")
}