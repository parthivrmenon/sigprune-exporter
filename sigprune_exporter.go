package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sigprune/pkg/grafana"
	"sigprune/pkg/utils"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// A Gauge Metric to represent unused metric names and cardinality
var unusedMetricCardinality = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "unused_metric_cardinality",
		Help: "Series count of a top-K unused metric",
	},
	[]string{"job", "metric"},
)

// Collector collects metrics from Prometheus and Grafana
type Collector struct {
	topK          int
	grafanaURL    string
	adminUser     string
	adminPassword string
	datasource    string
}

func NewCollector(
	topK int, grafanaURL string,
	adminUser string,
	adminPassword string,
	datasource string,
) *Collector {
	return &Collector{
		topK:          topK,
		grafanaURL:    grafanaURL,
		adminUser:     adminUser,
		adminPassword: adminPassword,
		datasource:    datasource,
	}
}

func (c *Collector) collectGrafanaDashboardMetrics() []string {
	var metrics []string
	g := grafana.NewClient(c.grafanaURL, c.adminUser, c.adminPassword)
	dashboards, err := g.GetDashboards()
	if err != nil {
		log.Fatal(err)
	}

	for _, dashboard := range dashboards {
		dashboardResponse, err := g.GetDashboardByUID(dashboard.UID)
		if err != nil {
			log.Fatal(err)
		}
		panelExprs := grafana.GetDashboardPanelExprs(*dashboardResponse)
		for _, expr := range panelExprs {
			metricNames := utils.ExtractMetricNames(expr)
			metrics = append(metrics, metricNames...)
		}
	}
	return metrics
}

func (c *Collector) collectGrafanaAlertRuleMetrics() []string {
	var metrics []string
	g := grafana.NewClient(c.grafanaURL, c.adminUser, c.adminPassword)
	alerts, err := g.GetAlertRules()
	if err != nil {
		log.Fatal(err)
	}
	for _, alert := range alerts {
		alertRule, err := g.GetAlertRuleByUID(alert.UID)
		if err != nil {
			log.Fatal(err)
		}
		alertExprs := grafana.GetAlertRuleExprs(*alertRule)
		for _, expr := range alertExprs {
			metricNames := utils.ExtractMetricNames(expr)
			metrics = append(metrics, metricNames...)
		}
	}
	return metrics

}

func (c *Collector) getTopKMetricsFromPrometheus() grafana.TSDBStatusSeriesCountByMetricName {
	g := grafana.NewClient(c.grafanaURL, c.adminUser, c.adminPassword)
	return g.GetTopKMetricsFromPrometheusDatasourceTSDBStatus(c.datasource, c.topK)

}

func (c *Collector) getJobsForMetric(metric string) map[string]int64 {
	g := grafana.NewClient(c.grafanaURL, c.adminUser, c.adminPassword)
	return g.GetPrometheusJobsForMetric(c.datasource, metric)
}
func (c *Collector) Collect() {

	// get top K metrics from Prometheus
	tsdbStatus := c.getTopKMetricsFromPrometheus()

	// get grafana dashboards metrics
	usedMetrics := c.collectGrafanaDashboardMetrics()
	fmt.Println("Got", len(usedMetrics), "used metrics from dashboards")

	// get grafana alert rule metrics
	alertMetrics := c.collectGrafanaAlertRuleMetrics()
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
		jobs := c.getJobsForMetric(metric)
		for job, count := range jobs {
			unusedMetricCardinality.WithLabelValues(job, metric).Set(float64(count))
		}
	}

}

func main() {
	fmt.Println("starting sigprune-exporter...")

	var (
		addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
		// promURL       = flag.String("prometheus", "http://localhost:9090", "Prometheus URL")
		grafanaURL    = flag.String("grafana", "http://localhost:3000", "Grafana URL")
		datasource    = flag.String("datasource", "", "Prometheus Datasource UID to collect metrics from")
		topK          = flag.Int("top-k", 100, "Number of top metrics to collect")
		adminUser     = flag.String("user", "", "Username for authentication")
		adminPassword = flag.String("password", "", "Password for authentication")
	)
	flag.Parse()
	if *datasource == "" {
		log.Fatal("datasourceUID for a Prometheus type datasource MUST be provided")
	}
	if *adminUser == "" || *adminPassword == "" {
		log.Fatal("Username and password are required to access Grafana")
	}

	collector := NewCollector(*topK, *grafanaURL, *adminUser, *adminPassword, *datasource)

	// Test Grafana connection before starting
	g := grafana.NewClient(*grafanaURL, *adminUser, *adminPassword)
	if err := g.TestConnection(); err != nil {
		log.Fatal("Failed to connect to Grafana:", err)
	}

	reg := prometheus.NewRegistry()

	reg.MustRegister(
		unusedMetricCardinality,
	)

	// Collect metrics before starting server
	go collector.Collect()

	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
