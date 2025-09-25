package main

import (
	"fmt"
)

func main() {
	fmt.Println("=== 全面测试监控类指标阈值判断功能 ===")
	
	// 测试所有阈值类型
	tests := []struct {
		name          string
		value         float64
		threshold     float64
		thresholdType string
		expected      string
		description   string
	}{
		// greater类型测试
		{"CPU使用率-正常", 50, 80, "greater", "normal", "50 <= 80"},
		{"CPU使用率-警告", 70, 80, "greater", "warning", "70 >= 80*0.8(64)"},
		{"CPU使用率-严重", 90, 80, "greater", "critical", "90 > 80"},
		
		// greater_equal类型测试
		{"内存使用率-正常", 60, 85, "greater_equal", "normal", "60 < 85"},
		{"内存使用率-警告", 70, 85, "greater_equal", "warning", "70 >= 85*0.8(68)"},
		{"内存使用率-严重", 85, 85, "greater_equal", "critical", "85 >= 85"},
		
		// equal类型测试 (K8s状态)
		{"节点就绪-正常", 0, 0, "equal", "normal", "0 == 0"},
		{"节点就绪-异常", 1, 0, "equal", "critical", "1 != 0"},
		{"Pod运行-正常", 1, 1, "equal", "normal", "1 == 1"},
		{"Pod运行-异常", 0, 1, "equal", "critical", "0 != 1"},
		
		// less类型测试
		{"响应时间-正常", 100, 200, "less", "normal", "100 >= 200"},
		{"响应时间-警告", 250, 200, "less", "warning", "250 <= 200*1.2(240)"},
		{"响应时间-严重", 150, 200, "less", "critical", "150 < 200"},
		
		// less_equal类型测试  
		{"可用节点-正常", 5, 3, "less_equal", "normal", "5 > 3"},
		{"可用节点-警告", 4, 3, "less_equal", "warning", "4 <= 3*1.2(3.6)"},
		{"可用节点-严重", 3, 3, "less_equal", "critical", "3 <= 3"},
	}
	
	fmt.Println("\n监控类指标阈值判断测试结果:")
	fmt.Println("指标类型\t\t\t值\t阈值\t类型\t\t期望\t实际\t状态\t说明")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	
	allPassed := true
	for _, test := range tests {
		result := getStatus(test.value, test.threshold, test.thresholdType, "monitoring")
		status := "✓"
		if result != test.expected {
			status = "✗"
			allPassed = false
		}
		
		fmt.Printf("%-20s\t%.0f\t%.0f\t%-12s\t%s\t%s\t%s\t%s\n", 
			test.name, test.value, test.threshold, test.thresholdType, 
			test.expected, result, status, test.description)
	}
	
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	if allPassed {
		fmt.Println("✅ 所有监控类指标阈值判断功能测试通过！")
	} else {
		fmt.Println("❌ 部分测试失败，需要检查实现。")
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
	case "greater_equal":
		if value >= threshold {
			return "critical"
		} else if value >= threshold*0.8 {
			return "warning"
		}
	case "less":
		if value < threshold {
			return "critical"
		} else if value <= threshold*1.2 {
			return "warning"
		}
		return "normal"
	case "less_equal":
		if value <= threshold {
			return "critical"
		} else if value <= threshold*1.2 {
			return "warning"
		}
		return "normal"
	case "equal":
		if value == threshold {
			return "normal"
		}
		return "critical"
	}
	return "normal"
}