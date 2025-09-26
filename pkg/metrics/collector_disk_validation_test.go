package metrics

import (
	"testing"
)

// TestValidateDiskData 测试磁盘数据验证函数
func TestValidateDiskData(t *testing.T) {
	tests := []struct {
		name        string
		metricName  string
		value       float64
		expectError bool
		errorMsg    string
	}{
		// 磁盘总量测试用例
		{
			name:        "磁盘总量_正常值",
			metricName:  "磁盘总量",
			value:       1024 * 1024 * 1024 * 100, // 100GB
			expectError: false,
		},
		{
			name:        "磁盘总量_负数值",
			metricName:  "磁盘总量",
			value:       -1024,
			expectError: true,
			errorMsg:    "磁盘总量不能为负数或零",
		},
		{
			name:        "磁盘总量_零值",
			metricName:  "磁盘总量",
			value:       0,
			expectError: true,
			errorMsg:    "磁盘总量不能为负数或零",
		},
		{
			name:        "磁盘总量_异常过大",
			metricName:  "磁盘总量",
			value:       2 * 1024 * 1024 * 1024 * 1024 * 1024, // 2PB 过大
			expectError: true,
			errorMsg:    "磁盘总量异常过大",
		},
		
		// 磁盘可用量测试用例
		{
			name:        "磁盘可用量_正常值",
			metricName:  "磁盘可用量",
			value:       1024 * 1024 * 1024 * 50, // 50GB
			expectError: false,
		},
		{
			name:        "磁盘可用量_零值",
			metricName:  "磁盘可用量",
			value:       0,
			expectError: false, // 可用量为零是合理的
		},
		{
			name:        "磁盘可用量_负数值",
			metricName:  "磁盘可用量",
			value:       -1024,
			expectError: true,
			errorMsg:    "磁盘可用量不能为负数",
		},
		
		// 磁盘使用率测试用例
		{
			name:        "磁盘使用率_正常值_50%",
			metricName:  "磁盘使用率",
			value:       50.5,
			expectError: false,
		},
		{
			name:        "磁盘使用率_边界值_0%",
			metricName:  "磁盘使用率",
			value:       0.0,
			expectError: false,
		},
		{
			name:        "磁盘使用率_边界值_100%",
			metricName:  "磁盘使用率",
			value:       100.0,
			expectError: false,
		},
		{
			name:        "磁盘使用率_负数值",
			metricName:  "磁盘使用率",
			value:       -10.5,
			expectError: true,
			errorMsg:    "磁盘使用率超出合理范围(0-100%)",
		},
		{
			name:        "磁盘使用率_超过100%",
			metricName:  "磁盘使用率",
			value:       105.8,
			expectError: true,
			errorMsg:    "磁盘使用率超出合理范围(0-100%)",
		},
		
		// 非磁盘指标测试用例
		{
			name:        "非磁盘指标_CPU使用率",
			metricName:  "CPU使用率",
			value:       75.5,
			expectError: false, // 非磁盘指标不会被磁盘验证函数处理
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDiskData(tt.metricName, tt.value, nil)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误但没有返回错误，测试用例: %s", tt.name)
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("错误信息不匹配，期望包含: %s, 实际: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误但返回了错误: %v，测试用例: %s", err, tt.name)
				}
			}
		})
	}
}

// TestValidateMetricValue 测试指标数值验证函数
func TestValidateMetricValue(t *testing.T) {
	tests := []struct {
		name        string
		metricName  string
		value       float64
		expectError bool
	}{
		{
			name:        "有效磁盘使用率",
			metricName:  "磁盘使用率",
			value:       85.5,
			expectError: false,
		},
		{
			name:        "无效磁盘使用率_负数",
			metricName:  "磁盘使用率",
			value:       -276114550784.00, // 模拟bug中的负数值
			expectError: true,
		},
		{
			name:        "有效内存使用率",
			metricName:  "内存使用率",
			value:       70.2,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMetricValue(tt.metricName, tt.value)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误但没有返回错误，测试用例: %s", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误但返回了错误: %v，测试用例: %s", err, tt.name)
				}
			}
		})
	}
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr ||
		      containsInMiddle(s, substr))))
}

// containsInMiddle 检查字符串中间是否包含子字符串
func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}