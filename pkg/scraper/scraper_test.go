package scraper

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

func TestGetUnusedMetrics(t *testing.T) {
	s := NewScraper(5, mockGrafanaServer.URL, "notarealuser", "notarealpassword", "notarealdatasource")
	unusedMetrics := s.GetUnusedMetrics()
	expectedUnusedMetrics := []string{
		"prometheus_http_requests_total",
		"node_filesystem_device_error",
		"node_filesystem_readonly",
		"node_scrape_collector_success",
		"node_scrape_collector_duration_seconds",
	}
	if len(unusedMetrics) != len(expectedUnusedMetrics) {
		t.Fatalf("Expected %d metrics, got %d: %v", len(expectedUnusedMetrics), len(unusedMetrics), unusedMetrics)
	}

	for _, expected := range expectedUnusedMetrics {
		found := false
		for _, metric := range unusedMetrics {
			if metric == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected metric %s not found in %v", expected, unusedMetrics)
		}

	}

}
