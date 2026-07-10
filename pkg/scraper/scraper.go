package scraper

import (
	"fmt"
	"log"
	"sigprune/pkg/grafana"
	"sigprune/pkg/utils"
)

var systemLabels = map[string]struct{}{
	"__name__":            {},
	"__address__":         {},
	"__scheme__":          {},
	"__metrics_path__":    {},
	"__scrape_interval__": {},
	"__scrape_timeout__":  {},
	"__param_":            {},
	"__tmp__":             {},
	"__meta_":             {},
	"le":                  {},
	"quantile":            {},
}

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

func (s *Scraper) getGrafanaDashboardLabels() []string {
	var labels []string
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
			labelNames := utils.ExtractLabelNames(expr)
			labels = append(labels, labelNames...)
		}
	}
	return labels

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

func (s *Scraper) getGrafanaAlertRuleLabels() []string {
	var labels []string
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
			labelNames := utils.ExtractLabelNames(expr)
			labels = append(labels, labelNames...)
		}
	}
	return labels
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

func (s *Scraper) GetJobsForMetric(metric string) map[string]int64 {
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	return g.GetPrometheusJobsForMetric(s.datasource, metric)
}

func (s *Scraper) GetJobsForLabel(label string) map[string]int64 {
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	return g.GetPrometheusJobsForLabel(s.datasource, label)
}

type UnusedMetricsAndLabels struct {
	UnusedMetrics []string
	UnusedLabels  []string
}

func (s *Scraper) GetUnusedMetricsAndLabels() UnusedMetricsAndLabels {
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	tsdbStatus := g.GetTSDBStatus(s.datasource, s.tsdbMetricsLimit)

	topTSDBLabels := tsdbStatus.TSDBData.LabelCounts
	fmt.Println("Got", len(topTSDBLabels), "labels from TSDB")

	topTSDBMetrics := tsdbStatus.TSDBData.MetricCounts
	fmt.Println("Got", len(topTSDBMetrics), "metrics from TSDB")

	var filteredLabels []grafana.LabelCount
	for _, label := range topTSDBLabels {
		// skip system labels
		if _, exists := systemLabels[label.Name]; exists {
			continue
		}
		filteredLabels = append(filteredLabels, label)
	}

	usedLabels := s.getGrafanaDashboardLabels()
	fmt.Println("Got", len(usedLabels), "used labels from dashboards")

	alertLabels := s.getGrafanaAlertRuleLabels()
	fmt.Println("Got", len(alertLabels), "used labels from alert rules")

	usedLabels = append(usedLabels, alertLabels...)

	// create a used labels Set for lookup
	usedSet := make(map[string]bool, len(usedLabels))
	for _, m := range usedLabels {
		usedSet[m] = true
	}

	var unusedMetricsAndLabels UnusedMetricsAndLabels

	// Filter out used labels
	for _, label := range filteredLabels {
		if !usedSet[label.Name] {
			unusedMetricsAndLabels.UnusedLabels = append(unusedMetricsAndLabels.UnusedLabels, label.Name)
		}
	}

	usedMetrics := s.getGrafanaDashboardMetrics()
	fmt.Println("Got", len(usedMetrics), "used metrics from dashboards")

	alertMetrics := s.getGrafanaAlertRuleMetrics()
	fmt.Println("Got", len(alertMetrics), "used metrics from alert rules")

	usedMetrics = append(usedMetrics, alertMetrics...)

	// create a used metrics Set for lookup
	usedSet = make(map[string]bool, len(usedMetrics))
	for _, m := range usedMetrics {
		usedSet[m] = true
	}

	// Filter out used metrics
	for _, metric := range topTSDBMetrics {
		if !usedSet[metric.Name] {
			unusedMetricsAndLabels.UnusedMetrics = append(unusedMetricsAndLabels.UnusedMetrics, metric.Name)
		}
	}

	return unusedMetricsAndLabels

}

func (s *Scraper) TestConnection() {
	g := grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
	if err := g.TestConnection(); err != nil {
		log.Fatal("Failed to connect to Grafana:", err)
	}

}
