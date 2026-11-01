package grafana

import "fmt"

// PrometheusRules is the subset of a Prometheus /api/v1/rules response that
// sigprune reads. The API returns a great deal more per rule - state, duration,
// labels, annotations, health, timings - and encoding/json drops every field not
// named here, so only what gets used is declared.
type PrometheusRules struct {
	RulesData PrometheusRulesData `json:"data"`
}

type PrometheusRulesData struct {
	Groups []PrometheusRuleGroup `json:"groups"`
}

type PrometheusRuleGroup struct {
	Rules []PrometheusRule `json:"rules"`
}

// PrometheusRule is one rule from a group. The API also gives each rule a "type"
// of "alerting" or "recording"; it is deliberately not captured, because both
// kinds are real usage of whatever their query references. Adding the field back
// in order to filter on it would silently drop every recording rule.
type PrometheusRule struct {
	Query string `json:"query"`
}

func (c *Client) PrometheusRules(datasourceUID string) (PrometheusRules, error) {
	rulesURI := fmt.Sprintf("/api/datasources/proxy/uid/%s/api/v1/rules", datasourceUID)
	var rules PrometheusRules
	err := c.getJSON(rulesURI, nil, &rules)
	if err != nil {
		return PrometheusRules{}, err
	}

	return rules, nil

}

// Exprs returns the query of every rule in every group. Alerting and recording
// rules are both included; see PrometheusRule for why they are not told apart.
func (r *PrometheusRules) Exprs() []string {
	var exprs []string
	for _, g := range r.RulesData.Groups {
		for _, rule := range g.Rules {
			if rule.Query != "" {
				exprs = append(exprs, rule.Query)
			}
		}
	}
	return exprs
}
