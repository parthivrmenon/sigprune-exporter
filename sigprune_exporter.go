package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sigprune/pkg/scanner"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metric prefix
const namespace = "sigprune"

type Exporter struct {
	scanner                             *scanner.Scanner
	sigpruneUnusedMetricCardinalityDesc *prometheus.Desc
	sigpruneUnusedLabelCardinalityDesc  *prometheus.Desc
}

func NewExporter(tsdbMetricsLimit int, grafanaURL string, adminUser string, adminPassword string, apiKey string, datasource string, exportLimitMetrics int, exportLimitLabels int) *Exporter {
	return &Exporter{
		scanner: scanner.NewScanner(
			tsdbMetricsLimit,
			grafanaURL,
			adminUser,
			adminPassword,
			apiKey,
			datasource,
			exportLimitMetrics,
			exportLimitLabels,
		),
		sigpruneUnusedMetricCardinalityDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "unused_metric_cardinality"),
			"Series count of a top-K unused metric",
			[]string{"job", "metric"},
			nil,
		),
		sigpruneUnusedLabelCardinalityDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "unused_label_cardinality"),
			"Series count of a top-K unused label",
			[]string{"job", "label"},
			nil,
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.sigpruneUnusedMetricCardinalityDesc
	ch <- e.sigpruneUnusedLabelCardinalityDesc

}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	unusedMetricsAndLabels := e.scanner.GetUnusedMetricsAndLabels()

	for _, metric := range unusedMetricsAndLabels.UnusedMetrics {
		jobs := e.scanner.GetJobsForMetric(metric)
		for job, count := range jobs {
			ch <- prometheus.MustNewConstMetric(
				e.sigpruneUnusedMetricCardinalityDesc,
				prometheus.GaugeValue,
				float64(count),
				job, metric,
			)
		}
	}

	for _, label := range unusedMetricsAndLabels.UnusedLabels {
		jobs := e.scanner.GetJobsForLabel(label)
		for job, count := range jobs {
			ch <- prometheus.MustNewConstMetric(
				e.sigpruneUnusedLabelCardinalityDesc,
				prometheus.GaugeValue,
				float64(count),
				job, label,
			)
		}
	}

}

func main() {
	fmt.Println("starting sigprune-exporter...")

	var (
		addr               = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
		grafanaURL         = flag.String("grafana", "http://localhost:3000", "Grafana URL")
		datasource         = flag.String("datasource", "", "Prometheus Datasource UID to collect metrics from")
		tsdbMetricsLimit   = flag.Int("tsdb-metrics-limit", 10000, "Number of top metrics to analyze from TSDB status")
		exportLimitMetrics = flag.Int("metrics-limit", 50, "Limit the number of metrics to export")
		exportLimitLabels  = flag.Int("labels-limit", 50, "Limit the number of labels to export")
		adminUser          = flag.String("user", "", "Username for authentication")
		adminPassword      = flag.String("password", "", "Password for authentication")
		apiKey             = flag.String("api-key", "", "Grafana API key for authentication")
	)
	flag.Parse()
	if *datasource == "" {
		log.Fatal("datasourceUID for a Prometheus type datasource MUST be provided")
	}
	usingBasicAuth := *adminUser != "" || *adminPassword != ""
	usingAPIKey := *apiKey != ""

	if usingBasicAuth && usingAPIKey {
		log.Fatal("Provide either --api-key or --user/--password, not both")
	}
	if !usingBasicAuth && !usingAPIKey {
		log.Fatal("Authentication required: provide --api-key or --user and --password")
	}
	if usingBasicAuth && (*adminUser == "" || *adminPassword == "") {
		log.Fatal("Both --user and --password are required when using basic auth")
	}
	if *exportLimitMetrics <= 0 {
		log.Fatal("metrics-limit must be a positive integer")
	}
	if *exportLimitLabels <= 0 {
		log.Fatal("labels-limit must be a positive integer")
	}

	exporter := NewExporter(*tsdbMetricsLimit, *grafanaURL, *adminUser, *adminPassword, *apiKey, *datasource, *exportLimitMetrics, *exportLimitLabels)

	// Test Grafana connection before starting
	exporter.scanner.TestConnection()

	reg := prometheus.NewRegistry()

	reg.MustRegister(exporter)

	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
