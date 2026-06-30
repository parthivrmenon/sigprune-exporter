package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sigprune/pkg"
	"strconv"
	"strings"

	// "github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/prometheus/promql/parser"
)

var (
	addr          = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	promURL       = flag.String("prometheus", "http://localhost:9090", "Prometheus URL")
	grafanaURL    = flag.String("grafana", "http://localhost:3000", "Grafana URL")
	topK          = flag.Int("top-k", 100, "Number of top metrics to collect")
	adminUser     = flag.String("user", "", "Username for authentication")
	adminPassword = flag.String("password", "", "Password for authentication")
)

var unusedMetricCardinality = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "unused_metric_cardinality",
		Help: "Series count of a top-K unused metric",
	},
	[]string{"job", "metric"},
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

/*
Extract Metric names from PromQL Expressions
Handles template variables
*/
func extractMetricNames(expr string) []string {
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

func getGrafanaDashboardMetrics(grafanaUrl string, username string, password string) []string {
	var metrics []string
	g := pkg.NewClient(grafanaUrl, username, password)
	g.TestConnection()

	dashboards, err := g.GetDashboards()
	if err != nil {
		log.Fatal(err)
	}

	for _, dashboard := range dashboards {
		dashboardResponse, err := g.GetDashboardByUID(dashboard.UID)
		if err != nil {
			log.Fatal(err)
		}
		panelExprs := pkg.GetDashboardPanelExprs(*dashboardResponse)
		for _, expr := range panelExprs {
			metricNames := extractMetricNames(expr)
			metrics = append(metrics, metricNames...)
		}
	}
	return metrics
}

func getGrafanaAlertRuleMetrics(grafanaUrl string, username string, password string) []string {
	var metrics []string
	g := pkg.NewClient(grafanaUrl, username, password)
	g.TestConnection()
	alerts, err := g.GetAlertRules()
	if err != nil {
		log.Fatal(err)
	}
	for _, alert := range alerts {
		alertRule, err := g.GetAlertRuleByUID(alert.UID)
		if err != nil {
			log.Fatal(err)
		}
		alertExprs := pkg.GetAlertRuleExprs(*alertRule)
		for _, expr := range alertExprs {
			metricNames := extractMetricNames(expr)
			metrics = append(metrics, metricNames...)
		}
	}
	return metrics

}

/*
Get Top K Metrics from Prometheus
Query: /api/v1/status/tsdb?limit=<topK>
*/
type MetricCount struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type TSDBStatus struct {
	Data struct {
		SeriesCountByMetricName []MetricCount `json:"seriesCountByMetricName"`
	} `json:"data"`
}

func getTopKMetricsFromPrometheus(promUrl string, limit int) TSDBStatus {
	req, err := http.NewRequest("GET", promUrl+"/api/v1/status/tsdb?limit="+fmt.Sprintf("%d", limit), nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var tsdbStatus TSDBStatus
	if err := json.Unmarshal(result, &tsdbStatus); err != nil {
		log.Fatal(err)
	}

	return tsdbStatus
}

/*
Get Jobs associated with a metric
Query: count by (job) (metricName)
*/
type QueryResponse struct {
	Data struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]interface{}    `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func getJobsForMetric(metricName string) map[string]int64 {
	req, err := http.NewRequest("GET", *promURL+"/api/v1/query", nil)
	if err != nil {
		log.Fatal(err)
	}
	params := req.URL.Query()
	params.Add("query", fmt.Sprintf("count by (job) (%s)", metricName))
	req.URL.RawQuery = params.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var queryResponse QueryResponse
	if err := json.Unmarshal(result, &queryResponse); err != nil {
		log.Fatal(err)
	}

	jobs := make(map[string]int64)
	for _, r := range queryResponse.Data.Result {
		job := r.Metric["job"]
		countStr := r.Value[1].(string) // value[1] is the count as a string
		count, _ := strconv.ParseInt(countStr, 10, 64)
		jobs[job] = count
	}
	return jobs

}

/*
Collect Metrics from Prometheus
*/
func collectMetrics() {
	// get top K metrics from Prometheus
	tsdbStatus := getTopKMetricsFromPrometheus(*promURL, *topK)

	// get grafana dashboards metrics
	usedMetrics := getGrafanaDashboardMetrics(*grafanaURL, *adminUser, *adminPassword)
	fmt.Println("Got", len(usedMetrics), "used metrics from dashboards")

	// get grafana alert rule metrics
	alertMetrics := getGrafanaAlertRuleMetrics(*grafanaURL, *adminUser, *adminPassword)
	fmt.Println("Got", len(alertMetrics), "used metrics from alert rules")

	// combine used metrics
	usedMetrics = append(usedMetrics, alertMetrics...)

	// create a used metrics Set for lookup
	usedSet := make(map[string]bool, len(usedMetrics))
	for _, m := range usedMetrics {
		usedSet[m] = true
	}

	// Filter out used metrics
	var unusedMetrics []string
	for _, metric := range tsdbStatus.Data.SeriesCountByMetricName {
		if !usedSet[metric.Name] {
			unusedMetrics = append(unusedMetrics, metric.Name)
		}
	}
	fmt.Println("Got", len(unusedMetrics), "unused metrics")

	// register unused metrics
	for _, metric := range unusedMetrics {
		jobs := getJobsForMetric(metric)
		for job, count := range jobs {
			unusedMetricCardinality.WithLabelValues(job, metric).Set(float64(count))

		}
	}

}

func main() {
	fmt.Println("Starting sigprune exporter...")
	flag.Parse()
	if *adminUser == "" || *adminPassword == "" {
		log.Fatal("Username and password are required to access Grafana")
	}

	reg := prometheus.NewRegistry()

	// Add go runtime metrics and process collectors later
	// For now, keep them commented out
	reg.MustRegister(
		// collectors.NewGoCollector(),
		// collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		unusedMetricCardinality,
	)

	// Set a value
	// TODO: Implement 'ticker' to collect metrics periodically
	go collectMetrics()

	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
