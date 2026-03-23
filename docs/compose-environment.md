# Compose Inputs

`powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1` writes `.generated/compose/.env` and `.generated/compose/config/*.yaml`, runs `docker compose pull`, and then starts Compose.

The main operator inputs are:

- deployment type: `compose`
- storage mode: `minio` or `external`
- base domain or explicit public hosts
- secrets for auth, webhook JWT, PostgreSQL, etcd, and OpenSearch
- external S3 endpoint and credentials when `external` storage is selected
- image repository prefix and tag when you want something other than `ghcr.io/flameinthedark:*`

## Rendered Host Variables

The generated Compose env file includes:

- `APP_HOST`
- `API_HOST`
- `WS_HOST`
- `STORAGE_HOST`
- `MINIO_CONSOLE_HOST`
- `OPENSEARCH_DASHBOARDS_PORT`
- `OPENOBSERVE_PORT`
- `OTEL_COLLECTOR_HEALTH_PORT`
- `OTEL_COLLECTOR_FLUENTD_PORT`
- `OTEL_COLLECTOR_GRPC_PORT`
- `OTEL_COLLECTOR_HTTP_PORT`

`API_HOST` and `WS_HOST` are aligned to `APP_HOST` so the router matches the upstream backend compose for the automated stack:

- `/api/v1/*` on `APP_HOST`
- `/ws/*` on `APP_HOST` with `/ws` stripped

`APP_HOST`, `STORAGE_HOST`, and `MINIO_CONSOLE_HOST` must resolve to the deployment machine.

By default, `APP_HOST` is the base domain itself. Example:

- UI: `http://example.com`
- API: `http://example.com/api/v1`
- WS: `ws://example.com/ws/subscribe`

SFU is not deployed by Compose. If you need voice, deploy SFU separately and point it at the generated webhook and etcd settings.

## Storage Variables

When bundled MinIO is enabled, the env file also carries:

- `MINIO_ROOT_USER`
- `MINIO_ROOT_PASSWORD`
- `MINIO_BUCKET`

When external S3 is selected, MinIO is not started and the generated service configs point directly at the external endpoint instead.

## Images And Migrations

Application containers default to the upstream GHCR images:

- `ghcr.io/flameinthedark/gochat-api`
- `ghcr.io/flameinthedark/gochat-auth`
- `ghcr.io/flameinthedark/gochat-attachments`
- `ghcr.io/flameinthedark/gochat-ws`
- `ghcr.io/flameinthedark/gochat-webhook`
- `ghcr.io/flameinthedark/gochat-indexer`
- `ghcr.io/flameinthedark/gochat-embedder`
- `ghcr.io/flameinthedark/gochat-react`

Database migrations run from the pulled `ghcr.io/<owner>/gochat-migrations:<backend-tag>` image by default. The generated env file provides `PG_ADDRESS` and `CASSANDRA_ADDRESS`, and the Compose stack runs a single `migrations` container before the application services start.

The Compose stack also publishes OpenSearch Dashboards on `http://<host>:5601` by default. Override that with `OPENSEARCH_DASHBOARDS_PORT` in the generated env file if needed.

The Compose stack also publishes OpenObserve on `http://<host>:5080` by default. Override that with `OPENOBSERVE_PORT` in the generated env file if needed.

## Observability Variables

Compose mirrors the upstream backend local observability model:

- app services send traces and metrics to `http://otel-collector:4318`
- Docker ships container logs through the Fluentd logging driver to `localhost:24224`
- the collector forwards logs, traces, and metrics into OpenObserve

The generated env file also includes:

- `OPENOBSERVE_ROOT_EMAIL`
- `OPENOBSERVE_ROOT_PASSWORD`
- `OPENOBSERVE_ORG`
- `OPENOBSERVE_BASIC_AUTH`
- `OPENOBSERVE_LOG_STREAM`
- `OPENOBSERVE_METRIC_STREAM`
- `OPENOBSERVE_TRACE_STREAM`

Collector endpoints exposed on the host:

- health: `http://<host>:13133/`
- OTLP gRPC: `<host>:4317`
- OTLP HTTP: `http://<host>:4318`
- Fluentd ingress for Docker logs: `<host>:24224`

OpenObserve dashboards and alert bootstrap assets are vendored in this repo under `monitoring/openobserve/`.
Apply them from the deployment repo root with `go run ./research/gochat/cmd/tools observability bootstrap --dashboard-dir ./monitoring/openobserve/bootstrap/dashboards --alerts-file ./monitoring/openobserve/bootstrap/alerts/alerts.seed.yaml ...`.

## Service Config Files

The deploy wrapper renders:

- `api_config.yaml`
- `auth_config.yaml`
- `attachments_config.yaml`
- `ws_config.yaml`
- `webhook_config.yaml`
- `indexer_config.yaml`
- `embedder_config.yaml`

These are mounted into the matching containers from `.generated/compose/config/`.
