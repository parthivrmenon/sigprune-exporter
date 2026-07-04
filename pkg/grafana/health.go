package grafana

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type HealthResponse struct {
	Status  string `json:"database"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func (c *Client) TestConnection() error {
	fmt.Println("Testing connection to Grafana at", c.URL)

	req, err := http.NewRequest("GET", c.URL+"/api/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var health HealthResponse
	err = json.NewDecoder(resp.Body).Decode(&health)
	if err != nil {
		return err
	}
	fmt.Println("Connection successful")
	fmt.Printf("status: %s, version: %s, commit: %s\n", health.Status, health.Version, health.Commit)
	return nil
}
