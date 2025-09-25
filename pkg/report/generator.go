package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

type LabelData struct {
	Name  string // 原始标签名
	Alias string // 显示的别名
	Value string // 标签值
}
type GroupStats struct {
	MaxValue      float64
	MinValue      float64
	Average       float64
	AlertCount    int // 告警数量
	CriticalCount int // 严重告警数量
	WarningCount  int // 警告数量
	TotalCount    int // 总指标数
}
type MetricData struct {
	Instance    string
	Name        string
	Description string
	Value       float64
	Threshold   float64
	Unit        string
	Status      string
	StatusText  string
	Timestamp   time.Time
	Labels      []LabelData // 改用结构化的标签数据
}

type MetricGroup struct {
	Type          string
	MetricsByName map[string][]MetricData
	MetricOrder   []string
	Stats         GroupStats // 替换原来的 Average
}

// 新增：主机资源聚合结构
type DiskInfo struct {
	MountPoint string
	DiskTotal  float64
	DiskUsed   float64
	DiskUsage  float64
	Status     string // 新增：磁盘使用率状态
}

// 新增：磁盘IO信息结构
type DiskIOInfo struct {
	Device         string  // 磁盘设备名
	AvgReadRate    float64 // 30分钟平均读取速率
	AvgWriteRate   float64 // 30分钟平均写入速率
}

// 新增：网络IO信息结构
type NetworkIOInfo struct {
	Interface      string  // 网络接口名
	AvgDownloadRate float64 // 30分钟平均下载速率
	AvgUploadRate   float64 // 30分钟平均上传速率
}

type HostSummary struct {
	Hostname  string
	IP        string
	CPUCount  int64
	CPUUsage  float64
	CPUStatus string  // 新增：CPU使用率状态
	MemTotal  float64
	MemUsed   float64
	MemUsage  float64
	MemStatus string  // 新增：内存使用率状态
	DiskData  []DiskInfo
	Timestamp time.Time
	
	// 新增字段
	Uptime          float64            // 运行时间(秒)
	Load5           float64            // 5分钟负载
	DiskIOStats     []DiskIOInfo       // 磁盘IO统计
	TCPConnections  int64              // TCP连接数
	TCPTimeWait     int64              // TCP TIME_WAIT连接数
	NetworkStats    []NetworkIOInfo    // 网络IO统计
}

type ReportData struct {
	Timestamp    time.Time
	MetricGroups map[string]*MetricGroup
	GroupOrder   []string
	ChartData    map[string]template.JS
	Project      string
	HostSummary  []HostSummary // 新增：主机资源汇总
}

func GetStatusText(status string) string {
	switch status {
	case "critical":
		return "严重"
	case "warning":
		return "警告"
	default:
		return "正常"
	}
}

// 新增：字节格式化函数
func formatBytes(bytes float64) string {
	if bytes == 0 {
		return "0 B"
	}

	unitPrefixes := []string{"B", "KB", "MB", "GB", "TB"}
	const unitSize = 1024

	unitIndex := 0
	floatBytes := float64(bytes)

	for floatBytes >= unitSize && unitIndex < len(unitPrefixes)-1 {
		floatBytes /= unitSize
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", floatBytes, unitPrefixes[unitIndex])
}

// 新增：运行时间格式化函数
func formatUptime(seconds float64) string {
	if seconds <= 0 {
		return "0秒"
	}

	days := int(seconds) / 86400
	hours := (int(seconds) % 86400) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60

	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%d天%d小时%d分钟", days, hours, minutes)
		}
		return fmt.Sprintf("%d天%d分钟", days, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%d分钟%d秒", minutes, secs)
	}
	return fmt.Sprintf("%d秒", secs)
}

