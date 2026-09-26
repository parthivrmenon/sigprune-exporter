package scanner

import (
	"os"
	"testing"
	"time"

	sigprunetestutil "sigprune/internal/testutil"
)

var mockGrafanaServerURL string

func TestMain(m *testing.M) {
	srv := sigprunetestutil.NewMockGrafanaServer()
	mockGrafanaServerURL = srv.URL
	code := m.Run()
	srv.Close()
	os.Exit(code)
}

func TestGetUnusedMetricsAndLabelsWithLimit(t *testing.T) {
	s := NewScanner(5, mockGrafanaServerURL, sigprunetestutil.MockUser, sigprunetestutil.MockPassword, "", sigprunetestutil.MockDatasource, 1, 1, 10*time.Second)
	result, err := s.Scan()
	if err != nil {
		t.Fatalf("Did not expect err %v", err)
	}

	if len(result.UnusedMetrics) != 1 {
		t.Fatalf("expected 1 metric, got %d: %v", len(result.UnusedMetrics), result.UnusedMetrics)
	}
	if len(result.UnusedLabels) != 1 {
		t.Fatalf("expected 1 label, got %d: %v", len(result.UnusedLabels), result.UnusedLabels)
	}

	// Verify the nested map structure is correct
	for metric, jobCounts := range result.UnusedMetrics {
		if len(jobCounts) == 0 {
			t.Fatalf("expected job counts for metric %s, got empty map", metric)
		}
	}
	for label, jobCounts := range result.UnusedLabels {
		if len(jobCounts) == 0 {
			t.Fatalf("expected job counts for label %s, got empty map", label)
		}
	}

	// Assert export limits
	if result.MetricsExportLimit != 1 {
		t.Errorf("Expected MetricsExportLimit 1, got %d", result.MetricsExportLimit)
	}
	if result.LabelsExportLimit != 1 {
		t.Errorf("Expected LabelsExportLimit 1, got %d", result.LabelsExportLimit)
	}
}

func TestScan(t *testing.T) {
	s := NewScanner(5, mockGrafanaServerURL, sigprunetestutil.MockUser, sigprunetestutil.MockPassword, "", sigprunetestutil.MockDatasource, 5, 5, 10*time.Second)
	result, err := s.Scan()
	if err != nil {
		t.Fatalf("Did not expect err %v", err)
	}

	expectedUnusedMetrics := []string{
		"prometheus_http_requests_total",
		"node_filesystem_device_error",
		"node_filesystem_readonly",
		"node_scrape_collector_success",
		"node_scrape_collector_duration_seconds",
	}

	expectedUnusedLabels := []string{
		"handler",
		"collector",
	}

	if len(result.UnusedMetrics) != len(expectedUnusedMetrics) {
		t.Fatalf("Expected %d metrics, got %d: %v", len(expectedUnusedMetrics), len(result.UnusedMetrics), result.UnusedMetrics)
	}

	for _, expected := range expectedUnusedMetrics {
		jobCounts, found := result.UnusedMetrics[expected]
		if !found {
			t.Fatalf("Expected metric %s not found in %v", expected, result.UnusedMetrics)
		}
		if len(jobCounts) == 0 {
			t.Fatalf("Expected job counts for metric %s, got empty map", expected)
		}
	}

	if len(result.UnusedLabels) != len(expectedUnusedLabels) {
		t.Fatalf("Expected %d labels, got %d: %v", len(expectedUnusedLabels), len(result.UnusedLabels), result.UnusedLabels)
	}

	for _, expected := range expectedUnusedLabels {
		jobCounts, found := result.UnusedLabels[expected]
		if !found {
			t.Fatalf("Expected label %s not found in %v", expected, result.UnusedLabels)
		}
		if len(jobCounts) == 0 {
			t.Fatalf("Expected job counts for label %s, got empty map", expected)
		}
	}

	// Assert statistics
	expectedStats := struct {
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
	}{
		DashboardCount:     1,  // From dashboards.json
		AlertRuleCount:     4,  // From alert_rules.json
		TotalTSDBMetrics:   5,  // From tsdb_status.json (limited by tsdb-metrics-limit=5)
		TotalTSDBLabels:    6,  // Every label in tsdb_status.json, before system-label filtering
		TotalUsedMetrics:   11, // From dashboard and alert rule expressions
		TotalUsedLabels:    10, // From dashboard and alert rule expressions
		TotalUnusedMetrics: 5,  // All TSDB metrics are unused
		TotalUnusedLabels:  2,  // handler, collector are unused
		MetricsExportLimit: 5,  // From scanner config
		LabelsExportLimit:  5,  // From scanner config
	}

	if result.DashboardCount != expectedStats.DashboardCount {
		t.Errorf("Expected DashboardCount %d, got %d", expectedStats.DashboardCount, result.DashboardCount)
	}
	if result.AlertRuleCount != expectedStats.AlertRuleCount {
		t.Errorf("Expected AlertRuleCount %d, got %d", expectedStats.AlertRuleCount, result.AlertRuleCount)
	}
	if result.TotalTSDBMetrics != expectedStats.TotalTSDBMetrics {
		t.Errorf("Expected TotalTSDBMetrics %d, got %d", expectedStats.TotalTSDBMetrics, result.TotalTSDBMetrics)
	}
	if result.TotalTSDBLabels != expectedStats.TotalTSDBLabels {
		t.Errorf("Expected TotalTSDBLabels %d, got %d", expectedStats.TotalTSDBLabels, result.TotalTSDBLabels)
	}
	if result.TotalUsedMetrics != expectedStats.TotalUsedMetrics {
		t.Errorf("Expected TotalUsedMetrics %d, got %d", expectedStats.TotalUsedMetrics, result.TotalUsedMetrics)
	}
	if result.TotalUsedLabels != expectedStats.TotalUsedLabels {
		t.Errorf("Expected TotalUsedLabels %d, got %d", expectedStats.TotalUsedLabels, result.TotalUsedLabels)
	}
	if result.TotalUnusedMetrics != expectedStats.TotalUnusedMetrics {
		t.Errorf("Expected TotalUnusedMetrics %d, got %d", expectedStats.TotalUnusedMetrics, result.TotalUnusedMetrics)
	}
	if result.TotalUnusedLabels != expectedStats.TotalUnusedLabels {
		t.Errorf("Expected TotalUnusedLabels %d, got %d", expectedStats.TotalUnusedLabels, result.TotalUnusedLabels)
	}
	if result.MetricsExportLimit != expectedStats.MetricsExportLimit {
		t.Errorf("Expected MetricsExportLimit %d, got %d", expectedStats.MetricsExportLimit, result.MetricsExportLimit)
	}
	if result.LabelsExportLimit != expectedStats.LabelsExportLimit {
		t.Errorf("Expected LabelsExportLimit %d, got %d", expectedStats.LabelsExportLimit, result.LabelsExportLimit)
	}
}

func TestScanUnauthorized(t *testing.T) {
	s := NewScanner(5, mockGrafanaServerURL, "wronguser", "wrongpassword", "", sigprunetestutil.MockDatasource, 5, 5, 10*time.Second)
	_, err := s.Scan()
	if err == nil {
		t.Fatal("expected error for unauthorized request, got nil")
	}
}
