package main

import (
	"fmt"
)

func main() {
	fmt.Println("=== 测试指标状态判断逻辑修复 ===")
	
	// 展示类指标测试用例
	displayTests := []struct {
		name  string
		value float64
	}{
		{"CPU核心数", 8.0},
		{"内存总量", 8589934592},
		{"内存使用量", 4294967296},
		{"磁盘总量", 107374182400},
		{"磁盘可用量", 85899345920},
	}
	
	fmt.Println("\n1. 展示类指标测试 (全部应为 normal):")
	for _, test := range displayTests {
		result := getStatus(test.value, 0, "greater", "display")
		status := "✓"
		if result != "normal" {
			status = "✗"
		}
		fmt.Printf("  %s %s: %.0f -> %s\n", status, test.name, test.value, result)
	}
	
	// 监控类指标测试用例  
	fmt.Println("\n2. 监控类指标测试:")
	
	// CPU使用率测试 (阈值80)
	fmt.Println("  CPU使用率 (阈值80%):")
	cpuTests := []struct{ value, expected string }{
		{"50", "normal"}, {"70", "warning"}, {"90", "critical"},
	}
	for _, test := range cpuTests {
		var val float64
		fmt.Sscanf(test.value, "%f", &val)
		result := getStatus(val, 80, "greater", "monitoring")
		status := "✓"
		if result != test.expected {
			status = "✗"
		}
		fmt.Printf("    %s %s%% -> %s (期望: %s)\n", status, test.value, result, test.expected)
	}
	
	// 节点就绪状态测试 (阈值0, equal类型)
	fmt.Println("  节点就绪状态 (阈值0, equal):")
	nodeTests := []struct{ value float64; expected string }{
		{0, "normal"}, {1, "critical"},
	}
	for _, test := range nodeTests {
		result := getStatus(test.value, 0, "equal", "monitoring")
		status := "✓"
		if result != test.expected {
			status = "✗"
		}
		fmt.Printf("    %s %.0f -> %s (期望: %s)\n", status, test.value, result, test.expected)
	}
	
	fmt.Println("\n=== 测试完成 ===")
}

func getStatus(value, threshold float64, thresholdType, metricType string) string {
	if metricType == "display" {
		return "normal"
	}
	
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
	case "equal":
		if value == threshold {
			return "normal"
		}
		return "critical"
	}
	return "normal"
}