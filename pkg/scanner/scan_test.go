package scanner

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var mockGrafanaServer *httptest.Server

func TestMain(m *testing.M) {
	// Load all fixture data
	dashboardsData, _ := os.ReadFile("testdata/dashboards.json")
	dashboardDetailData, _ := os.ReadFile("testdata/dashboard_detail.json")
	alertRulesData, _ := os.ReadFile("testdata/alert_rules.json")
	alertRuleHostDownData, _ := os.ReadFile("testdata/alert_rule_host_down.json")
	alertRuleHighCPUData, _ := os.ReadFile("testdata/alert_rule_high_cpu.json")
	alertRuleDiskFillingData, _ := os.ReadFile("testdata/alert_rule_disk_filling.json")
	alertRuleHighMemoryData, _ := os.ReadFile("testdata/alert_rule_high_memory.json")
	tsdbStatus, _ := os.ReadFile("testdata/tsdb_status.json")

	// Create single mock server with all endpoints
	mockGrafanaServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/search":
			if r.URL.Query().Get("type") == "dash-db" {
				w.Header().Set("Content-Type", "application/json")
				w.Write(dashboardsData)
			}
		case "/api/dashboards/uid/nodeExporterBasic":
			w.Header().Set("Content-Type", "application/json")
			w.Write(dashboardDetailData)
		case "/api/v1/provisioning/alert-rules":
			w.Header().Set("Content-Type", "application/json")
			w.Write(alertRulesData)
		case "/api/v1/provisioning/alert-rules/node_exporter_host_down":
			w.Header().Set("Content-Type", "application/json")
			w.Write(alertRuleHostDownData)
		case "/api/v1/provisioning/alert-rules/node_exporter_high_cpu":
			w.Header().Set("Content-Type", "application/json")
			w.Write(alertRuleHighCPUData)
		case "/api/v1/provisioning/alert-rules/node_exporter_disk_filling":
			w.Header().Set("Content-Type", "application/json")
			w.Write(alertRuleDiskFillingData)
		case "/api/v1/provisioning/alert-rules/node_exporter_high_memory":
			w.Header().Set("Content-Type", "application/json")
			w.Write(alertRuleHighMemoryData)
		case "/api/datasources/proxy/uid/notarealdatasource/api/v1/status/tsdb":
			w.Header().Set("Content-Type", "application/json")
			w.Write(tsdbStatus)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Run tests
	code := m.Run()

	// Cleanup
	mockGrafanaServer.Close()
	os.Exit(code)
}

func TestGetUnusedMetricsAndLabelsWithLimit(t *testing.T) {
	s := NewScanner(5, mockGrafanaServer.URL, "notarealuser", "notarealpassword", "", "notarealdatasource", 1, 1)
	result := s.GetUnusedMetricsAndLabels()

	if len(result.UnusedMetrics) != 1 {
		t.Fatalf("expected 1 metric, got %d: %v", len(result.UnusedMetrics), result.UnusedMetrics)
	}
	if len(result.UnusedLabels) != 1 {
		t.Fatalf("expected 1 label, got %d: %v", len(result.UnusedLabels), result.UnusedLabels)
	}
}

func TestGetUnusedMetricsAndLabels(t *testing.T) {
	s := NewScanner(5, mockGrafanaServer.URL, "notarealuser", "notarealpassword", "", "notarealdatasource", 5, 5)
	result := s.GetUnusedMetricsAndLabels()

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
		found := false
		for _, metric := range result.UnusedMetrics {
			if metric == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected metric %s not found in %v", expected, result.UnusedMetrics)
		}
	}

	if len(result.UnusedLabels) != len(expectedUnusedLabels) {
		t.Fatalf("Expected %d labels, got %d: %v", len(expectedUnusedLabels), len(result.UnusedLabels), result.UnusedLabels)
	}

	for _, expected := range expectedUnusedLabels {
		found := false
		for _, label := range result.UnusedLabels {
			if label == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected label %s not found in %v", expected, result.UnusedLabels)
		}
	}
}
