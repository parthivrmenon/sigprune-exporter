package grafana

import (
	"net/http"
	"time"
)

type Client struct {
	URL        string
	Username   string
	Password   string
	httpClient *http.Client
}

func NewClient(url, username, password string) *Client {
	return &Client{
		URL:      url,
		Username: username,
		Password: password,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
