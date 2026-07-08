package grafana

import (
	"encoding/json"
	"net/http"
)

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
	resp, err := c.httpClient.Do(req)
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
	resp, err := c.httpClient.Do(req)
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
