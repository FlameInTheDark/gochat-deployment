# OpenObserve Assets

This directory vendors the repo-managed OpenObserve assets from the upstream
backend repository so deployment changes can carry dashboards and alerts
alongside the runtime config.

Files:

- `dashboards.yaml`: human-editable dashboard intent and panel inventory.
- `alerts.yaml`: human-editable alert inventory and thresholds.
- `bootstrap/dashboards/*.dashboard.json`: repo-managed dashboard bootstrap payloads.
- `bootstrap/alerts/alerts.seed.yaml`: alert bootstrap seed to bind to your org's alert destinations.

Run the bootstrap commands below from the deployment repo root.

## Local bootstrap

1. Start fresh with `docker compose down --remove-orphans`.
2. Bring the stack up with `docker compose up -d`.
3. Bootstrap OpenObserve assets with:
   `go run ./research/gochat/cmd/tools observability bootstrap --url http://localhost:5080 --org default --user root@example.com --password Complexpass#123 --dashboard-dir ./monitoring/openobserve/bootstrap/dashboards --alerts-file ./monitoring/openobserve/bootstrap/alerts/alerts.seed.yaml`
4. If you already created an OpenObserve alert destination, bind alerts during bootstrap:
   `go run ./research/gochat/cmd/tools observability bootstrap --url http://localhost:5080 --org default --user root@example.com --password Complexpass#123 --dashboard-dir ./monitoring/openobserve/bootstrap/dashboards --alerts-file ./monitoring/openobserve/bootstrap/alerts/alerts.seed.yaml --alert-destination-id <destination-name>`
5. Run the smoke check after the stack settles:
   `go run ./research/gochat/cmd/tools observability smoke --url http://localhost:5080 --org default --user root@example.com --password Complexpass#123`
6. Review legacy metric streams before deleting them:
   `go run ./research/gochat/cmd/tools observability cleanup --url http://localhost:5080 --org default --user root@example.com --password Complexpass#123`
7. Delete only the exporter-era metric streams when you are ready:
   `go run ./research/gochat/cmd/tools observability cleanup --url http://localhost:5080 --org default --user root@example.com --password Complexpass#123 --delete-legacy-streams`

## Hosted cluster bootstrap

Use the same bootstrap flow against the exposed deployment panel:

`go run ./research/gochat/cmd/tools observability bootstrap --url https://gochatobserve.anticode.dev --org default --user <email> --password <password> --dashboard-dir ./monitoring/openobserve/bootstrap/dashboards --alerts-file ./monitoring/openobserve/bootstrap/alerts/alerts.seed.yaml`

Alerts are not applied automatically. Create an OpenObserve alert destination
first, then rerun bootstrap with `--alert-destination-id <destination-name>`.

The upstream bootstrap tool can still report dashboard drift after a successful
upsert because OpenObserve rewrites part of the stored dashboard payload. When
that happens, confirm the dashboards in the UI or with the dashboards API
instead of assuming the import failed.

Local Compose does not run the SFU service anymore. Voice/SFU dashboards and
alerts remain in OpenObserve for externally deployed SFU nodes. In this stage
the SFU ships traces, metrics, and best-effort logs through the public
telemetry gateway and does not require a sidecar, daemon, or collector process
on the host.

See `docs/project/observability/ExternalSFU.md` for the standalone SFU
environment contract and `docs/project/observability/README.md` for the full
project observability documentation set.

## Streams

- Logs are written to the `gochat_logs` stream.
- Metrics are queryable as individual metric streams such as `gochat_http_server_requests`.
- PostgreSQL availability is reported through native service-side probe streams such as `gochat_postgres_probe_status`.
- Traces are queryable through the `gochat_traces` stream.

## Volume note

If OpenObserve shows a very large event count in local development, it is usually metric-heavy volume rather than log spam.

- The hottest live streams are typically histogram bucket streams such as `gochat_dependency_duration_bucket` and `gochat_http_server_duration_bucket`.
- Older local stacks may also still contain stale exporter-era metric families such as `pg_*`, `scrape_*`, `promhttp_*`, `postgres_exporter_*`, `citus_*`, `go_*`, `process_*`, `http_client_*`, and `up`.
- Use `go run ./cmd/tools observability cleanup ...` first in dry-run mode to confirm how much of the volume is historical.
- The shared Go runtime now defaults metric export to `60s` instead of `15s`. Override it with `OTEL_METRIC_EXPORT_INTERVAL` only when you need denser short-term debugging.

## Environment

- OTLP ingress for app services: `http://otel-collector:4318`
- Public OTLP ingress for external SFU nodes: `http://telemetry.<base-domain>`
- Deployment environment override: `GOCHAT_DEPLOYMENT_ENV`
- OpenObserve org env in compose: `OPENOBSERVE_ORG`
- Collector health endpoint: `http://localhost:13133/`
- Standalone SFU OTLP env:
  `OTEL_EXPORTER_OTLP_ENDPOINT`,
  `OTEL_EXPORTER_OTLP_HEADERS`,
  `OTEL_EXPORTER_OTLP_PROTOCOL`,
  `OTEL_METRIC_EXPORT_INTERVAL`

## Windows / Docker Desktop note

The local stack now ships container logs through Docker's `fluentd` logging
driver into the collector on `localhost:24224`. That is the supported path for
Docker Desktop because the Docker daemon emits logs itself and does not use the
collector container's internal DNS name. Go processes started directly on the
host still need their own OTLP/log shipping if you want them to appear in
OpenObserve.
