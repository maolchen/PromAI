package main

import (
	"fmt"
)

func main() {
	fmt.Println("=== 验证实际配置中的监控类指标阈值判断功能 ===")
	
	// 实际配置中使用的监控类指标测试
	tests := []struct {
		name          string
		value         float64
		threshold     float64
		thresholdType string
		expected      string
		description   string
	}{
		// CPU使用率 (threshold: 80, type: greater)
		{"CPU使用率-正常", 50, 80, "greater", "normal", "配置中实际使用"},
		{"CPU使用率-警告", 70, 80, "greater", "warning", "配置中实际使用"},
		{"CPU使用率-严重", 90, 80, "greater", "critical", "配置中实际使用"},
		
		// 内存使用率 (threshold: 85, type: greater)
		{"内存使用率-正常", 60, 85, "greater", "normal", "配置中实际使用"},
		{"内存使用率-警告", 70, 85, "greater", "warning", "配置中实际使用"},
		{"内存使用率-严重", 90, 85, "greater", "critical", "配置中实际使用"},
		
		// 磁盘使用率 (threshold: 80, type: greater)
		{"磁盘使用率-正常", 50, 80, "greater", "normal", "配置中实际使用"},
		{"磁盘使用率-警告", 70, 80, "greater", "warning", "配置中实际使用"},
		{"磁盘使用率-严重", 90, 80, "greater", "critical", "配置中实际使用"},
		
		// 节点就绪状态 (threshold: 0, type: equal)
		{"节点就绪-正常", 0, 0, "equal", "normal", "配置中实际使用"},
		{"节点就绪-异常", 1, 0, "equal", "critical", "配置中实际使用"},
		
		// Pod运行状态 (threshold: 1, type: equal)
		{"Pod运行-正常", 1, 1, "equal", "normal", "配置中实际使用"},
		{"Pod运行-异常", 0, 1, "equal", "critical", "配置中实际使用"},
		
		// PVC使用率 (threshold: 90, type: greater)
		{"PVC使用率-正常", 60, 90, "greater", "normal", "配置中实际使用"},
		{"PVC使用率-警告", 80, 90, "greater", "warning", "配置中实际使用"},
		{"PVC使用率-严重", 95, 90, "greater", "critical", "配置中实际使用"},
	}
	
	fmt.Println("\n实际配置监控类指标阈值判断测试结果:")
	fmt.Println("指标类型\t\t\t值\t阈值\t类型\t\t期望\t实际\t状态")
	fmt.Println("─────────────────────────────────────────────────────────────────────")
	
	allPassed := true
	for _, test := range tests {
		result := getStatus(test.value, test.threshold, test.thresholdType, "monitoring")
		status := "✓"
		if result != test.expected {
			status = "✗"
			allPassed = false
		}
		
		fmt.Printf("%-20s\t%.0f\t%.0f\t%-8s\t%s\t%s\t%s\n", 
			test.name, test.value, test.threshold, test.thresholdType, 
			test.expected, result, status)
	}
	
	fmt.Println("─────────────────────────────────────────────────────────────────────")
	if allPassed {
		fmt.Println("✅ 所有实际配置的监控类指标阈值判断功能测试通过！")
		fmt.Println("   - greater类型阈值判断正确")
		fmt.Println("   - equal类型阈值判断正确")
		fmt.Println("   - 三级状态判断机制正常 (normal/warning/critical)")
	} else {
		fmt.Println("❌ 部分测试失败，需要检查实现。")
	}
	
	fmt.Println("\n=== 验证完成 ===")
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
		return "critical"
	case "less_equal":
		if value <= threshold {
			return "normal"
		} else if value <= threshold*1.2 {
			return "warning"
		}
		return "critical"
	case "equal":
		if value == threshold {
			return "normal"
		}
		return "critical"
	}
	return "normal"
}