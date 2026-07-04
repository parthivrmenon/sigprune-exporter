package main

import (
	"reflect"
	"testing"

	"sigprune/pkg/utils"
)

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
