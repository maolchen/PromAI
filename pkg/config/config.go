package config

import "PromAI/pkg/notify"

type Config struct {
	PrometheusURL string       `yaml:"prometheus_url"`
	MetricTypes   []MetricType `yaml:"metric_types"`
	ProjectName   string       `yaml:"project_name"`
	CronSchedule  string       `yaml:"cron_schedule"`
	ReportCleanup struct {
		Enabled      bool   `yaml:"enabled"`
		MaxAge       int    `yaml:"max_age"`
		CronSchedule string `yaml:"cron_schedule"`
	} `yaml:"report_cleanup"`
	Notifications struct {
		Dingtalk notify.DingtalkConfig `yaml:"dingtalk"`
		Email    notify.EmailConfig    `yaml:"email"`
		Wecom    notify.WeComConfig    `yaml:"wecom"`
	} `yaml:"notifications"`
	Port string `yaml:"port"`
}

type MetricType struct {
	Type    string         `yaml:"type"`
	Metrics []MetricConfig `yaml:"metrics"`
}

type MetricConfig struct {
	Name          string            `yaml:"name"`
	Type          string            `yaml:"type"`
	Description   string            `yaml:"description"`
	Query         string            `yaml:"query"`
	Threshold     float64           `yaml:"threshold"`
	Unit          string            `yaml:"unit"`
	Labels        map[string]string `yaml:"labels"`
	ThresholdType string            `yaml:"threshold_type"`
}
