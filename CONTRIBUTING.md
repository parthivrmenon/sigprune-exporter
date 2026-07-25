# Contributing to sigprune-exporter

Contributions are welcome. This project is in early development so please open an issue before starting significant work. PRs submitted without a prior issue or discussion will be closed.

## Reporting Issues

Open a [GitHub Issue](https://github.com/parthivrmenon/sigprune-exporter/issues) for:
- Bug reports
- Feature requests or suggestions

## Submitting a Pull Request

1. Open an issue first to discuss the change
2. Fork the repository and create a branch from `main`
3. Make your changes and ensure tests pass: `go test ./...`
4. Add or update tests as needed
5. Submit a pull request referencing the issue

## Development Setup

```bash
git clone https://github.com/parthivrmenon/sigprune-exporter.git
cd sigprune-exporter
go build ./...
go test ./...
```

To run a local Grafana + Prometheus stack for manual testing:

```bash
# Start
docker compose -f docker/docker-compose.yml up -d

# Stop
docker compose -f docker/docker-compose.yml down

# Full reset (wipes volumes)
docker compose -f docker/docker-compose.yml down -v
```

Grafana is available at `http://localhost:3000` (admin/admin) and Prometheus at `http://localhost:9090`.

