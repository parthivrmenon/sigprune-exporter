package grafana

type AlertRule struct {
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
	var alertRules []AlertRule
	err := c.getJSON("/api/v1/provisioning/alert-rules", nil, &alertRules)
	if err != nil {
		return nil, err
	}
	return alertRules, nil
}

func GetAlertRuleExprs(a AlertRule) []string {
	var allExprs []string
	for _, queryData := range a.Data {
		if queryData.Model.Expr != "" {
			allExprs = append(allExprs, queryData.Model.Expr)
		}
	}
	return allExprs
}