// 新增：速率格式化函数（用于网络和磁盘IO）
func formatRate(bytesPerSecond float64) string {
	if bytesPerSecond == 0 {
		return "0 B/s"
	}

	unitPrefixes := []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s"}
	const unitSize = 1024

	unitIndex := 0
	floatRate := float64(bytesPerSecond)

	for floatRate >= unitSize && unitIndex < len(unitPrefixes)-1 {
		floatRate /= unitSize
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", floatRate, unitPrefixes[unitIndex])
}

// 新增：数值格式化函数（用于连接数等）
func formatNumber(number int64) string {
	if number == 0 {
		return "0"
	}
	
	// 添加千位分隔符
	str := fmt.Sprintf("%d", number)
	if len(str) <= 3 {
		return str
	}
	
	var result []rune
	for i, r := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, r)
	}
	
	return string(result)
}

// 新增：提取IP地址函数，从instance:9100 提取
func extractIP(instance string) string {
	if idx := strings.LastIndex(instance, ":"); idx != -1 {
		return instance[:idx]
	}
	return instance
}

func GenerateReport(data ReportData) (string, error) {
	log.Printf("GroupOrder: %+v", data.GroupOrder)
	for groupType, group := range data.MetricGroups {
		log.Printf("Group [%s] MetricOrder: %+v", groupType, group.MetricOrder) // ✅ 这里应该不报错且有值
	}
	// 计算每个组的统计信息
	for _, group := range data.MetricGroups {
		stats := GroupStats{}

		for _, metrics := range group.MetricsByName {
			for _, metric := range metrics {
				log.Printf("Metric: %s, Status: %s, Value: %f, Threshold: %f, Unit: %s", metric.Name, metric.Status, metric.Value, metric.Threshold, metric.Unit)
				stats.TotalCount++

				// 统计告警数量 - 优化后逻辑：严重和警告不重复计算
				switch metric.Status {
				case "warning":
					stats.WarningCount++
				case "critical":
					stats.CriticalCount++
				}
			}
		}

		// 告警总数 = 严重告警数 + 警告告警数
		stats.AlertCount = stats.CriticalCount + stats.WarningCount
		group.Stats = stats
	}

	// 处理图表数据
	allLabels := make(map[string]bool)      // 用于存储所有唯一的标签值
	chartData := make(map[string][]float64) // 用于存储图表数据
	// 收集所有唯一的标签值和准备图表数据
	labelValuesByMetric := make(map[string]map[string]bool) // 按指标存储唯一标签值

	// 第一次遍历收集每个指标的唯一标签值
	for _, group := range data.MetricGroups {
		for metricName, metrics := range group.MetricsByName {
			metricKey := fmt.Sprintf("%s_%s", group.Type, metricName)
			labelValuesByMetric[metricKey] = make(map[string]bool)
			// log.Println("指标组：", group.Type, "指标：", metricName, "指标键：", metricKey)
			for _, metric := range metrics {
				for _, label := range metric.Labels {
					labelValuesByMetric[metricKey][label.Value] = true
					// log.Println("指标组：", group.Type, "指标：", metricName, "指标键：", metricKey, "标签值：", label.Value)
					allLabels[label.Value] = true

				}
			}
		}
	}

	// 第二次遍历按标签值顺序生成图表数据
	for _, group := range data.MetricGroups {
		for metricName, metrics := range group.MetricsByName {
			metricKey := fmt.Sprintf("%s_%s", group.Type, metricName)
			metricValues := make(map[string]float64)
			// log.Println("指标类型：", group.Type, "指标名称：", metricName, "指标Key：", metricKey)

			// 初始化所有标签值对应的指标值为0
			for labelValue := range labelValuesByMetric[metricKey] {

				metricValues[labelValue] = 0

				log.Println("标签值：", labelValue, "指标值：", metricValues[labelValue])
			}

			// 填充实际的指标值
			for _, metric := range metrics {
				if len(metric.Labels) > 0 {
					metricValues[metric.Labels[0].Value] = metric.Value
				}
			}

			// 按标签值顺序添加到图表数据
			chartData[metricKey] = make([]float64, 0)
			for labelValue := range labelValuesByMetric[metricKey] {
				chartData[metricKey] = append(chartData[metricKey], metricValues[labelValue])
			}
			// log.Println("图表数据：", metricKey, "图表数据值：", chartData[metricKey])
		}
	}

	// 转换标签为数组并排序
	labels := make([]string, 0, len(allLabels))
	for label := range allLabels {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	// 转换为JSON
	labelsJSON, _ := json.Marshal(labels)
	data.ChartData["labels"] = template.JS(labelsJSON)
	// log.Println("标签：", labels)
	// 为每个指标生成图表数据
	for key, values := range chartData {
		valuesJSON, _ := json.Marshal(values)
		data.ChartData[key] = template.JS(valuesJSON)
	}

	// 按主机聚合数据
	hostMap := make(map[string]*HostSummary)
	for _, group := range data.MetricGroups {
		for metricName, metrics := range group.MetricsByName {
			for _, m := range metrics {
				var instance string
				for _, label := range m.Labels {
					if label.Name == "instance" {
						instance = label.Value
						break
					}
				}
				if instance == "" {
					continue
				}

				if _, exists := hostMap[instance]; !exists {
					hostMap[instance] = &HostSummary{
						Hostname:     instance,
						IP:           extractIP(instance),
						DiskData:     make([]DiskInfo, 0),
						DiskIOStats:  make([]DiskIOInfo, 0),
						NetworkStats: make([]NetworkIOInfo, 0),
						Timestamp:    m.Timestamp,
					}
				}

				host := hostMap[instance]

				// 更新最新时间戳
				if m.Timestamp.After(host.Timestamp) {
					host.Timestamp = m.Timestamp
				}

				log.Printf("Processing metric: %s from instance %s, value: %f, status: %s", metricName, instance, m.Value, m.Status)
				// 根据指标名填充数据
				switch metricName {
				case "CPU使用率":
					host.CPUUsage = m.Value
					host.CPUStatus = m.Status // 传递状态
				case "CPU核心数":
					host.CPUCount = int64(m.Value)
				case "内存总量":
					host.MemTotal = m.Value
				case "内存使用量":
					host.MemUsed = m.Value
					if host.MemTotal > 0 {
						host.MemUsage = (host.MemUsed / host.MemTotal) * 100
					}
				case "内存使用率":
					host.MemUsage = m.Value
					host.MemStatus = m.Status // 传递状态
				// 新增：处理运行时间指标
				case "运行时间":
					host.Uptime = m.Value
				// 新增：处理5分钟负载指标
				case "5分钟负载":
					host.Load5 = m.Value
				// 新增：处理TCP连接数指标
				case "TCP连接数":
					host.TCPConnections = int64(m.Value)
				// 新增：处理TCP_TW数指标
				case "TCP_TW数":
					host.TCPTimeWait = int64(m.Value)
				// 新增：处理磁盘IO指标
				case "30分钟内磁盘平均读取值", "30分钟内磁盘平均写入值":
					var device string
					for _, label := range m.Labels {
						if label.Name == "device" {
							device = label.Value
							break
						}
					}
					if device == "" {
						continue
					}

					var diskIO *DiskIOInfo
					for i := range host.DiskIOStats {
						if host.DiskIOStats[i].Device == device {
							diskIO = &host.DiskIOStats[i]
							break
						}
					}
					if diskIO == nil {
						host.DiskIOStats = append(host.DiskIOStats, DiskIOInfo{Device: device})
						diskIO = &host.DiskIOStats[len(host.DiskIOStats)-1]
					}

					if metricName == "30分钟内磁盘平均读取值" {
						diskIO.AvgReadRate = m.Value
					} else if metricName == "30分钟内磁盘平均写入值" {
						diskIO.AvgWriteRate = m.Value
					}
				// 新增：处理网络IO指标
				case "30分钟内下载速率", "30分钟内上传速率":
					var networkDevice string
					for _, label := range m.Labels {
						if label.Name == "device" {
							networkDevice = label.Value
							break
						}
					}
					if networkDevice == "" {
						continue
					}

					var networkIO *NetworkIOInfo
					for i := range host.NetworkStats {
						if host.NetworkStats[i].Interface == networkDevice {
							networkIO = &host.NetworkStats[i]
							break
						}
					}
					if networkIO == nil {
						host.NetworkStats = append(host.NetworkStats, NetworkIOInfo{Interface: networkDevice})
						networkIO = &host.NetworkStats[len(host.NetworkStats)-1]
					}

					if metricName == "30分钟内下载速率" {
						networkIO.AvgDownloadRate = m.Value
					} else if metricName == "30分钟内上传速率" {
						networkIO.AvgUploadRate = m.Value
					}
				case "磁盘总量", "磁盘可用量":
					var mountPoint string
					for _, label := range m.Labels {
						if label.Name == "mountpoint" {
							mountPoint = label.Value
							break
						}
					}
					if mountPoint == "" {
						continue
					}

					var disk *DiskInfo
					for i := range host.DiskData {
						if host.DiskData[i].MountPoint == mountPoint {
							disk = &host.DiskData[i]
							break
						}
					}
					if disk == nil {
						host.DiskData = append(host.DiskData, DiskInfo{MountPoint: mountPoint})
						disk = &host.DiskData[len(host.DiskData)-1]
					}

					if metricName == "磁盘总量" {
						disk.DiskTotal = m.Value
					} else if metricName == "磁盘可用量" {
						disk.DiskUsed = disk.DiskTotal - m.Value
						if disk.DiskTotal > 0 {
							disk.DiskUsage = (disk.DiskUsed / disk.DiskTotal) * 100
						}
					}
				case "磁盘使用率":
					// 处理磁盘使用率指标
					var mountPoint string
					for _, label := range m.Labels {
						if label.Name == "mountpoint" {
							mountPoint = label.Value
							break
						}
					}
					if mountPoint == "" {
						continue
					}

					var disk *DiskInfo
					for i := range host.DiskData {
						if host.DiskData[i].MountPoint == mountPoint {
							disk = &host.DiskData[i]
							break
						}
					}
					if disk == nil {
						host.DiskData = append(host.DiskData, DiskInfo{MountPoint: mountPoint})
						disk = &host.DiskData[len(host.DiskData)-1]
					}

					// 传递磁盘使用率和状态
					disk.DiskUsage = m.Value
					disk.Status = m.Status // 传递状态
				}
			}
		}
	}

	// 转换为切片
	data.HostSummary = make([]HostSummary, 0, len(hostMap))
	for _, h := range hostMap {
		data.HostSummary = append(data.HostSummary, *h)
	}

	// // ✅ 注册模板函数
	funcMap := template.FuncMap{
		"formatBytes":  formatBytes,
		"formatUptime": formatUptime,
		"formatRate":   formatRate,
		"formatNumber": formatNumber,
	}

	tmpl, err := template.New("report.html").Funcs(funcMap).ParseFiles("templates/report.html")
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	// // 生成报告
	// tmpl, err := template.ParseFiles("templates/report.html")
	// if err != nil {
	// 	return "", fmt.Errorf("parsing template: %w", err)
	// }

	// 创建输出文件
	filename := fmt.Sprintf("reports/inspection_report_%s.html", time.Now().Format("20060102_150405"))
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("creating output file: %w", err)
	}
	defer file.Close()

	// 执行模板
	if err := tmpl.Execute(file, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	// log.Println("Report generated successfully:", filename)
	for _, h := range data.HostSummary {
		log.Printf("Host: %s, CPUCount: %d, MemTotal: %f, MemUsed: %f, DiskData: %d",
			h.Hostname, h.CPUCount, h.MemTotal, h.MemUsed, len(h.DiskData))
	}
	log.Printf("项目[%s]报告生成成功: %s", data.Project, filename)

	return filename, nil // 添加返回语句
}
