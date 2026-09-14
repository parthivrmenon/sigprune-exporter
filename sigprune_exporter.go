package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sigprune/pkg/scanner"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metric prefix
const namespace = "sigprune"

// Build Info
var (
	buildVersion  = "dev"
	buildRevision = "unknown"
)

type Exporter struct {
	scanner                             *scanner.Scanner
	sigpruneBuildInfoDesc               *prometheus.Desc
	sigpruneUpDesc                      *prometheus.Desc
	sigpruneUnusedMetricCardinalityDesc *prometheus.Desc
	sigpruneUnusedLabelCardinalityDesc  *prometheus.Desc
	sigpruneDashboardCountDesc          *prometheus.Desc
	sigpruneAlertRuleCountDesc          *prometheus.Desc
	sigpruneTotalTSDBMetricsDesc        *prometheus.Desc
	sigpruneTotalTSDBLabelsDesc         *prometheus.Desc
	sigpruneTotalUsedMetricsDesc        *prometheus.Desc
	sigpruneTotalUsedLabelsDesc         *prometheus.Desc
	sigpruneTotalUnusedMetricsDesc      *prometheus.Desc
	sigpruneTotalUnusedLabelsDesc       *prometheus.Desc
	sigpruneMetricsExportLimitDesc      *prometheus.Desc
	sigpruneLabelsExportLimitDesc       *prometheus.Desc
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
		sigpruneBuildInfoDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "build_info"),
			"Build information",
			[]string{"version", "revision"},
			nil,
		),
		sigpruneUpDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Was the last background scan successful",
			nil,
			nil,
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
		sigpruneDashboardCountDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "dashboard_count"),
			"Number of Grafana dashboards analyzed for metric/label references",
			nil,
			nil,
		),
		sigpruneAlertRuleCountDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "alert_rule_count"),
			"Number of Grafana alert rules analyzed for metric/label references",
			nil,
			nil,
		),
		sigpruneTotalTSDBMetricsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "total_tsdb_metrics"),
			"Total number of metrics retrieved from TSDB status API (limited by tsdb-metrics-limit)",
			nil,
			nil,
		),
		sigpruneTotalTSDBLabelsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "total_tsdb_labels"),
			"Total number of labels retrieved from TSDB status API (limited by tsdb-metrics-limit)",
			nil,
			nil,
		),
		sigpruneTotalUsedMetricsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "total_used_metrics"),
			"Number of unique metrics referenced in dashboards and alert rules",
			nil,
			nil,
		),
		sigpruneTotalUsedLabelsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "total_used_labels"),
			"Number of unique labels referenced in dashboards and alert rules",
			nil,
			nil,
		),
		sigpruneTotalUnusedMetricsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "total_unused_metrics"),
			"Number of unused metrics identified in the last successful background scan (capped by export limit)",
			nil,
			nil,
		),
		sigpruneTotalUnusedLabelsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "total_unused_labels"),
			"Number of unused labels identified in the last successful background scan (capped by export limit)",
			nil,
			nil,
		),
		sigpruneMetricsExportLimitDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "metrics_export_limit"),
			"Configured limit on number of unused metrics to export (metrics-limit flag)",
			nil,
			nil,
		),
		sigpruneLabelsExportLimitDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "labels_export_limit"),
			"Configured limit on number of unused labels to export (labels-limit flag)",
			nil,
			nil,
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.sigpruneBuildInfoDesc
	ch <- e.sigpruneUpDesc
	ch <- e.sigpruneUnusedMetricCardinalityDesc
	ch <- e.sigpruneUnusedLabelCardinalityDesc
	ch <- e.sigpruneDashboardCountDesc
	ch <- e.sigpruneAlertRuleCountDesc
	ch <- e.sigpruneTotalTSDBMetricsDesc
	ch <- e.sigpruneTotalTSDBLabelsDesc
	ch <- e.sigpruneTotalUsedMetricsDesc
	ch <- e.sigpruneTotalUsedLabelsDesc
	ch <- e.sigpruneTotalUnusedMetricsDesc
	ch <- e.sigpruneTotalUnusedLabelsDesc
	ch <- e.sigpruneMetricsExportLimitDesc
	ch <- e.sigpruneLabelsExportLimitDesc

}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneBuildInfoDesc,
		prometheus.GaugeValue,
		1,
		buildVersion,
		buildRevision,
	)

	// Reads the latest snapshot published by the background scan.
	snap, err := e.scanner.GetSnapshot()

	// Cold start: no successful scan yet. Emit an empty result
	if snap == nil {
		ch <- prometheus.MustNewConstMetric(e.sigpruneUpDesc, prometheus.GaugeValue, 0)
		return
	}

	// up reflects the the actual last ATTEMPT status;
	// The metrics emitted comes from the last SUCCESS. (i.e they can be stale)
	up := 1.0
	if err != nil {
		up = 0
	}
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneUpDesc,
		prometheus.GaugeValue,
		up,
	)

	for metric, perJobCount := range snap.UnusedMetrics {
		for job, count := range perJobCount {
			ch <- prometheus.MustNewConstMetric(
				e.sigpruneUnusedMetricCardinalityDesc,
				prometheus.GaugeValue,
				float64(count),
				job, metric,
			)

		}
	}

	for label, perJobCount := range snap.UnusedLabels {
		for job, count := range perJobCount {
			ch <- prometheus.MustNewConstMetric(
				e.sigpruneUnusedLabelCardinalityDesc,
				prometheus.GaugeValue,
				float64(count),
				job, label,
			)

		}
	}

	ch <- prometheus.MustNewConstMetric(
		e.sigpruneDashboardCountDesc,
		prometheus.GaugeValue,
		float64(snap.DashboardCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneAlertRuleCountDesc,
		prometheus.GaugeValue,
		float64(snap.AlertRuleCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneTotalTSDBMetricsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalTSDBMetrics),
	)
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneTotalTSDBLabelsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalTSDBLabels),
	)
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneTotalUsedMetricsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalUsedMetrics),
	)
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneTotalUsedLabelsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalUsedLabels),
	)
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneTotalUnusedMetricsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalUnusedMetrics),
	)
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneTotalUnusedLabelsDesc,
		prometheus.GaugeValue,
		float64(snap.TotalUnusedLabels),
	)
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneMetricsExportLimitDesc,
		prometheus.GaugeValue,
		float64(snap.MetricsExportLimit),
	)
	ch <- prometheus.MustNewConstMetric(
		e.sigpruneLabelsExportLimitDesc,
		prometheus.GaugeValue,
		float64(snap.LabelsExportLimit),
	)
}

