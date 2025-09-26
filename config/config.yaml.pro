prometheus_url: "http://10.1.114.50:8390"

project_name: "测试项目巡检报告"

# 定时任务：每天９点半和１７半执行

cron_schedule: "30 9,17 * * *"

# 报告清理

report_cleanup:
  enabled: true
  max_age: 7 # 保留最近7天的报告
  cron_schedule: "0 0 * * *" # 如果为空，则执行执行上面定时任务，即生成报告时清理

# 配置发送钉钉和邮件和企业微信通知

notifications:
  # 企微
  wecom:
    enabled: true
    webhook: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=1df4d960-b7a7-43d6-9793-59f9e460c1d8" # 企微机器人webhook
    report_url: "http://10.1.114.66:8091"
    project_title: "测试项目" #企微通知标题，用于多项目时区分项目
  dingtalk:
    enabled: false
    webhook: "https://oapi.dingtalk.com/robot/send?access_token=xxxxxxxxxxxxxxxxxxxxxxxxxxxx" # 这里填写自己的webhook
    secret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" # 这里填写的是加钉钉机器人加签的secret
    report_url: "http://10.1.114.49:8091" # 这里可以填写ip+端口，也可以填写域名,如果是k8s里部署，推荐采用域名的方式，如果不行可以将 svc 以nodeport方式暴露出来，这里就可以使用ip+端口方式
  email:
    enabled: false
    smtp_host: "smtp.exmail.qq.com" # 我这里用的是腾讯企业邮箱，需要改成自己的
    smtp_port: 465
    username: "demo@demo.cn" # 填写自己的邮箱账号
    password: "xxxxxxxxxxxxxxxxxxxx" # 这里填写的是授权码
    from: "demo@demo.cn"
    to:
    - "demo@demo.cn"
    report_url: "https://promai.lichengjun.top" # 这里可以填写ip+端口，也可以填写域名，如果是k8s里部署，推荐采用域名的方式，如果不行可以将 svc 以nodeport方式暴露出来，这里就可以使用ip+端口方式,如果是部署在k8s里，ingress 的需要自己去编写

