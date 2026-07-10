package utils

import (
	"regexp"
	"strings"

	"github.com/prometheus/prometheus/promql/parser"
)

// sanitizeExpr replaces Grafana template variables with valid PromQL literals
// so the expression can be parsed by the PromQL parser.
// e.g. [$__rate_interval] -> [5m], $node / ${job} -> placeholder
func sanitizeExpr(expr string) string {
	// Replace built-in range variables
	expr = strings.ReplaceAll(expr, "[$__interval]", "[5m]")
	expr = strings.ReplaceAll(expr, "[$__interval_ms]", "[5000]")
	expr = strings.ReplaceAll(expr, "[$__range]", "[5m]")
	expr = strings.ReplaceAll(expr, "[$__range_ms]", "[5000]")
	expr = strings.ReplaceAll(expr, "[$__rate_interval]", "[5m]")
	expr = strings.ReplaceAll(expr, "[$__timeFrom]", "[5m]")
	expr = strings.ReplaceAll(expr, "[$__timeTo]", "[5m]")

	// Replace custom template variables (e.g., $node, ${job}) with placeholder
	// Matches ${...} OR $identifier formats
	reTemplateVar := regexp.MustCompile(`\$\{[^}]+\}|\$[a-zA-Z_]\w*`)
	expr = reTemplateVar.ReplaceAllString(expr, "placeholder")

	return expr
}

// ExtractMetricNames extracts metric names from PromQL Expressions
// Handles template variables by sanitizing them before parsing
func ExtractMetricNames(expr string) []string {
	p := parser.NewParser(parser.Options{})
	ast, err := p.ParseExpr(sanitizeExpr(expr))
	if err != nil {
		return nil
	}
	var names []string
	parser.Inspect(ast, func(node parser.Node, _ []parser.Node) error {
		if vs, ok := node.(*parser.VectorSelector); ok {
			if vs.Name != "" {
				names = append(names, vs.Name)
			}
		}
		return nil
	})
	return names
}

// ExtractLabelNames extracts label names from PromQL Expressions
// Handles template variables by sanitizing them before parsing
func ExtractLabelNames(expr string) []string {
	p := parser.NewParser(parser.Options{})
	ast, err := p.ParseExpr(sanitizeExpr(expr))
	if err != nil {
		return []string{}
	}
	var names []string
	seen := make(map[string]bool)
	parser.Inspect(ast, func(node parser.Node, _ []parser.Node) error {
		if vs, ok := node.(*parser.VectorSelector); ok {
			for _, matcher := range vs.LabelMatchers {
				// Skip __name__ as it's an internal label
				if matcher.Name == "__name__" {
					continue
				}
				if !seen[matcher.Name] {
					names = append(names, matcher.Name)
					seen[matcher.Name] = true
				}
			}
		}
		return nil
	})
	if names == nil {
		return []string{}
	}
	return names
}
