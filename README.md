# sigprune-exporter

**sigprune** *(n.)* — *"Signal Pruner"*

A Prometheus exporter for detecting unused metrics and labels within a Grafana datasource.

A metric/label is considered 'unused' if:
- it does not appear in any Grafana Dashboard
- it does not appear in any Grafana Alert Rule

The exporter allows you to visualize these unused metrics and labels in a Grafana dashboard like the one below:

![Unused Metrics And Labels dashboard](docs/images/dashboard.png)

A sample dashboard is included in `docker/grafana/provisioning/dashboards/sigprune.json` and is automatically provisioned when using the local Docker stack.

### Importing the dashboard

Import `docker/grafana/provisioning/dashboards/sigprune.json` from this repo, then pick your Prometheus datasource.

**Note:** For the dashboard to work correctly the job that scrapes sigprune-exporter must set `honor_labels: true`

>
> ```yaml
> scrape_configs:
>   - job_name: "sigprune"
>     honor_labels: true
>     static_configs:
>       - targets: ["localhost:8080"]
> ```

`sigprune_unused_metric_cardinality` and `sigprune_unused_label_cardinality` carry a `job` label naming the original job that collected the series. However, without `honor_labels` Prometheus overwrites it with `sigprune`, so every series reports the same job and the per-job breakdowns collapse into one number.


## Installation

### Download a release binary

Pre-built binaries for Linux (amd64) and macOS (arm64 and amd64) are attached to each [release](https://github.com/parthivrmenon/sigprune-exporter/releases). Download the one for your platform, make it executable, and run it:

```bash
chmod +x sigprune-exporter-linux-amd64
./sigprune-exporter-linux-amd64 --help
```

### Build from source

**Requires** [Go](https://go.dev/dl/) 1.25+

```bash
git clone https://github.com/parthivrmenon/sigprune-exporter.git
cd sigprune-exporter
go build \
  -ldflags "-X main.buildVersion=v0.1.0 -X main.buildRevision=$(git rev-parse --short HEAD)" \
  -o sigprune-exporter .
```

This produces a `sigprune-exporter` binary in the current directory. The `buildVersion` and `buildRevision` flags are injected at build time and exposed via the `sigprune_build_info` metric.

```bash
./sigprune-exporter --help
```

## Running the exporter

**Using Basic auth:**
```bash
./sigprune-exporter \
  -grafana http://localhost:3000 \
  -user admin \
  -password admin \
  -datasource <prometheus-datasource-uid>
```

**Using an API key:**
```bash
./sigprune-exporter \
  -grafana http://localhost:3000 \
  -api-key <grafana-api-key> \
  -datasource <prometheus-datasource-uid>
```

See [Configuration](#configuration) for the full list.

## Configuration

| Argument | Description | Default |
|---|---|---|
| `-grafana` | Grafana URL | `http://localhost:3000` |
| `-user` | Username for basic auth | *(required with `-password`, unless using `-api-key`)* |
| `-password` | Password for basic auth | *(required with `-user`, unless using `-api-key`)* |
| `-api-key` | Grafana API key for authentication | *(required, unless using `-user`/`-password`)* |
| `-datasource` | Prometheus datasource UID to collect metrics from | *(required)* |
| `-listen-address` | Address to listen on for HTTP requests | `:8080` |
| `-tsdb-metrics-limit` | Number of top metrics to analyze from TSDB status | `10000` |
| `-metrics-limit` | Limit the number of unused metrics to export | `50` |
| `-labels-limit` | Limit the number of unused labels to export | `10` |
| `-scan-interval` | How often to rescan Grafana and Prometheus in the background. Takes a Go duration such as `30s`, `5m` or `1h30m` | `5m` |
| `-grafana-timeout` | Timeout for each HTTP request to Grafana, including Prometheus queries sent through Grafana's datasource proxy. Takes a Go duration | `60s` |

> **Note:** 
> - Increasing `labels-limit` makes each background scan take longer. Each label adds one sequential Prometheus API call to fetch per-job series counts.
> - Prometheus queries go through Grafana's datasource proxy, which has its own timeout which is 30 seconds by default. Setting `-grafana-timeout` above that has no effect on those queries unless you also raise Grafana's setting as well.

### How scanning works

The exporter scans Grafana and Prometheus in the background, once at startup, then every `-scan-interval`. 

A scrape of `/metrics` serves the result of the latest successful scan and never waits for a scan to run, so scrape latency stays small regardless of how many dashboards and alert rules Grafana has.

This has a few consequences:
- **Data can be up to one `-scan-interval` old** (longer if a scan fails or takes longer than the interval).
- **Before the first scan completes**, `/metrics` exposes only `sigprune_build_info` and `sigprune_up 0`.
- **If a scan fails**, the exporter keeps serving the last successful result and sets `sigprune_up` to `0` until a scan succeeds again.

Dashboards and alert rules usually change over hours, not minutes, so the `5m` default is a reasonable. Try to avoid very short intervals against a production Grafana, since every scan fetches every dashboard.


## Metrics Reference

| Metric | Type | Labels | Description |
|---|---|---|---|
| `sigprune_build_info` | Gauge | `version`, `revision` | Always `1`. Exposes build version and git revision of the running binary |
| `sigprune_up` | Gauge | — | `1` if the most recent background scan succeeded, `0` if it failed or no scan has completed yet. When `0` after a failure, the other metrics still show the last successful scan (see [How scanning works](#how-scanning-works)) |
| `sigprune_unused_metric_cardinality` | Gauge | `job`, `metric` | Series count of a top-K unused metric (not referenced in any dashboard or alert rule) |
| `sigprune_unused_label_cardinality` | Gauge | `job`, `label` | Series count of a top-K unused label (not referenced in any dashboard or alert rule) |
| `sigprune_dashboard_count` | Gauge | — | Number of Grafana dashboards analyzed |
| `sigprune_alert_rule_count` | Gauge | — | Number of Grafana alert rules analyzed |
| `sigprune_total_tsdb_metrics` | Gauge | — | Number of metric names returned by the Prometheus TSDB status API, capped at `-tsdb-metrics-limit`. If it equals the limit, Prometheus likely has more metrics than were analyzed |
| `sigprune_total_tsdb_labels` | Gauge | — | Number of label names returned by the Prometheus TSDB status API, capped at `-tsdb-metrics-limit` |
| `sigprune_total_used_metrics` | Gauge | — | Number of distinct metric names referenced in dashboards and alert rules. A metric used in 20 panels counts once |
| `sigprune_total_used_labels` | Gauge | — | Number of distinct label names referenced in dashboards and alert rules |
| `sigprune_total_unused_metrics` | Gauge | — | Number of unused metrics exported, capped at `-metrics-limit`. Not a true total: if it equals `sigprune_metrics_export_limit`, there are likely more unused metrics |
| `sigprune_total_unused_labels` | Gauge | — | Number of unused labels exported, capped at `-labels-limit`. Not a true total: if it equals `sigprune_labels_export_limit`, there are likely more unused labels |
| `sigprune_metrics_export_limit` | Gauge | — | Configured value of `-metrics-limit` |
| `sigprune_labels_export_limit` | Gauge | — | Configured value of `-labels-limit` |

Apart from `sigprune_build_info` and `sigprune_up`, every metric comes from the last successful background scan and is not exposed until the first scan completes (see [How scanning works](#how-scanning-works)).