metric_types:
- type: "基础资源使用情况"
  metrics:
  # 基础资源中的metrics 的name不能修改，不然会导致判断不到，无法渲染到主机资源概览表中（只要是希望渲染到主机资源概览表中的，就不能修改）
  - name: "CPU使用率"
    type: "monitoring" #type 有三种：monitoring: 报告中会要警告颜色区分 display：仅做数据展示，不区分颜色
    show_in_table: false # 是否在资源类型详情表中展示，false 表示不展示,true 表示展示。一般在主机资源概览表中展示的就没必要再到资源类型详情的监控中展示了，如果全都在主机资源概览表中展示，表会很丑陋，所以增加了该参数来控制展示区域
    description: "节点CPU使用率统计"
    query: "100 - (avg by(instance) (irate(node_cpu_seconds_total{mode='idle'}[5m])) * 100)"
    threshold: 80
    threshold_type: "greater"
    unit: "%"
    labels:
      instance: "节点"

  - name: "CPU核心数"
    type: "display"
    show_in_table: false
    query: "count by (instance) (node_cpu_seconds_total{mode='idle'})"
    description: "节点CPU核心数统计"
    unit: "core"
    labels:
      instance: "节点"

  - name: "内存总量"
    type: "display"
    show_in_table: false
    query: "node_memory_MemTotal_bytes"
    description: "节点内存总量统计"
    unit: "B"
    labels:
      instance: "节点"

  - name: "内存使用量"
    type: "display"
    show_in_table: false
    query: "node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes"
    description: "节点内存使用量统计"
    unit: "B"
    labels:
      instance: "节点"

  - name: "内存使用率"
    type: "monitoring"
    show_in_table: false
    description: "节点内存使用率统计"
    query: "100 - ((node_memory_MemAvailable_bytes * 100) / node_memory_MemTotal_bytes)"
    threshold: 85
    threshold_type: "greater"
    unit: "%"
    labels:
      instance: "节点"

  - name: "磁盘总量"
    type: "display"
    show_in_table: false
    query: >-
      node_filesystem_size_bytes{mountpoint!~"/run.*|/var.*|/boot.*|/tmp.*"} * on(instance) group_left(nodename) node_uname_info
    description: "节点磁盘总量统计"
    unit: "B"
    labels:
      instance: "节点"
      mountpoint: "挂载点"
      device: "磁盘"
      nodename: "节点名称"

  - name: "磁盘可用量"
    type: "display"
    show_in_table: false
    query: >-
      node_filesystem_avail_bytes{mountpoint!~"/run.*|/var.*|/boot.*|/tmp.*"} * on(instance) group_left(nodename) node_uname_info
    description: "节点磁盘可用量统计"
    unit: "B"
    labels:
      instance: "节点"
      mountpoint: "挂载点"
      device: "磁盘"
      nodename: "节点名称"

  - name: "磁盘使用率"
    type: "monitoring"
    show_in_table: false
    description: "节点磁盘使用率统计"
    query: >-
      (((100 -((node_filesystem_avail_bytes * 100) / node_filesystem_size_bytes)) and ON (instance, device, mountpoint) node_filesystem_readonly{mountpoint!~"/run.*|/var.*|/boot.*|/tmp.*"}== 0) + on(instance) group_left(node_uname_info) node_uname_info) * on(instance) group_left(nodename) node_uname_info
    threshold: 80
    threshold_type: "greater"
    unit: "%"
    labels:
      instance: "节点"
      mountpoint: "挂载点"
      device: "磁盘"
      nodename: "节点名称"

  # 新增展示类指标 - 注意：速率指标已在PromQL中进行单位换算，直接输出MB/s
  - name: "运行时间"
    type: "display"
    show_in_table: false
    description: "系统运行时长统计"
    query: "time() - node_boot_time_seconds"
    unit: "s"
    labels:
      instance: "节点"

  - name: "5分钟负载"
    type: "display"
    show_in_table: false
    description: "系统5分钟平均负载"
    query: "node_load5"
    unit: ""
    labels:
      instance: "节点"

  - name: "30分钟内磁盘平均读取值"
    type: "display"
    show_in_table: false
    description: "30分钟内磁盘平均读取速率"
    query: 'avg_over_time(rate(node_disk_read_bytes_total{device=~"vd.*|sd.*"}[5m])[30m:1m]) / 1024 / 1024'
    unit: "MB/s"
    labels:
      instance: "节点"
      device: "设备"

  - name: "30分钟内磁盘平均写入值"
    type: "display"
    show_in_table: true
    description: "30分钟内磁盘平均写入速率"
    query: 'avg_over_time(rate(node_disk_written_bytes_total{device=~"vd.*|sd.*"}[5m])[30m:1m]) / 1024 / 1024'
    unit: "MB/s"
    labels:
      instance: "节点"
      device: "设备"

  - name: "TCP连接数"
    type: "display"
    show_in_table: false
    description: "当前活跃的TCP连接总数"
    query: "node_netstat_Tcp_CurrEstab"
    unit: "个"
    labels:
      instance: "节点"

  - name: "TCP_TW数"
    type: "display"
    show_in_table: false
    description: "TCP TIME_WAIT状态连接数"
    query: "node_sockstat_TCP_tw"
    unit: "个"
    labels:
      instance: "节点"

  - name: "30分钟内下载速率"
    type: "display"
    description: "30分钟内网络平均下载速率"
    query: 'avg_over_time(rate(node_network_receive_bytes_total{device=~"eth.*|ens.*"}[5m])[30m:1m]) / 1024 / 1024'
    unit: "MB/s"
    labels:
      instance: "节点"
      device: "设备"

  - name: "30分钟内上传速率"
    type: "display"
    description: "30分钟内网络平均上传速率"
    query: 'avg_over_time(rate(node_network_transmit_bytes_total{device=~"eth.*|ens.*"}[5m])[30m:1m]) / 1024 / 1024'
    unit: "MB/s"
    labels:
      instance: "节点"
      device: "设备"
  # - name: "固定机器内存使用率"
  #   description: "固定机器内存使用率统计"
  #   query: >-
  #     100 - ((node_memory_MemAvailable_bytes{instance="172.16.5.132:9100"} * 100) / node_memory_MemTotal_bytes{instance="172.16.5.132:9100"})
  #   threshold: 16.84
  #   threshold_type: "greater"
  #   unit: "%"
  #   labels:
  #     instance: "节点"


  # - type: "PaaS平台巡检"
  #   metrics:
  #     - name: "K8s集群关键服务"
  #       description: "K8s集群关键服务状态统计"
  #       query: "key_pod_status"
  #       threshold: 1
  #       threshold_type: "equal"
  #       unit: ""
  #       labels:
  #         component: "服务名称"
  #         namespace: "命名空间"
  #         # describe: "服务描述"
  #         hostname: "主机名称"
  #         owner: "负责人"
  #         instance: "节点"

