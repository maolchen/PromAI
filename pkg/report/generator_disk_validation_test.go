package report

import (
	"testing"
)

// TestValidateDiskConsistency 测试磁盘数据一致性验证函数
func TestValidateDiskConsistency(t *testing.T) {
	tests := []struct {
		name           string
		hostSummary    HostSummary
		expectedValues map[string]float64 // 期望的修正后值
		description    string
	}{
		{
			name: "正常磁盘数据_无需修正",
			hostSummary: HostSummary{
				Hostname: "test-host-1",
				DiskData: []DiskInfo{
					{
						MountPoint: "/",
						DiskTotal:  100 * 1024 * 1024 * 1024, // 100GB
						DiskUsed:   50 * 1024 * 1024 * 1024,  // 50GB
						DiskUsage:  50.0,                     // 50%
						Status:     "normal",
					},
				},
			},
			expectedValues: map[string]float64{
				"DiskUsage": 50.0,
				"DiskUsed":  50 * 1024 * 1024 * 1024,
			},
			description: "正常数据应保持不变",
		},
		{
			name: "磁盘使用量为负数_需要修正",
			hostSummary: HostSummary{
				Hostname: "test-host-2",
				DiskData: []DiskInfo{
					{
						MountPoint: "/",
						DiskTotal:  100 * 1024 * 1024 * 1024, // 100GB
						DiskUsed:   -1024 * 1024 * 1024,      // -1GB (异常负数)
						DiskUsage:  -1.0,                     // -1% (异常负数)
						Status:     "critical",
					},
				},
			},
			expectedValues: map[string]float64{
				"DiskUsage": 0.0, // 应修正为0
				"DiskUsed":  0.0, // 应修正为0
			},
			description: "负数使用量应修正为0",
		},
		{
			name: "磁盘使用率超过100%_需要修正",
			hostSummary: HostSummary{
				Hostname: "test-host-3",
				DiskData: []DiskInfo{
					{
						MountPoint: "/",
						DiskTotal:  100 * 1024 * 1024 * 1024, // 100GB
						DiskUsed:   120 * 1024 * 1024 * 1024, // 120GB (超过总量)
						DiskUsage:  150.0,                    // 150% (超过100%)
						Status:     "critical",
					},
				},
			},
			expectedValues: map[string]float64{
				"DiskUsage": 100.0, // 应修正为100%
			},
			description: "超过100%的使用率应修正为100%",
		},
		{
			name: "磁盘使用率计算不一致_需要修正",
			hostSummary: HostSummary{
				Hostname: "test-host-4",
				DiskData: []DiskInfo{
					{
						MountPoint: "/",
						DiskTotal:  100 * 1024 * 1024 * 1024, // 100GB
						DiskUsed:   75 * 1024 * 1024 * 1024,  // 75GB
						DiskUsage:  50.0,                     // 50% (与计算值75%不符，差异>5%)
						Status:     "warning",
					},
				},
			},
			expectedValues: map[string]float64{
				"DiskUsage": 75.0, // 应修正为计算值75%
			},
			description: "使用率计算不一致时应使用计算值修正",
		},
		{
			name: "多个磁盘_混合情况",
			hostSummary: HostSummary{
				Hostname: "test-host-5",
				DiskData: []DiskInfo{
					{
						MountPoint: "/",
						DiskTotal:  100 * 1024 * 1024 * 1024, // 100GB
						DiskUsed:   50 * 1024 * 1024 * 1024,  // 50GB
						DiskUsage:  50.0,                     // 50% (正常)
						Status:     "normal",
					},
					{
						MountPoint: "/var",
						DiskTotal:  200 * 1024 * 1024 * 1024, // 200GB
						DiskUsed:   -10 * 1024 * 1024 * 1024, // -10GB (异常负数)
						DiskUsage:  -5.0,                     // -5% (异常负数)
						Status:     "critical",
					},
				},
			},
			expectedValues: map[string]float64{
				"/DiskUsage":     50.0, // 第一个磁盘保持不变
				"/var/DiskUsage": 0.0,  // 第二个磁盘修正为0
				"/var/DiskUsed":  0.0,  // 第二个磁盘修正为0
			},
			description: "多个磁盘应分别验证和修正",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建hostSummary的副本进行测试
			testHost := tt.hostSummary
			
			// 执行验证函数
			validateDiskConsistency(&testHost)
			
			// 验证结果
			for mountPoint, disk := range getDiskMap(testHost.DiskData) {
				// 检查DiskUsage修正
				if expectedUsage, exists := tt.expectedValues[mountPoint+"DiskUsage"]; exists {
					if disk.DiskUsage != expectedUsage {
						t.Errorf("DiskUsage修正失败 [%s]: 期望 %.2f, 实际 %.2f", 
							mountPoint, expectedUsage, disk.DiskUsage)
					}
				}
				
				// 检查DiskUsed修正
				if expectedUsed, exists := tt.expectedValues[mountPoint+"DiskUsed"]; exists {
					if disk.DiskUsed != expectedUsed {
						t.Errorf("DiskUsed修正失败 [%s]: 期望 %.2f, 实际 %.2f", 
							mountPoint, expectedUsed, disk.DiskUsed)
					}
				}
			}
			
			// 如果只有一个磁盘，也检查不带前缀的期望值
			if len(testHost.DiskData) == 1 {
				disk := testHost.DiskData[0]
				if expectedUsage, exists := tt.expectedValues["DiskUsage"]; exists {
					if disk.DiskUsage != expectedUsage {
						t.Errorf("DiskUsage修正失败: 期望 %.2f, 实际 %.2f", expectedUsage, disk.DiskUsage)
					}
				}
				if expectedUsed, exists := tt.expectedValues["DiskUsed"]; exists {
					if disk.DiskUsed != expectedUsed {
						t.Errorf("DiskUsed修正失败: 期望 %.2f, 实际 %.2f", expectedUsed, disk.DiskUsed)
					}
				}
			}
			
			t.Logf("测试通过: %s - %s", tt.name, tt.description)
		})
	}
}