func main() {
	fmt.Println("starting sigprune-exporter...")

	var (
		addr               = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
		grafanaURL         = flag.String("grafana", "http://localhost:3000", "Grafana URL")
		datasource         = flag.String("datasource", "", "Prometheus Datasource UID to collect metrics from")
		tsdbMetricsLimit   = flag.Int("tsdb-metrics-limit", 10000, "Number of top metrics to analyze from TSDB status")
		exportLimitMetrics = flag.Int("metrics-limit", 50, "Limit the number of metrics to export")
		exportLimitLabels  = flag.Int("labels-limit", 10, "Limit the number of labels to export")
		adminUser          = flag.String("user", "", "Username for authentication")
		adminPassword      = flag.String("password", "", "Password for authentication")
		apiKey             = flag.String("api-key", "", "Grafana API key for authentication")
		scanInterval       = flag.Duration("scan-interval", 5*time.Minute, "How often to rescan Grafana and Prometheus in the background")
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
	if *scanInterval <= 0 {
		log.Fatal("scan-interval must be a positive duration")
	}

	exporter := NewExporter(*tsdbMetricsLimit, *grafanaURL, *adminUser, *adminPassword, *apiKey, *datasource, *exportLimitMetrics, *exportLimitLabels)

	// Test Grafana connection before starting
	exporter.scanner.TestConnection()

	reg := prometheus.NewRegistry()

	reg.MustRegister(exporter)

	// Scans run in the background; Collect only reads the latest snapshot.
	go exporter.scanner.Run(*scanInterval)

	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
