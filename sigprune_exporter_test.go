package main

import (
	"os"
	"reflect"
	"testing"
	"time"

	sigprunetestutil "sigprune/internal/testutil"
	"sigprune/pkg/utils"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCollectSuccess(t *testing.T) {
	srv := sigprunetestutil.NewMockGrafanaServer()
	defer srv.Close()

	exporter := NewExporter(5, srv.URL, sigprunetestutil.MockUser, sigprunetestutil.MockPassword, "", sigprunetestutil.MockDatasource, 1, 1, 10*time.Second)
	exporter.scanner.RefreshOnce()

	expected, err := os.Open("testdata/collect_success.txt")
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer expected.Close()

	if err := testutil.CollectAndCompare(exporter, expected,
		"sigprune_build_info",
		"sigprune_up",
		"sigprune_unused_label_cardinality",
		"sigprune_unused_metric_cardinality",
	); err != nil {
		t.Errorf("unexpected metric output:\n%v", err)
	}
}

func TestCollectError(t *testing.T) {
	srv := sigprunetestutil.NewMockGrafanaServer()
	defer srv.Close()

	exporter := NewExporter(5, srv.URL, "wronguser", "wrongpassword", "", sigprunetestutil.MockDatasource, 1, 1, 10*time.Second)
	exporter.scanner.RefreshOnce()

	expected, err := os.Open("testdata/collect_error.txt")
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer expected.Close()

	if err := testutil.CollectAndCompare(exporter, expected,
		"sigprune_build_info",
		"sigprune_up",
	); err != nil {
		t.Errorf("unexpected metric output:\n%v", err)
	}
}

// A failed refresh must keep serving the last successful snapshot, with
// sigprune_up 0 reporting that the most recent attempt failed.
func TestCollectStaleSnapshotAfterFailure(t *testing.T) {
	srv := sigprunetestutil.NewMockGrafanaServer()

	exporter := NewExporter(5, srv.URL, sigprunetestutil.MockUser, sigprunetestutil.MockPassword, "", sigprunetestutil.MockDatasource, 1, 1, 10*time.Second)
	exporter.scanner.RefreshOnce() // succeeds: snapshot published

	srv.Close()
	exporter.scanner.RefreshOnce() // fails: lastErr set, snapshot kept

	expected, err := os.Open("testdata/collect_stale.txt")
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer expected.Close()

	if err := testutil.CollectAndCompare(exporter, expected,
		"sigprune_build_info",
		"sigprune_up",
		"sigprune_unused_label_cardinality",
		"sigprune_unused_metric_cardinality",
	); err != nil {
		t.Errorf("unexpected metric output:\n%v", err)
	}
}

func TestExtractMetricNames(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected []string
	}{
		{
			name:     "complex arithmetic with avg and rate",
			expr:     `100 * (1 - avg(rate(node_cpu_seconds_total{mode="idle",instance="$node",job="$job"}[$__rate_interval])))`,
			expected: []string{"node_cpu_seconds_total"},
		},
		{
			name:     "simple metric with label matchers",
			expr:     `node_memory_MemTotal_bytes{instance="$node",job="$job"}`,
			expected: []string{"node_memory_MemTotal_bytes"},
		},
		{
			name:     "binary operation with two metrics",
			expr:     `node_time_seconds{instance="$node",job="$job"} - node_boot_time_seconds{instance="$node",job="$job"}`,
			expected: []string{"node_time_seconds", "node_boot_time_seconds"},
		},
		{
			name:     "rate with multiplication",
			expr:     `rate(node_network_receive_bytes_total{instance="$node",job="$job"}[$__rate_interval])*8`,
			expected: []string{"node_network_receive_bytes_total"},
		},
		{
			name:     "rate with label matchers",
			expr:     `rate(node_pressure_cpu_waiting_seconds_total{instance="$node",job="$job"}[$__rate_interval])`,
			expected: []string{"node_pressure_cpu_waiting_seconds_total"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.ExtractMetricNames(tt.expr)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("utils.ExtractMetricNames(%q) = %v, want %v", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestExtractLabelNames(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected []string
	}{
		{
			name:     "complex arithmetic with multiple label matchers",
			expr:     `100 * (1 - avg(rate(node_cpu_seconds_total{mode="idle",instance="$node",job="$job"}[$__rate_interval])))`,
			expected: []string{"mode", "instance", "job"},
		},
		{
			name:     "simple metric with label matchers",
			expr:     `node_memory_MemTotal_bytes{instance="$node",job="$job"}`,
			expected: []string{"instance", "job"},
		},
		{
			name:     "binary operation with duplicate labels",
			expr:     `node_time_seconds{instance="$node",job="$job"} - node_boot_time_seconds{instance="$node",job="$job"}`,
			expected: []string{"instance", "job"},
		},
		{
			name:     "rate with label matchers",
			expr:     `rate(node_network_receive_bytes_total{instance="$node",job="$job",device="eth0"}[$__rate_interval])*8`,
			expected: []string{"instance", "job", "device"},
		},
		{
			name:     "metric without label matchers",
			expr:     `up`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.ExtractLabelNames(tt.expr)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("utils.ExtractLabelNames(%q) = %v, want %v", tt.expr, got, tt.expected)
			}
		})
	}
}
