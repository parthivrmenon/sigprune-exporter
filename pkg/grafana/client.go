package grafana

import (
	"net/http"
	"time"
)

type Client struct {
	URL        string
	Username   string
	Password   string
	APIKey     string
	httpClient *http.Client
}

func NewClient(url, username, password string, timeout time.Duration) *Client {
	return &Client{
		URL:      url,
		Username: username,
		Password: password,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func NewClientWithAPIKey(url, apiKey string, timeout time.Duration) *Client {
	return &Client{
		URL:    url,
		APIKey: apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) setAuth(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	} else {
		req.SetBasicAuth(c.Username, c.Password)
	}
}
