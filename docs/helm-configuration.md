# Helm Inputs

For Kubernetes deployments, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1 -DeploymentType helm
```

The wrapper generates `.generated/helm/values.generated.yaml`, `.generated/helm/scylla-values.generated.yaml`, and `.generated/helm/yugabyte-values.generated.yaml`, then installs the database charts and the GoChat chart from [helm/gochat](/H:/Projects/Deployment/gochat-deployment/helm/gochat).

## Required Operator Inputs

- image repository prefix and tag for pulled images
- base domain or explicit public hosts
- namespace and release name
- storage mode: bundled MinIO or external S3
- ingress controller strategy:
  - existing ingress controller
  - bundled Traefik
- ScyllaDB and YugabyteDB namespaces, release names, node counts, replication factors, storage, and resources
- YugabyteDB YSQL service host, port, user, password, database, and sslmode

## YugabyteDB YSQL

GoChat uses YugabyteDB YSQL as the active relational store. The deployer renders values for the official YugabyteDB Helm chart before applying the GoChat chart. The default GoChat values expect a YugabyteDB release named `yb` in namespace `gochat-yb`, which exposes YSQL through `yb-tservers.gochat-yb.svc.cluster.local:5433`.

## ScyllaDB

The deployer renders values for the Scylla Operator chart and the ScyllaDB cluster chart. Defaults match a 3-node ScyllaDB cluster in namespace `gochat-scylla`, exposed through `gochat-scylla-client.gochat-scylla.svc.cluster.local`. The GoChat chart includes a keyspace hook that uses `NetworkTopologyStrategy` with the configured datacenter and replication factor.

For extreme highload, keep `relational.yugabyte.colocation=false`. This creates a non-colocated database so large guild, channel, membership, invite, and role tables can split and rebalance across tablets and nodes instead of being pinned into one colocated tablet.

The GoChat chart includes a `yugabyte-init` Helm hook that:

- waits for YSQL,
- creates the `gochat` database only if it is missing,
- uses `CREATE DATABASE ... WITH COLOCATION = false`,
- reports existing database colocation state,
- never drops or recreates an existing database.

## Storage Notes

Bundled MinIO mode enables:

- `minio.enabled=true`
- public bucket bootstrap
- permissive CORS for presigned browser uploads
- storage and console ingress hosts

External S3 mode disables bundled MinIO and writes the external endpoint details directly into the attachments config.

Important: for bundled MinIO, the configured public storage host must be reachable from both the browser and the running pods, because GoChat presigns uploads against that public endpoint.

## Router Shape

The Helm deployment is configured to match the upstream backend compose router:

- `<domain>/api/v1/*`
- `<domain>/ws/*`
- `telemetry.<domain>/*`

When bundled Traefik is enabled, the chart creates the strip-prefix resources needed for `/ws`. If you use another ingress controller, you need to reproduce that rewrite behavior yourself.

SFU and stream services are intentionally excluded from the Helm chart. If you need voice or screen sharing, deploy them separately with the required direct networking and reuse the automated `webhook`, `etcd`, and telemetry gateway components for registration plus observability. Stream nodes must use the same region ids as voice nodes (`global`, `eu`, `us-east`) because the API selects stream discovery entries only from the effective voice region.

Example app endpoints for `example.com`:

- UI: `https://example.com`
- API: `https://example.com/api/v1`
- WS: `wss://example.com/ws`
- Telemetry Gateway: `https://telemetry.example.com`

## Values Rendered By The Wrapper

The generated override file pins:

- image repository and tag for every application container
- the migrations container image, which defaults to `gochat-migrations:<backend-tag>`
- `routing.appHost`
- shared OTEL app env, including `OTEL_METRIC_EXPORT_INTERVAL=60000`
- rendered config blocks for API/auth/attachments/ws/webhook/indexer/embedder
- a generated `.generated/compose/config/stream_config.yaml` starter file for the first external stream node, with `dave_allow_av1: false` by default for DAVE-encrypted stream compatibility
- rendered telemetry gateway image, config, and ingress host
- YugabyteDB YSQL, ScyllaDB contact points, etcd, and OpenSearch secrets
- ScyllaDB keyspace creation with explicit datacenter and replication factor
- ingress host rules for app, storage, and MinIO console
- MinIO credentials and bucket settings when enabled

The wrapper does not edit the base chart in place at runtime. It renders an override file and applies it with `helm upgrade --install`.
