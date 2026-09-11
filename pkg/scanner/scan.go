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

type ScanResult struct {
	UnusedMetrics      map[string]map[string]int64
	UnusedLabels       map[string]map[string]int64
	DashboardCount     int
	AlertRuleCount     int
	TotalTSDBMetrics   int
	TotalTSDBLabels    int
	TotalUsedMetrics   int
	TotalUsedLabels    int
	TotalUnusedMetrics int
	TotalUnusedLabels  int
	MetricsExportLimit int
	LabelsExportLimit  int
}

func (s *Scanner) Scan() (ScanResult, error) {
	var scanResult ScanResult

	g := s.newGrafanaClient()

	tsdbStatus, err := g.GetTSDBStatus(s.datasource, s.tsdbMetricsLimit)
	if err != nil {
		return ScanResult{}, err
	}

	topTSDBLabels := tsdbStatus.TSDBData.LabelCounts
	log.Printf("Got %d labels from TSDB", len(topTSDBLabels))
	scanResult.TotalTSDBLabels = len(topTSDBLabels)

	topTSDBMetrics := tsdbStatus.TSDBData.MetricCounts
	log.Printf("Got %d metrics from TSDB", len(topTSDBMetrics))
	scanResult.TotalTSDBMetrics = len(topTSDBMetrics)

	var filteredLabels []grafana.LabelCount
	for _, label := range topTSDBLabels {
		// skip system labels
		if _, exists := systemLabels[label.Name]; exists {
			continue
		}
		filteredLabels = append(filteredLabels, label)
	}

	// Scan Dashboards
	dashboards, err := g.GetDashboards()
	if err != nil {
		return ScanResult{}, err
	}
	scanResult.DashboardCount = len(dashboards)

	var metrics []string
	var labels []string
	for _, dashboard := range dashboards {
		dashboardResponse, err := g.GetDashboardByUID(dashboard.UID)
		if err != nil {
			return ScanResult{}, err
		}
		panelExprs := grafana.GetDashboardPanelExprs(*dashboardResponse)
		for _, expr := range panelExprs {
			labelNames := utils.ExtractLabelNames(expr)
			labels = append(labels, labelNames...)
			metricNames := utils.ExtractMetricNames(expr)
			metrics = append(metrics, metricNames...)
		}
	}

	// Scan Alert Rules
	alerts, err := g.GetAlertRules()
	if err != nil {
		return ScanResult{}, err
	}

	scanResult.AlertRuleCount = len(alerts)

	for _, alert := range alerts {
		alertExprs := grafana.GetAlertRuleExprs(alert)
		for _, expr := range alertExprs {
			labelNames := utils.ExtractLabelNames(expr)
			labels = append(labels, labelNames...)
			metricNames := utils.ExtractMetricNames(expr)
			metrics = append(metrics, metricNames...)
		}
	}

	// create a used labels Set for lookup
	usedSet := make(map[string]bool, len(labels))
	for _, m := range labels {
		usedSet[m] = true
	}

	scanResult.UnusedMetrics = make(map[string]map[string]int64)
	scanResult.UnusedLabels = make(map[string]map[string]int64)

	// Filter out used labels
	var unusedLabels []string
	for _, label := range filteredLabels {
		if !usedSet[label.Name] && len(unusedLabels) < s.exportLimitLabels {
			unusedLabels = append(unusedLabels, label.Name)
		}
	}

	// usedMetrics := append(dashboardData.Metrics, alertData.Metrics...)
	// log.Printf("Got %d used metrics from dashboards and alert rules", len(usedMetrics))

	// create a used metrics Set for lookup
	usedSet = make(map[string]bool, len(metrics))
	for _, m := range metrics {
		usedSet[m] = true
	}

	// Filter out used metrics
	var unusedMetrics []string
	for _, metric := range topTSDBMetrics {
		if !usedSet[metric.Name] && len(unusedMetrics) < s.exportLimitMetrics {
			unusedMetrics = append(unusedMetrics, metric.Name)
		}
	}

	metricJobMap := g.GetPrometheusJobsForMetrics(s.datasource, unusedMetrics)

	for metric, jobCounts := range metricJobMap {
		for job, count := range jobCounts {
			if scanResult.UnusedMetrics[metric] == nil {
				scanResult.UnusedMetrics[metric] = make(map[string]int64)
			}
			scanResult.UnusedMetrics[metric][job] = count
		}
	}

	for _, label := range unusedLabels {
		jobs := g.GetPrometheusJobsForLabel(s.datasource, label)
		for job, count := range jobs {
			if scanResult.UnusedLabels[label] == nil {
				scanResult.UnusedLabels[label] = make(map[string]int64)
			}
			scanResult.UnusedLabels[label][job] = count
		}
	}

	// Populate statistics
	scanResult.TotalUsedMetrics = len(metrics)
	scanResult.TotalUsedLabels = len(labels)
	scanResult.TotalUnusedMetrics = len(unusedMetrics)
	scanResult.TotalUnusedLabels = len(unusedLabels)
	scanResult.MetricsExportLimit = s.exportLimitMetrics
	scanResult.LabelsExportLimit = s.exportLimitLabels

	log.Printf("Returning %d unused metrics", len(unusedMetrics))
	log.Printf("Returning %d unused labels", len(unusedLabels))

	return scanResult, nil

}

func (s *Scanner) TestConnection() {
	g := s.newGrafanaClient()
	if err := g.TestConnection(); err != nil {
		log.Fatal("Failed to connect to Grafana:", err)
	}

}
