package grafana

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// getJSON sends an authenticated GET to path and decodes a 200 OK JSON body into
// out, which must be a pointer.
// path is a bare API path with no query string of its own ("/api/search", not "/api/search?type=dash-db")
// query parameters belong in params which may be nil
// Any non-200 response is returned as an error carrying the status and body.
func (c *Client) getJSON(path string, params url.Values, out any) error {
	u := c.URL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("request failed with status %d, and failed to read body: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("API error (Status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return err
	}
	return nil

}
