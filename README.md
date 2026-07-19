# sigprune-exporter

A Prometheus exporter for detecting *unused* metrics and labels in a Grafana datasources.

A metric/label is deemed 'unused' if:
- it does not appear in any Grafana Dashboard
- it does not appear in any Grafana Alert Rule


*Currently, the exporter only supports Prometheus datasources*


## Installation

**Prerequisites:** [Go](https://go.dev/dl/) 1.21+

```bash
git clone https://github.com/parthivrmenon/sigprune-exporter.git
cd sigprune-exporter
go build -o sigprune-exporter .
```

This produces a `sigprune-exporter` binary in the current directory.

```bash
./sigprune-exporter --help
```

## Configuration

| Argument | Description | Default |
|---|---|---|
| `-grafanaURL` | Grafana URL | `http://localhost:3000` |
| `-user` | Username for basic auth | |
| `-password` | Password for basic auth | |
| `-api-key` | Grafana API key for authentication | |
| `-datasource` | Prometheus datasource UID to collect metrics from | *(required)* |
| `-listen-address` | Address to listen on for HTTP requests | `:8080` |
| `-tsdb-metrics-limit` | Number of top metrics to analyze from TSDB status | `10000` |
| `-metrics-limit` | Limit the number of unused metrics to export | `50` |
| `-labels-limit` | Limit the number of unused labels to export | `50` |
