package scraper

import (
	"fmt"
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

func (s *Scraper) getGrafanaDashboardMetrics() []string {
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

func (s *Scraper) getGrafanaAlertRuleMetrics() []string {
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

func (s *Scraper) getTopTSDBMetrics() []grafana.MetricCount {
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	return g.GetTopTSDBMetrics(s.datasource, s.tsdbMetricsLimit)
}

func (s *Scraper) GetJobsForMetric(metric string) map[string]int64 {
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	return g.GetPrometheusJobsForMetric(s.datasource, metric)
}

func (s *Scraper) GetUnusedMetrics() []string {
	topTSDBMetrics := s.getTopTSDBMetrics()
	fmt.Println("Got", len(topTSDBMetrics), "metrics from TSDB")

	usedMetrics := s.getGrafanaDashboardMetrics()
	fmt.Println("Got", len(usedMetrics), "used metrics from dashboards")

	alertMetrics := s.getGrafanaAlertRuleMetrics()
	fmt.Println("Got", len(alertMetrics), "used metrics from alert rules")

	usedMetrics = append(usedMetrics, alertMetrics...)

	// create a used metrics Set for lookup
	usedSet := make(map[string]bool, len(usedMetrics))
	for _, m := range usedMetrics {
		usedSet[m] = true
	}

	// Filter out used metrics
	var unusedMetrics []string
	for _, metric := range topTSDBMetrics {
		if !usedSet[metric.Name] {
			unusedMetrics = append(unusedMetrics, metric.Name)
		}
	}

	return unusedMetrics

}

func (s *Scraper) TestConnection() {
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	if err := g.TestConnection(); err != nil {
		log.Fatal("Failed to connect to Grafana:", err)
	}

}