// getDiskMap 将磁盘切片转换为以挂载点为键的映射，便于测试
func getDiskMap(disks []DiskInfo) map[string]DiskInfo {
	diskMap := make(map[string]DiskInfo)
	for _, disk := range disks {
		diskMap[disk.MountPoint] = disk
	}
	return diskMap
}

// TestFormatBytes 测试字节格式化函数
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    float64
		expected string
	}{
		{
			name:     "零字节",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "小于1KB",
			bytes:    512,
			expected: "512.00 B",
		},
		{
			name:     "1KB整数",
			bytes:    1024,
			expected: "1.00 KB",
		},
		{
			name:     "1MB",
			bytes:    1024 * 1024,
			expected: "1.00 MB",
		},
		{
			name:     "1GB",
			bytes:    1024 * 1024 * 1024,
			expected: "1.00 GB",
		},
		{
			name:     "1TB",
			bytes:    1024 * 1024 * 1024 * 1024,
			expected: "1.00 TB",
		},
		{
			name:     "复杂数值",
			bytes:    1536 * 1024 * 1024, // 1.5GB
			expected: "1.50 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%f) = %s, 期望 %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

// TestFormatRate 测试速率格式化函数（针对修改后的版本）
func TestFormatRate(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		expected string
	}{
		{
			name:     "零速率",
			rate:     0,
			expected: "0 B/s",
		},
		{
			name:     "小于1024MB/s",
			rate:     512.5,
			expected: "512.50 MB/s",
		},
		{
			name:     "刚好1024MB/s",
			rate:     1024,
			expected: "1.00 GB/s",
		},
		{
			name:     "大于1024MB/s",
			rate:     1536, // 1.5GB/s
			expected: "1.50 GB/s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRate(tt.rate)
			if result != tt.expected {
				t.Errorf("formatRate(%f) = %s, 期望 %s", tt.rate, result, tt.expected)
			}
		})
	}
}