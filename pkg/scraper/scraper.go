package scraper

import (
	"log"
	"sigprune/pkg/grafana"
	"sigprune/pkg/utils"
)

type Scraper struct {
	tsdbMetricsLimit int
	grafanaURL       string
	adminUser        string
	adminPassword    string
	datasource       string
}

func NewScraper(
	tsdbMetricsLimit int, grafanaURL string,
	adminUser string,
	adminPassword string,
	datasource string,
) *Scraper {
	return &Scraper{
		tsdbMetricsLimit: tsdbMetricsLimit,
		grafanaURL:       grafanaURL,
		adminUser:        adminUser,
		adminPassword:    adminPassword,
		datasource:       datasource,
	}
}

func (s *Scraper) GetGrafanaDashboardMetrics() []string {
	var metrics []string
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	dashboards, err := g.GetDashboards()
	if err != nil {
		log.Fatal(err)
	}

	for _, dashboard := range dashboards {
		dashboardResponse, err := g.GetDashboardByUID(dashboard.UID)
		if err != nil {
			log.Fatal(err)
		}
		panelExprs := grafana.GetDashboardPanelExprs(*dashboardResponse)
		for _, expr := range panelExprs {
			metricNames := utils.ExtractMetricNames(expr)
			metrics = append(metrics, metricNames...)
		}
	}
	return metrics
}

func (s *Scraper) GetGrafanaAlertRuleMetrics() []string {
	var metrics []string
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	alerts, err := g.GetAlertRules()
	if err != nil {
		log.Fatal(err)
	}
	for _, alert := range alerts {
		alertRule, err := g.GetAlertRuleByUID(alert.UID)
		if err != nil {
			log.Fatal(err)
		}
		alertExprs := grafana.GetAlertRuleExprs(*alertRule)
		for _, expr := range alertExprs {
			metricNames := utils.ExtractMetricNames(expr)
			metrics = append(metrics, metricNames...)
		}
	}
	return metrics
}

func (s *Scraper) GetTopTSDBMetrics() []grafana.MetricCount {
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	return g.GetTopTSDBMetrics(s.datasource, s.tsdbMetricsLimit)
}

func (s *Scraper) GetJobsForMetric(metric string) map[string]int64 {
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	return g.GetPrometheusJobsForMetric(s.datasource, metric)
}
