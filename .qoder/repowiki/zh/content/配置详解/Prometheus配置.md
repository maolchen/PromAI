# Prometheus配置技术文档

<cite>
**本文档引用的文件**
- [config.yaml](file://config/config.yaml)
- [config.go](file://pkg/config/config.go)
- [client.go](file://pkg/prometheus/client.go)
- [prometheus.go](file://pkg/prometheus/prometheus.go)
- [main.go](file://main.go)
- [collector.go](file://pkg/metrics/collector.go)
- [status.go](file://pkg/status/status.go)
- [generator.go](file://pkg/report/generator.go)
- [utils.go](file://pkg/utils/utils.go)
- [cleanup.go](file://pkg/report/cleanup.go)
</cite>

## 目录
1. [简介](#简介)
2. [配置文件结构](#配置文件结构)
3. [PrometheusURL字段详解](#prometheusurl字段详解)
4. [配置加载机制](#配置加载机制)
5. [客户端初始化流程](#客户端初始化流程)
6. [错误处理机制](#错误处理机制)
7. [配置示例](#配置示例)
8. [最佳实践](#最佳实践)
9. [故障排除指南](#故障排除指南)
10. [总结](#总结)

## 简介

PromAI是一个基于Prometheus的监控指标巡检系统，通过配置文件中的`prometheus_url`字段来指定Prometheus服务的访问地址。该字段是整个系统的核心配置项，直接影响所有指标查询操作的执行。

本文档将深入解析Prometheus配置的工作原理，包括配置文件结构、字段映射关系、客户端初始化过程以及错误处理机制，帮助用户正确配置和使用Prometheus服务。

## 配置文件结构

### YAML配置文件架构

```mermaid
graph TD
A[config.yaml] --> B[prometheus_url]
A --> C[project_name]
A --> D[cron_schedule]
A --> E[report_cleanup]
A --> F[notifications]
A --> G[metric_types]
E --> E1[enabled]
E --> E2[max_age]
E --> E3[cron_schedule]
F --> F1[dingtalk]
F --> F2[email]
F --> F3[wecom]
F1 --> F11[enabled]
F1 --> F12[webhook]
F1 --> F13[secret]
F1 --> F14[report_url]
F2 --> F21[enabled]
F2 --> F22[smtp_host]
F2 --> F23[smtp_port]
F2 --> F24[username]
F2 --> F25[password]
F2 --> F26[to]
F2 --> F27[report_url]
F3 --> F31[enabled]
F3 --> F32[webhook]
F3 --> F33[report_url]
F3 --> F34[project_title]
G --> G1[type]
G --> G2[metrics]
G2 --> G21[name]
G2 --> G22[description]
G2 --> G23[query]
G2 --> G24[threshold]
G2 --> G25[unit]
G2 --> G26[labels]
G2 --> G27[threshold_type]
```

**图表来源**
- [config.yaml](file://config/config.yaml#L1-L196)
- [config.go](file://pkg/config/config.go#L1-L37)

### Go结构体映射

配置文件中的`prometheus_url`字段通过YAML标签映射到Go结构体的`PrometheusURL`字段：

```go
type Config struct {
    PrometheusURL string       `yaml:"prometheus_url"`
    // 其他字段...
}
```

**章节来源**
- [config.go](file://pkg/config/config.go#L5-L10)
- [config.yaml](file://config/config.yaml#L1-L1)

## PrometheusURL字段详解

### 字段作用机制

`prometheus_url`字段在系统中的工作流程如下：

```mermaid
sequenceDiagram
participant Config as 配置文件
participant Loader as 配置加载器
participant Env as 环境变量
participant Struct as Config结构体
participant Client as Prometheus客户端
Config->>Loader : 读取prometheus_url
Loader->>Env : 检查PROMETHEUS_URL环境变量
alt 环境变量存在
Env->>Struct : 使用环境变量值
else 环境变量不存在
Config->>Struct : 使用配置文件值
end
Struct->>Client : 初始化客户端连接
Client->>Client : 建立HTTP连接
```

**图表来源**
- [main.go](file://main.go#L28-L40)
- [config.go](file://pkg/config/config.go#L5-L10)

### 字段验证与处理

系统对`prometheus_url`字段进行了严格的验证和处理：

1. **格式验证**：确保URL格式正确
2. **协议支持**：支持HTTP和HTTPS协议
3. **环境变量优先级**：环境变量覆盖配置文件值
4. **默认值处理**：如果未配置则使用配置文件中的值

**章节来源**
- [main.go](file://main.go#L28-L40)

## 配置加载机制

### 加载流程

```mermaid
flowchart TD
Start([开始加载配置]) --> ReadFile["读取config.yaml文件"]
ReadFile --> ParseYAML["解析YAML内容"]
ParseYAML --> LoadEnv["检查环境变量"]
LoadEnv --> CheckEnv{"PROMETHEUS_URL<br/>环境变量存在?"}
CheckEnv --> |是| UseEnv["使用环境变量值"]
CheckEnv --> |否| UseConfig["使用配置文件值"]
UseEnv --> CreateStruct["创建Config结构体"]
UseConfig --> CreateStruct
CreateStruct --> ValidateURL["验证URL格式"]
ValidateURL --> Success["配置加载成功"]
ValidateURL --> Error["配置加载失败"]
Error --> End([结束])
Success --> End
```

**图表来源**
- [main.go](file://main.go#L28-L40)

### 环境变量优先级

系统实现了环境变量优先于配置文件的机制：

```go
// 从环境变量中获取 PrometheusURL
if envPrometheusURL := os.Getenv("PROMETHEUS_URL"); envPrometheusURL != "" {
    log.Printf("使用环境变量中的 Prometheus URL: %s", envPrometheusURL)
    config.PrometheusURL = envPrometheusURL
} else {
    log.Printf("使用配置文件中的 Prometheus URL: %s", config.PrometheusURL)
}
```

这种设计允许：
- 生产环境使用环境变量配置
- 开发环境使用配置文件
- 动态调整配置而无需重启应用

**章节来源**
- [main.go](file://main.go#L32-L40)

## 客户端初始化流程

### 客户端创建过程

```mermaid
sequenceDiagram
participant Setup as setup函数
participant Config as 配置加载
participant Client as Prometheus客户端
participant API as Prometheus API
Setup->>Config : 加载配置文件
Config->>Setup : 返回Config结构体
Setup->>Client : 调用NewClient(PrometheusURL)
Client->>Client : 创建api.Config
Client->>API : 初始化v1.API
API->>Client : 返回API实例
Client->>Setup : 返回Client对象
Setup->>Setup : 返回client和config
```

**图表来源**
- [main.go](file://main.go#L42-L50)
- [client.go](file://pkg/prometheus/client.go#L15-L27)

### 客户端结构设计

```go
// Client 封装 Prometheus 客户端
type Client struct {
    API v1.API
}

// NewClient 创建新的 Prometheus 客户端
func NewClient(url string) (*Client, error) {
    client, err := api.NewClient(api.Config{
        Address: url,
    })
    if err != nil {
        return nil, fmt.Errorf("creating prometheus client: %w", err)
    }

    return &Client{
        API: v1.NewAPI(client),
    }, nil
}
```

**章节来源**
- [client.go](file://pkg/prometheus/client.go#L8-L27)

## 错误处理机制

### 连接失败处理

当Prometheus服务不可达时，系统提供了多层次的错误处理：

```mermaid
flowchart TD
InitClient[初始化客户端] --> ConnectAttempt[尝试建立连接]
ConnectAttempt --> CheckError{"连接是否成功?"}
CheckError --> |成功| Success[客户端初始化成功]
CheckError --> |失败| LogError[记录错误日志]
LogError --> ReturnError[返回错误信息]
ReturnError --> FatalExit[程序终止]
Success --> NormalOperation[正常操作]
NormalOperation --> QueryExecution[执行查询]
QueryExecution --> CheckQueryError{"查询是否成功?"}
CheckQueryError --> |成功| ProcessResult[处理结果]
CheckQueryError --> |失败| LogQueryError[记录查询错误]
LogQueryError --> ContinueExecution[继续执行]
```

**图表来源**
- [main.go](file://main.go#L42-L50)
- [collector.go](file://pkg/metrics/collector.go#L45-L55)

### 查询错误处理

在指标收集过程中，系统对每个查询都进行了错误处理：

```go
result, _, err := c.Client.Query(ctx, metric.Query, time.Now())
if err != nil {
    log.Printf("警告: 查询指标 %s 失败: %v", metric.Name, err)
    continue
}
```

这种设计确保：
- 单个指标查询失败不会影响其他指标
- 错误信息会被记录但不影响整体流程
- 系统能够继续处理其他有效指标

**章节来源**
- [collector.go](file://pkg/metrics/collector.go#L45-L55)

## 配置示例

### 基础配置示例

```yaml
# 基础HTTP配置
prometheus_url: "http://10.1.114.50:8390"

# HTTPS配置
prometheus_url: "https://prometheus.example.com:9090"

# 带认证的配置
prometheus_url: "http://admin:password@prometheus.internal:8390"

# Kubernetes集群配置
prometheus_url: "http://prometheus-service.monitoring.svc.cluster.local:9090"
```

### 高级配置示例

```yaml
# 生产环境配置
prometheus_url: "https://prometheus.prod.company.com:443"

# 多环境配置示例
# 开发环境
prometheus_url: "http://localhost:9090"
# 测试环境  
prometheus_url: "http://test-prometheus:9090"
# 生产环境
prometheus_url: "https://prometheus.prod.company.com:443"

# 带路径的配置
prometheus_url: "http://prometheus.internal:9090/prometheus"
```

### 环境变量配置

```bash
# Linux/macOS
export PROMETHEUS_URL="https://prometheus.prod.company.com:443"

# Windows
set PROMETHEUS_URL=https://prometheus.prod.company.com:443
```

## 最佳实践

### 网络配置建议

1. **内网优先**：在内网环境中优先使用内网IP地址
2. **负载均衡**：对于高可用部署，使用负载均衡器地址
3. **SSL/TLS**：生产环境建议使用HTTPS连接
4. **防火墙配置**：确保PromAI服务器能够访问Prometheus端口

### 权限配置建议

1. **只读权限**：PromAI只需要查询权限，不需要写入权限
2. **认证配置**：如果Prometheus启用了身份认证，确保配置正确的凭据
3. **网络隔离**：在DMZ环境中部署PromAI，限制其访问范围

### 监控配置建议

```yaml
# 推荐的监控配置
prometheus_url: "https://prometheus.internal:9090"

# 配置文件
project_name: "生产环境巡检报告"
cron_schedule: "30 9,17 * * *"  # 每天9点半和17点半执行

# 报告清理配置
report_cleanup:
  enabled: true
  max_age: 7  # 保留最近7天的报告
  cron_schedule: "0 0 * * *"  # 每天凌晨执行清理
```

## 故障排除指南

### 常见问题及解决方案

#### 1. 连接超时问题

**症状**：程序启动时提示连接超时
**原因**：网络不通或Prometheus服务未启动
**解决方案**：
```bash
# 检查网络连通性
ping prometheus.internal
telnet prometheus.internal 9090

# 检查Prometheus服务状态
curl http://prometheus.internal:9090/api/v1/query?query=up
```

#### 2. SSL证书问题

**症状**：HTTPS连接失败，证书验证错误
**解决方案**：
```yaml
# 使用insecure_skip_verify（仅用于开发环境）
prometheus_url: "https://prometheus.internal:9090"
```

#### 3. 认证失败问题

**症状**：401 Unauthorized错误
**解决方案**：
```yaml
# 确保URL包含正确的认证信息
prometheus_url: "http://username:password@prometheus.internal:9090"
```

#### 4. 查询超时问题

**症状**：指标查询长时间无响应
**解决方案**：
- 检查Prometheus查询性能
- 优化PromQL查询语句
- 增加查询超时时间

### 调试技巧

1. **启用详细日志**：
```bash
# 设置日志级别
export LOG_LEVEL=debug
```

2. **直接测试连接**：
```bash
# 使用curl测试Prometheus API
curl -G \
  --data-urlencode 'query=up' \
  http://prometheus.internal:9090/api/v1/query
```

3. **检查配置加载**：
```bash
# 启动时查看配置加载日志
./promai -config=config/config.yaml
```

**章节来源**
- [main.go](file://main.go#L42-L50)
- [collector.go](file://pkg/metrics/collector.go#L45-L55)

## 总结

Prometheus配置是PromAI系统的核心组件，通过`prometheus_url`字段实现了灵活的服务发现和连接管理。系统提供了以下关键特性：

1. **双源配置**：支持配置文件和环境变量双重配置源
2. **错误容错**：完善的错误处理和日志记录机制
3. **灵活适配**：支持多种网络环境和认证方式
4. **易于维护**：清晰的配置结构和详细的错误信息

正确配置PrometheusURL字段是系统正常运行的基础，建议用户根据实际环境选择合适的配置方案，并定期检查连接状态以确保系统稳定性。

通过遵循本文档提供的最佳实践和故障排除指南，用户可以有效地部署和维护PromAI系统，实现高效的监控指标巡检功能。