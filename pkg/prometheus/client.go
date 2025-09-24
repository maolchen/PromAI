package prometheus

import (
	"fmt"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

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
