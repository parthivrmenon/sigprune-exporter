package scanner

import (
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

type Scanner struct {
	tsdbMetricsLimit   int
	grafanaURL         string
	adminUser          string
	adminPassword      string
	apiKey             string
	datasource         string
	exportLimitMetrics int
	exportLimitLabels  int
}

func NewScanner(
	tsdbMetricsLimit int, grafanaURL string,
	adminUser string,
	adminPassword string,
	apiKey string,
	datasource string,
	exportLimitMetrics int,
	exportLimitLabels int,
) *Scanner {
	return &Scanner{
		tsdbMetricsLimit:   tsdbMetricsLimit,
		grafanaURL:         grafanaURL,
		adminUser:          adminUser,
		adminPassword:      adminPassword,
		apiKey:             apiKey,
		datasource:         datasource,
		exportLimitMetrics: exportLimitMetrics,
		exportLimitLabels:  exportLimitLabels,
	}
}

func (s *Scanner) newGrafanaClient() *grafana.Client {
	if s.apiKey != "" {
		return grafana.NewClientWithAPIKey(s.grafanaURL, s.apiKey)
	}
	return grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword)
}

func (s *Scanner) getGrafanaDashboardMetricsAndLabels() MetricsAndLabels {
	var result MetricsAndLabels
	g := s.newGrafanaClient()
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
			result.Labels = append(result.Labels, labelNames...)
			metricNames := utils.ExtractMetricNames(expr)
			result.Metrics = append(result.Metrics, metricNames...)
		}
	}
	return result
}

func (s *Scanner) getGrafanaAlertRuleMetricsAndLabels() MetricsAndLabels {
	var result MetricsAndLabels
	g := s.newGrafanaClient()
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
			result.Labels = append(result.Labels, labelNames...)
			metricNames := utils.ExtractMetricNames(expr)
			result.Metrics = append(result.Metrics, metricNames...)
		}
	}
	return result
}

func (s *Scanner) GetJobsForMetric(metric string) map[string]int64 {
	g := s.newGrafanaClient()
	return g.GetPrometheusJobsForMetric(s.datasource, metric)
}

func (s *Scanner) GetJobsForLabel(label string) map[string]int64 {
	g := s.newGrafanaClient()
	return g.GetPrometheusJobsForLabel(s.datasource, label)
}

type MetricsAndLabels struct {
	Metrics []string
	Labels  []string
}

type UnusedMetricsAndLabels struct {
	UnusedMetrics []string
	UnusedLabels  []string
}

func (s *Scanner) GetUnusedMetricsAndLabels() UnusedMetricsAndLabels {
	g := s.newGrafanaClient()
	tsdbStatus := g.GetTSDBStatus(s.datasource, s.tsdbMetricsLimit)

	topTSDBLabels := tsdbStatus.TSDBData.LabelCounts
	log.Printf("Got %d labels from TSDB", len(topTSDBLabels))

	topTSDBMetrics := tsdbStatus.TSDBData.MetricCounts
	log.Printf("Got %d metrics from TSDB", len(topTSDBMetrics))

	var filteredLabels []grafana.LabelCount
	for _, label := range topTSDBLabels {
		// skip system labels
		if _, exists := systemLabels[label.Name]; exists {
			continue
		}
		filteredLabels = append(filteredLabels, label)
	}

	dashboardData := s.getGrafanaDashboardMetricsAndLabels()
	log.Printf("Got %d used labels from dashboards", len(dashboardData.Labels))

	alertData := s.getGrafanaAlertRuleMetricsAndLabels()
	log.Printf("Got %d used labels from alert rules", len(alertData.Labels))

	usedLabels := append(dashboardData.Labels, alertData.Labels...)

	// create a used labels Set for lookup
	usedSet := make(map[string]bool, len(usedLabels))
	for _, m := range usedLabels {
		usedSet[m] = true
	}

	var unusedMetricsAndLabels UnusedMetricsAndLabels

	// Filter out used labels
	for _, label := range filteredLabels {
		if !usedSet[label.Name] && len(unusedMetricsAndLabels.UnusedLabels) < s.exportLimitLabels {
			unusedMetricsAndLabels.UnusedLabels = append(unusedMetricsAndLabels.UnusedLabels, label.Name)
		}
	}

	usedMetrics := append(dashboardData.Metrics, alertData.Metrics...)
	log.Printf("Got %d used metrics from dashboards and alert rules", len(usedMetrics))

	// create a used metrics Set for lookup
	usedSet = make(map[string]bool, len(usedMetrics))
	for _, m := range usedMetrics {
		usedSet[m] = true
	}

	// Filter out used metrics
	for _, metric := range topTSDBMetrics {
		if !usedSet[metric.Name] && len(unusedMetricsAndLabels.UnusedMetrics) < s.exportLimitMetrics {
			unusedMetricsAndLabels.UnusedMetrics = append(unusedMetricsAndLabels.UnusedMetrics, metric.Name)
		}
	}

	log.Printf("Returning %d unused metrics", len(unusedMetricsAndLabels.UnusedMetrics))
	log.Printf("Returning %d unused labels", len(unusedMetricsAndLabels.UnusedLabels))

	return unusedMetricsAndLabels

}

func (s *Scanner) TestConnection() {
	g := s.newGrafanaClient()
	if err := g.TestConnection(); err != nil {
		log.Fatal("Failed to connect to Grafana:", err)
	}

}
