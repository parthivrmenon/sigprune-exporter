package utils

import (
	"reflect"
	"testing"
)

func TestExtractMetricsAndLabelsFromLabelValuesQuery(t *testing.T) {
	tests := []struct {
		query       string
		wantMetrics []string
		wantLabels  []string
	}{
		{
			query:       "label_values(node_uname_info, job)",
			wantMetrics: []string{"node_uname_info"},
			wantLabels:  []string{"job"},
		},
		{
			query:       `label_values(node_uname_info{job="$job"}, nodename)`,
			wantMetrics: []string{"node_uname_info"},
			wantLabels:  []string{"job", "nodename"},
		},
		{
			query:       `label_values(node_network_info{origin_prometheus=~"$origin_prometheus",device!~'tap.*'}, device)`,
			wantMetrics: []string{"node_network_info"},
			wantLabels:  []string{"device", "origin_prometheus"},
		},
		{
			query:       `label_values(up{job="prometheus"}, job)`,
			wantMetrics: []string{"up"},
			wantLabels:  []string{"job"},
		},
		{
			query:       `label_values(kubernetes_io_hostname)`,
			wantMetrics: nil,
			wantLabels:  []string{"kubernetes_io_hostname"},
		},
		{
			query:       `query_result(up{job="prometheus"})`,
			wantMetrics: []string{"up"},
			wantLabels:  []string{"job"},
		},
		{
			query:       `query_result(topk(1, max(node_filesystem_size_bytes{job="$job"})))`,
			wantMetrics: []string{"node_filesystem_size_bytes"},
			wantLabels:  []string{"job"},
		},
	}
	//q := "label_values(node_uname_info, job)"
	//wantMetrics := []string{"node_uname_info"}
	//wantLabels := []string{"job"}
	for _, tt := range tests {
		metrics, labels := ExtractMetricsAndLabelsFromTemplateQuery(tt.query)
		if !reflect.DeepEqual(metrics, tt.wantMetrics) {
			t.Errorf("metrics: got %v, want %v", metrics, tt.wantMetrics)
		}
		if !reflect.DeepEqual(labels, tt.wantLabels) {
			t.Errorf("labels: got %v, want %v", labels, tt.wantLabels)
		}

	}

}
