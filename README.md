# sigprune-exporter

An exporter for Prometheus/Grafana Metric pruning.

## Quick Start

```bash
# Start local Grafana and Prometheus
docker compose -f docker/docker-compose.yml up -d

# Access Grafana at http://localhost:3000 (admin/admin)
# Access Prometheus at http://localhost:9090 (no auth required)
```

Run the following command to start the sigprune service:
```bash
go run sigprune_exporter.go -user admin -password admin -datasource PBFA97CFB590B2093
```

## Shuting down
```bash
docker compose -f docker/docker-compose.yml down
```

To fully reset (wipes Grafana storage volume, re-provisions dashboards from scratch):
```bash
docker compose -f docker/docker-compose.yml down -v
```


