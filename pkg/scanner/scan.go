package scanner

import (
	"log"
	"sigprune/pkg/grafana"
	"sigprune/pkg/utils"
	"strings"
	"sync"
	"time"
)

var systemLabels = map[string]struct{}{
	"le":       {},
	"quantile": {},
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
	grafanaTimeout     time.Duration
	mu                 sync.RWMutex
	snapshot           *ScanResult
	lastErr            error
}

func NewScanner(
	tsdbMetricsLimit int, grafanaURL string,
	adminUser string,
	adminPassword string,
	apiKey string,
	datasource string,
	exportLimitMetrics int,
	exportLimitLabels int,
	grafanaTimeout time.Duration,
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
		grafanaTimeout:     grafanaTimeout,
	}
}

func (s *Scanner) newGrafanaClient() *grafana.Client {
	if s.apiKey != "" {
		return grafana.NewClientWithAPIKey(s.grafanaURL, s.apiKey, s.grafanaTimeout)
	}
	return grafana.NewClient(s.grafanaURL, s.adminUser, s.adminPassword, s.grafanaTimeout)
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

// Returns the result of the last successful scan. It is nil until the first scan completes
func (s *Scanner) GetSnapshot() (*ScanResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot, s.lastErr
}

// Run scans once immediately and then repeats on every tick of interval.
// It blocks forever and is meant to be started as a goroutine.
func (s *Scanner) Run(interval time.Duration) {
	s.RefreshOnce()
	t := time.NewTicker(interval)
	defer t.Stop()
	for range t.C {
		s.RefreshOnce()
	}
}

// RefreshOnce runs one full scan and publishes the result.
func (s *Scanner) RefreshOnce() {
	result, err := s.Scan()

	s.mu.Lock()
	if err != nil {
		s.lastErr = err // keep the previous snapshot and simply mark the error
	} else {
		s.snapshot = &result // update the pointer
		s.lastErr = nil      // and clear the error
	}
	s.mu.Unlock()

	if err != nil {
		log.Printf("Scan failed (serving previous snapshot): %v", err)
	}
}

// Check if label matches a system label
func isSystemLabel(labelName string) bool {
	if _, exists := systemLabels[labelName]; exists {
		return true
	}
	if strings.HasPrefix(labelName, "__") {
		return true
	}
	return false

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
		if isSystemLabel(label.Name) {
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
	usedLabelSet := make(map[string]bool, len(labels))
	for _, m := range labels {
		usedLabelSet[m] = true
	}

	scanResult.UnusedMetrics = make(map[string]map[string]int64)
	scanResult.UnusedLabels = make(map[string]map[string]int64)

	// Filter out used labels
	var unusedLabels []string
	for _, label := range filteredLabels {
		if !usedLabelSet[label.Name] && len(unusedLabels) < s.exportLimitLabels {
			unusedLabels = append(unusedLabels, label.Name)
		}
	}

	// create a used metrics Set for lookup
	usedMetricSet := make(map[string]bool, len(metrics))
	for _, m := range metrics {
		usedMetricSet[m] = true
	}

	// Filter out used metrics
	var unusedMetrics []string
	for _, metric := range topTSDBMetrics {
		if !usedMetricSet[metric.Name] && len(unusedMetrics) < s.exportLimitMetrics {
			unusedMetrics = append(unusedMetrics, metric.Name)
		}
	}

	metricJobMap, err := g.GetPrometheusJobsForMetrics(s.datasource, unusedMetrics)
	if err != nil {
		return ScanResult{}, err
	}

	for metric, jobCounts := range metricJobMap {
		for job, count := range jobCounts {
			if scanResult.UnusedMetrics[metric] == nil {
				scanResult.UnusedMetrics[metric] = make(map[string]int64)
			}
			scanResult.UnusedMetrics[metric][job] = count
		}
	}

	for _, label := range unusedLabels {
		jobs, err := g.GetPrometheusJobsForLabel(s.datasource, label)
		if err != nil {
			return ScanResult{}, err
		}
		for job, count := range jobs {
			if scanResult.UnusedLabels[label] == nil {
				scanResult.UnusedLabels[label] = make(map[string]int64)
			}
			scanResult.UnusedLabels[label][job] = count
		}
	}

	// Populate statistics
	scanResult.TotalUsedMetrics = len(usedMetricSet)
	scanResult.TotalUsedLabels = len(usedLabelSet)
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
