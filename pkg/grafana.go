package pkg

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	URL      string
	Username string
	Password string
}

func NewClient(url, username, password string) *Client {
	return &Client{
		URL:      url,
		Username: username,
		Password: password,
	}
}

type HealthResponse struct {
	Status  string `json:"database"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

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

func (c *Client) TestConnection() error {
	fmt.Println("Testing connection to Grafana at", c.URL)

	req, err := http.NewRequest("GET", c.URL+"/api/health", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
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

// /api/v1/provisioning/alert-rules

type AlertRule struct {
	ID    int    `json:"id"`
	UID   string `json:"uid"`
	Title string `json:"title"`
}

type AlertRuleResponse struct {
	ID    int              `json:"id"`
	UID   string           `json:"uid"`
	Title string           `json:"title"`
	Data  []AlertQueryData `json:"data"`
}

type AlertQueryData struct {
	RefID             string `json:"refId"`
	QueryType         string `json:"queryType"`
	RelativeTimeRange struct {
		From int `json:"from"`
		To   int `json:"to"`
	} `json:"relativeTimeRange"`
	DatasourceUID string          `json:"datasourceUid"`
	Model         AlertQueryModel `json:"model"`
}

type AlertQueryModel struct {
	Expr          string `json:"expr,omitempty"`
	Hide          bool   `json:"hide"`
	IntervalMs    int    `json:"intervalMs,omitempty"`
	MaxDataPoints int    `json:"maxDataPoints,omitempty"`
	RefID         string `json:"refId"`
	Type          string `json:"type,omitempty"`
}

func (c *Client) GetAlertRules() ([]AlertRule, error) {
	req, err := http.NewRequest("GET", c.URL+"/api/v1/provisioning/alert-rules", nil)
	req.SetBasicAuth(c.Username, c.Password)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var alertRules []AlertRule
	err = json.NewDecoder(resp.Body).Decode(&alertRules)
	if err != nil {
		return nil, err
	}

	return alertRules, nil
}

func (c *Client) GetAlertRuleByUID(uid string) (*AlertRuleResponse, error) {
	req, err := http.NewRequest("GET", c.URL+"/api/v1/provisioning/alert-rules/"+uid, nil)
	req.SetBasicAuth(c.Username, c.Password)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var alertRuleResponse AlertRuleResponse
	err = json.NewDecoder(resp.Body).Decode(&alertRuleResponse)
	if err != nil {
		return nil, err
	}

	return &alertRuleResponse, nil
}

func GetAlertRuleExprs(a AlertRuleResponse) []string {
	var allExprs []string
	for _, queryData := range a.Data {
		if queryData.Model.Expr != "" {
			allExprs = append(allExprs, queryData.Model.Expr)
		}
	}
	return allExprs
}

func (c *Client) GetDashboards() ([]Dashboard, error) {
	req, err := http.NewRequest("GET", c.URL+"/api/search?type=dash-db", nil)
	req.SetBasicAuth(c.Username, c.Password)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
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
	req.SetBasicAuth(c.Username, c.Password)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
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
	var allExprs []string
	for _, panel := range d.Dashboard.Panels {
		for _, target := range panel.Targets {
			allExprs = append(allExprs, target.Expr)
		}
	}
	return allExprs
}

func (c *Client) GetDatasources() ([]Datasource, error) {
	req, err := http.NewRequest("GET", c.URL+"/api/datasources", nil)
	req.SetBasicAuth(c.Username, c.Password)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
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
