package utils

import (
	"reflect"
	"testing"
)

func TestExtractLabelNames(t *testing.T) {
	tests := []struct {
		expr       string
		wantLabels []string
	}{
		{
			//test for VectorSelector.LabelMatchers
			expr:       "rate(node_pressure_memory_waiting_seconds_total{instance=\"$node\",job=\"$job\"}[$__rate_interval])",
			wantLabels: []string{"instance", "job"},
		},
		{
			// test for AggregateExpr.Grouping
			expr:       `sum by (device) (rate(node_disk_read_bytes_total{job="node-exporter"}[5m]))`,
			wantLabels: []string{"device", "job"},
		},
		{
			// BinaryExpr with a scalar operand - VectorMatching is nil here.
			expr:       `rate(node_cpu_seconds_total{mode="idle"}[5m]) * 100 > 80`,
			wantLabels: []string{"mode"},
		},
		{
			// test for BinaryExpr.VectorMatching
			// with MatchingLabels
			expr:       `node_filesystem_avail_bytes / on (instance, mountpoint) node_filesystem_size_bytes`,
			wantLabels: []string{"instance", "mountpoint"},
		},
		{
			// test for BinaryExpr.VectorMatching branch ...
			// with Include
			expr:       `node_cpu_seconds_total * on (instance) group_left (nodename) node_uname_info`,
			wantLabels: []string{"instance", "nodename"},
		},
		{
			// A BinaryExpr whose other operand is a scalar: VectorMatching is
			// nil here, not empty, so reading it without a nil check panics.
			// __name__ is spelled out as a matcher to confirm it is skipped
			// rather than reported as a label.
			expr:       `rate({__name__="node_cpu_seconds_total", mode="idle"}[5m]) * 100 > 80`,
			wantLabels: []string{"mode"},
		},
	}

	for _, tt := range tests {
		labels := ExtractLabelNames(tt.expr)
		if !reflect.DeepEqual(labels, tt.wantLabels) {
			t.Errorf("labels: got %v, want %v", labels, tt.wantLabels)
		}

	}
}
