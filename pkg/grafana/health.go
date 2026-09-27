package grafana

import (
	"fmt"
	"log"
	"net/url"
)

type HealthResponse struct {
	Status  string `json:"database"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func (c *Client) TestConnection(datasourceUID string) error {
	log.Println("Testing connection to Grafana at", c.URL)
	var health HealthResponse
	err := c.getJSON("/api/health", nil, &health)
	if err != nil {
		return err
	}
	var datasource Datasource
	dsURI := "/api/datasources/uid/" + url.PathEscape(datasourceUID)
	err = c.getJSON(dsURI, nil, &datasource)
	if err != nil {
		return fmt.Errorf("fetch datasource %q: %w", datasourceUID, err)
	}
	if datasource.Type != "prometheus" {
		return fmt.Errorf("datasource %q is type %q, not prometheus", datasource.Name, datasource.Type)
	}

	log.Println("Connection successful")
	log.Printf("status: %s, version: %s, commit: %s", health.Status, health.Version, health.Commit)
	log.Printf("datasource: %q (uid %s, type %s)", datasource.Name, datasource.UID, datasource.Type)

	return nil
}
