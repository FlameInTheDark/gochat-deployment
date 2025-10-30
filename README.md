# GoChat deployment

This repository contains infrastructure assets for running the GoChat stack either with Docker Compose or on Kubernetes through Helm. The compose configuration mirrors the original local development layout, while the Helm chart packages the same services with sensible defaults that can be tuned through values.

## Repository layout

- `compose/` – Docker Compose manifests, default configuration files, bootstrap scripts and templates.
- `helm/gochat/` – A Helm chart that deploys the complete GoChat stack on Kubernetes.
- `helm/gochat/files/scripts/run-migrations.sh` – Helper script executed by the Compose and Helm migrations jobs.
- `docs/compose-environment.md` – Environment variable reference for the Docker Compose stack.
- `docs/helm-configuration.md` – Value and environment overview for the Helm chart.
- `README.md` – This guide.

## Docker Compose usage

1. Copy the sample configuration files if you need custom values:
   ```bash
   cp compose/config/api_config.yaml compose/config/api_config.local.yaml
   cp compose/config/auth_config.yaml compose/config/auth_config.local.yaml
   cp compose/config/indexer_config.yaml compose/config/indexer_config.local.yaml
   cp compose/config/ws_config.yaml compose/config/ws_config.local.yaml
   ```
   Update the copied files and set the corresponding `*_CONFIG_FILE` environment variables (`API_CONFIG_FILE`, `AUTH_CONFIG_FILE`, `INDEXER_CONFIG_FILE`, `WS_CONFIG_FILE`) to point at the `.local.yaml` copies so Docker Compose picks them up without editing the manifest. The committed files contain default values that allow the stack to boot without additional changes.
2. Review the Traefik labels in the compose file and update domain names or paths to match your environment.
3. (Optional) Choose which GoChat application images to deploy by setting `GOCHAT_IMAGE_VARIANT` to either `latest` (default) or `dev` before running Compose. The UI container now serves both the marketing site and application shell directly from `/`.
4. Start the stack:
  ```bash
  docker compose -f compose/docker-compose.yaml up -d
  ```
5. Stop everything when finished:
   ```bash
   docker compose -f compose/docker-compose.yaml down
   ```

The compose bundle also includes:

- `compose/init/init-scylladb.sh` which guarantees the `gochat` keyspace exists when Scylla boots.
- Automatic PostgreSQL and Scylla migrations through the `migrations` one-shot service. Override `PG_ADDRESS`, `CASSANDRA_ADDRESS`, `GOCHAT_MIGRATIONS_REPO` or `GOCHAT_MIGRATIONS_BRANCH` to target external databases or a different revision of the upstream repository.
- Email templates under `compose/templates/` used by the auth service.

## Helm chart

The Helm chart deploys the same set of services as Docker Compose: ScyllaDB, NATS, KeyDB, the API/auth/ws/indexer services, the UI, OpenSearch plus dashboards, Traefik and a single-node Citus master. Additional workers and the membership manager can be enabled through the chart values if you need a clustered PostgreSQL deployment.

### Quick start

1. Package custom configuration by editing `helm/gochat/values.yaml` or supplying your own file with `--values`/`-f`.
2. Install the release into a namespace of your choice:
   ```bash
   helm install gochat ./helm/gochat -n gochat --create-namespace
   ```
3. To upgrade after making changes:
   ```bash
   helm upgrade gochat ./helm/gochat -n gochat
   ```
4. Remove the deployment when no longer required:
   ```bash
   helm uninstall gochat -n gochat
   ```

### Configuration highlights

- **Config maps** – API, auth, websocket and indexer services load their YAML configuration from config maps rendered from the values file. Update the relevant `config` blocks to match your infrastructure.
- **Migrations job** – A Helm hook spins up a transient job that runs both PostgreSQL and Scylla migrations on install/upgrade. Adjust the `migrations` block in `values.yaml` to point at custom database endpoints or a different repository/branch for fetching the latest migration files.
- **Persistent storage** – Scylla, OpenSearch and Citus master use persistent volume claims by default. Storage class names and sizes are configurable via the `persistence` sections.
- **Optional components** – Disable services by toggling the `enabled` flag under their respective section. For example, set `traefik.enabled=false` if you already run an ingress controller. Enable Citus workers by setting `citus.worker.enabled=true` and adjusting `replicaCount`.
- **Image variants** – Set `global.imageVariant` to `latest` (default) or `dev` to control which tag the API, auth, websocket, indexer and UI deployments use. Individual services can still override the `image.tag` field if necessary.
- **Ingress** – The chart ships with an optional generic ingress definition. Populate the `ingress` block to expose the UI, API or websocket routes through your ingress controller.

Refer to the environment and value references under [`docs/`](docs/) when preparing configuration overrides for either deployment path.

## Development notes

- Keep the Docker and Helm configurations in sync when adjusting environment variables, ports or new services.
- If you add new mounted files for the compose stack, mirror them in the Helm chart via config maps or secrets so both paths stay functional.
