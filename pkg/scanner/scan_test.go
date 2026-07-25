package scanner

import (
	"os"
	"testing"

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
	s := NewScanner(5, mockGrafanaServerURL, sigprunetestutil.MockUser, sigprunetestutil.MockPassword, "", sigprunetestutil.MockDatasource, 1, 1)
	result, err := s.GetUnusedMetricsAndLabels()
	if err != nil {
		t.Fatalf("Did not expect err %v", err)
	}

	if len(result.UnusedMetrics) != 1 {
		t.Fatalf("expected 1 metric, got %d: %v", len(result.UnusedMetrics), result.UnusedMetrics)
	}
	if len(result.UnusedLabels) != 1 {
		t.Fatalf("expected 1 label, got %d: %v", len(result.UnusedLabels), result.UnusedLabels)
	}
}

func TestGetUnusedMetricsAndLabels(t *testing.T) {
	s := NewScanner(5, mockGrafanaServerURL, sigprunetestutil.MockUser, sigprunetestutil.MockPassword, "", sigprunetestutil.MockDatasource, 5, 5)
	result, err := s.GetUnusedMetricsAndLabels()
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

func TestGetUnusedMetricsAndLabelsUnauthorized(t *testing.T) {
	s := NewScanner(5, mockGrafanaServerURL, "wronguser", "wrongpassword", "", sigprunetestutil.MockDatasource, 5, 5)
	_, err := s.GetUnusedMetricsAndLabels()
	if err == nil {
		t.Fatal("expected error for unauthorized request, got nil")
	}
}
