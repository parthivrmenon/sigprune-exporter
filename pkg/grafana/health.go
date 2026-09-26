package grafana

import (
	"fmt"
)

type HealthResponse struct {
	Status  string `json:"database"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func (c *Client) TestConnection() error {
	fmt.Println("Testing connection to Grafana at", c.URL)
	var health HealthResponse
	err := c.getJSON("/api/health", nil, &health)
	if err != nil {
		return err
	}
	fmt.Println("Connection successful")
	fmt.Printf("status: %s, version: %s, commit: %s\n", health.Status, health.Version, health.Commit)
	return nil
}
