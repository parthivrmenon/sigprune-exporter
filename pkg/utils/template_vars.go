package utils

import (
	"slices"
	"strings"
)

const (
	labelValuesPrefix = "label_values("
	queryResultPrefix = "query_result("
)

// ExtractMetricsAndLabelsFromTemplateQuery extracts the metrics and labels
// referenced by a Grafana template variable query.
//
// Form 1: label_values(<selector>, <label>)
//
//	label_values(node_uname_info, job)                  -> metrics=[node_uname_info], labels=[job]
//	label_values(node_uname_info{job="$job"}, nodename) -> metrics=[node_uname_info], labels=[job, nodename]
//
// Form 2: label_values(<label>)
//
//	label_values(kubernetes_io_hostname)                -> metrics=[], labels=[kubernetes_io_hostname]
//
// Form 3: query_result(<promql>)
//
//	query_result(up{job="prometheus"})                  -> metrics=[up], labels=[job]
//
// Any other shape - metrics(<regex>), label_names(), or a string that is not a
// recognised wrapper at all - yields nothing on both counts.
//
// Labels come back sorted and deduplicated. Metrics come back in the order the
// PromQL parser reports them.
func ExtractMetricsAndLabelsFromTemplateQuery(q string) (metrics, labels []string) {
	var foundLabels []string
	var foundMetrics []string

	// trim spaces
	//q = strings.TrimSpace(q)

	switch {
	case strings.HasPrefix(q, labelValuesPrefix) && strings.HasSuffix(q, ")"):
		q = unwrap(q, labelValuesPrefix)

		selector, label := parseLabelValues(q)
		if selector != nil {
			extractedMetrics := ExtractMetricNames(*selector)
			extractedLabels := ExtractLabelNames(*selector)
			foundMetrics = append(foundMetrics, extractedMetrics...)
			foundLabels = append(foundLabels, extractedLabels...)

		}
		// Compacted labels to remove duplicates
		foundLabels = append(foundLabels, label)
		slices.Sort(foundLabels)
		foundLabels = slices.Compact(foundLabels)
		return foundMetrics, foundLabels

	case strings.HasPrefix(q, queryResultPrefix) && strings.HasSuffix(q, ")"):
		q = unwrap(q, queryResultPrefix)
		foundMetrics = ExtractMetricNames(q)
		foundLabels = append(foundLabels, ExtractLabelNames(q)...)
		slices.Sort(foundLabels)
		return foundMetrics, slices.Compact(foundLabels)

	// A non conformant shape gets nil instead of failing
	default:
		return nil, nil

	}
}

// parseLabelValues splits a label_values body into its selector and label.
// A label_values body takes one of two forms:
//  1. <selector>, <label>
//  2. <label>, ie. there is no selector
//
// Two things to consider:
//   - selector can be empty
//   - selector can contain commas, so split on the last comma only. That comma
//     is always the right one: a label name cannot contain a comma, and every
//     comma a selector can hold sits inside its {...}.
func parseLabelValues(s string) (selector *string, label string) {
	lastCommaIndex := strings.LastIndex(s, ",")

	// if there is no ',', the whole body is a label
	if lastCommaIndex == -1 {
		return nil, strings.TrimSpace(s)
	}
	selectorPart := strings.TrimSpace(s[:lastCommaIndex])
	label = strings.TrimSpace(s[lastCommaIndex+1:])
	return &selectorPart, label

}

// unwrap strips a wrapper's prefix and its closing ")". It slices without
// re-checking, so the caller must have matched both with HasPrefix/HasSuffix.
func unwrap(s, prefix string) string {
	return s[len(prefix) : len(s)-1]
}