- type: "kubernetes集群监控状态"
  metrics:
  # - name: "K8s集群巡检"
  #   description: "K8s集群巡检"
  #   query: "k8s_cluster_auto_check"
  #   threshold: 1
  #   threshold_type: "equal"
  #   unit: ""
  #   labels:
  #     component: "服务名称"
  #     hostname: "主机名称"
  #     owner: "负责人"

  # - name: "自定义监控脚本执行情况"
  #   description: "script-exporter监控脚本执行情况"
  #   query: "script_success"
  #   threshold: 1
  #   threshold_type: "equal"
  #   unit: ""
  #   labels:
  #     instance: "宿主机器"
  #     script: "脚本名称"

  - name: "节点就绪状态"
    type: "monitoring"
    description: "K8s节点就绪状态检查"
    query: "kube_node_status_condition{condition='Ready',status!='true'}"
    threshold: 0
    threshold_type: "equal"
    unit: ""
    labels:
      node: "节点"
      condition: "状态类型"

  - name: "Pod运行状态"
    type: "monitoring"
    description: "集群Pod运行状态统计"
    query: "sum by (namespace, pod) (kube_pod_status_phase{phase='Running'})"
    threshold: 1
    threshold_type: "equal"
    unit: ""
    labels:
      namespace: "命名空间"
      pod: "Pod名称"

  - name: "PVC使用率"
    type: "monitoring"
    description: "持久化存储使用率"
    query: >-
      100 * (1 - kubelet_volume_stats_available_bytes / kubelet_volume_stats_capacity_bytes)
    threshold: 90
    threshold_type: "greater"
    unit: "%"
    labels:
      namespace: "命名空间"
      persistentvolumeclaim: "PVC名称"

- type: "进程指标"
  metrics:
  - name: "进程CPU使用率top5"
    type: "display"
    description: "进程CPU使用率top5"
    query: "topk(5,rate(namedprocess_namegroup_cpu_seconds_total{}[5m]) or irate(namedprocess_namegroup_cpu_seconds_total{}[5m]))"
    unit: "%"
    labels:
      instance: "节点"
      groupname: "进程名"
  - name: "进程内存使用率top5"
    type: "display"
    description: "进程内存使用率top5"
    query: 'topk(5,(avg_over_time(namedprocess_namegroup_memory_bytes{memtype="swapped"}[5m])+ ignoring (memtype) avg_over_time(namedprocess_namegroup_memory_bytes{memtype="resident"}[5m])) / (1024 * 1204))'
    unit: "MB"
    labels:
      instance: "节点"
      groupname: "进程名"

- type: "其他指标"
  metrics:
  - name: "域名证书有效期小于30天"
    type: "monitoring"
    description: "域名证书有效期检查"
    query: "round((probe_ssl_earliest_cert_expiry - time()) / 86400)"
    threshold: 60
    threshold_type: "at_least"
    unit: "天"
    labels:
      instance: "节点"
      target: "域名"
