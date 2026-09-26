package grafana

import (
	"net/url"
)

type Dashboard struct {
	ID    int    `json:"id"`
	UID   string `json:"uid"`
	Title string `json:"title"`
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
	var d []Dashboard
	err := c.getJSON("/api/search", url.Values{"type": {"dash-db"}}, &d)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (c *Client) GetDashboardByUID(uid string) (*DashboardResponse, error) {
	var dashboardResponse DashboardResponse
	err := c.getJSON("/api/dashboards/uid/"+uid, nil, &dashboardResponse)
	if err != nil {
		return nil, err
	}

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
	var datasources []Datasource
	err := c.getJSON("/api/datasources", nil, &datasources)
	if err != nil {
		return nil, err
	}

	return datasources, nil
}
