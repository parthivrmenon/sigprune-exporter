package grafana

import (
	"encoding/json"
	"net/http"
)

type Dashboard struct {
	ID    int    `json:"id"`
	UID   string `json:"uid"`
	Title string `json:"title"`
}

type DashboardPanelExpr struct {
	Expr string `json:"expr"`
}

type DashboardPanelTarget struct {
	Expr  string `json:"expr"`
	RefID string `json:"refId"`
}

type DashboardPanel struct {
	ID      int                    `json:"id"`
	UID     string                 `json:"uid"`
	Title   string                 `json:"title"`
	Targets []DashboardPanelTarget `json:"targets"`
	// Panels holds the children of a collapsed row. It is empty for expanded
	// rows, whose children sit at the top level instead.
	Panels []DashboardPanel `json:"panels"`
}

type DashboardResponse struct {
	Meta struct {
		Type string `json:"type"`
	} `json:"meta"`
	Dashboard struct {
		ID     int              `json:"id"`
		UID    string           `json:"uid"`
		Title  string           `json:"title"`
		Panels []DashboardPanel `json:"panels"`
	} `json:"dashboard"`
}

type Datasource struct {
	ID     int    `json:"id"`
	UID    string `json:"uid"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Access string `json:"access"`
}

func (c *Client) GetDashboards() ([]Dashboard, error) {
	req, err := http.NewRequest("GET", c.URL+"/api/search?type=dash-db", nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dashboards []Dashboard
	err = json.NewDecoder(resp.Body).Decode(&dashboards)
	if err != nil {
		return nil, err
	}

	return dashboards, nil
}

func (c *Client) GetDashboardByUID(uid string) (*DashboardResponse, error) {
	req, err := http.NewRequest("GET", c.URL+"/api/dashboards/uid/"+uid, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dashboardResponse DashboardResponse
	err = json.NewDecoder(resp.Body).Decode(&dashboardResponse)
	if err != nil {
		return nil, err
	}

	// bodyBytes, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return nil, err
	// }

	// utils.PrintAsJSON(bodyBytes)

	return &dashboardResponse, nil
}

func GetDashboardPanelExprs(d DashboardResponse) []string {
	return collectPanelExprs(d.Dashboard.Panels, nil)
}

// collectPanelExprs appends the target exprs of every panel, including panels
// nested inside collapsed rows.
func collectPanelExprs(panels []DashboardPanel, exprs []string) []string {
	for _, panel := range panels {
		for _, target := range panel.Targets {
			exprs = append(exprs, target.Expr)
		}
		exprs = collectPanelExprs(panel.Panels, exprs)
	}
	return exprs
}

func (c *Client) GetDatasources() ([]Datasource, error) {
	req, err := http.NewRequest("GET", c.URL+"/api/datasources", nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var datasources []Datasource
	err = json.NewDecoder(resp.Body).Decode(&datasources)
	if err != nil {
		return nil, err
	}

	return datasources, nil
}
