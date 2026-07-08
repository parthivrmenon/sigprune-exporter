package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sigprune/pkg/scraper"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metric prefix
const namespace = "sigprune"

type Exporter struct {
	scraper                             *scraper.Scraper
	sigpruneUnusedMetricCardinalityDesc *prometheus.Desc
}

func NewExporter(tsdbMetricsLimit int, grafanaURL string, adminUser string, adminPassword string, datasource string) *Exporter {
	return &Exporter{
		scraper: scraper.NewScraper(
			tsdbMetricsLimit,
			grafanaURL,
			adminUser,
			adminPassword,
			datasource,
		),
		sigpruneUnusedMetricCardinalityDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "unused_metric_cardinality"),
			"Series count of a top-K unused metric",
			[]string{"job", "metric"},
			nil,
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.sigpruneUnusedMetricCardinalityDesc
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	unusedMetrics := e.scraper.GetUnusedMetrics()

	for _, metric := range unusedMetrics {
		jobs := e.scraper.GetJobsForMetric(metric)
		for job, count := range jobs {
			ch <- prometheus.MustNewConstMetric(
				e.sigpruneUnusedMetricCardinalityDesc,
				prometheus.GaugeValue,
				float64(count),
				job, metric,
			)
		}
	}

}

func main() {
	fmt.Println("starting sigprune-exporter...")

	var (
		addr             = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
		grafanaURL       = flag.String("grafana", "http://localhost:3000", "Grafana URL")
		datasource       = flag.String("datasource", "", "Prometheus Datasource UID to collect metrics from")
		tsdbMetricsLimit = flag.Int("tsdb-metrics-limit", 10, "Number of top metrics to analyze from TSDB status")
		adminUser        = flag.String("user", "", "Username for authentication")
		adminPassword    = flag.String("password", "", "Password for authentication")
	)
	flag.Parse()
	if *datasource == "" {
		log.Fatal("datasourceUID for a Prometheus type datasource MUST be provided")
	}
	if *adminUser == "" || *adminPassword == "" {
		log.Fatal("Username and password are required to access Grafana")
	}

	exporter := NewExporter(*tsdbMetricsLimit, *grafanaURL, *adminUser, *adminPassword, *datasource)

	// Test Grafana connection before starting
	exporter.scraper.TestConnection()

	reg := prometheus.NewRegistry()

	reg.MustRegister(exporter)

	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
