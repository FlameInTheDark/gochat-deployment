# Docker Compose environment variables

The Docker Compose stack relies on a mix of environment variables with sensible defaults and placeholders. Set them in an `.env` file next to `compose/docker-compose.yaml` or export them in your shell before running `docker compose`.

## Required variables

- `PUBLIC_WS_URL` – Public WebSocket endpoint served by Traefik. Use the externally reachable `wss://` or `https://` origin for the `/ws` route so the UI can establish realtime connections.
- `PUBLIC_API_BASE_URL` – Public HTTPS endpoint for the REST API. The value must match the domain (and optional base path) exposed by Traefik for `/api/v1`.
- `PUBLIC_BASE_PATH` – Base pathname where the UI is mounted. Defaults to `/`, but update it when serving the SPA from a subdirectory so client-side routing generates correct links.

## Optional variables

- `GOCHAT_IMAGE_VARIANT` – Select the Docker tag (`latest` or `dev`) shared by the GoChat application containers.
- `GOCHAT_UI_IMAGE` – Override the repository name for the SPA image while keeping the standard tag handling.
- `API_CONFIG_FILE`, `AUTH_CONFIG_FILE`, `INDEXER_CONFIG_FILE`, `WS_CONFIG_FILE` – Override which configuration file from `compose/config/` is mounted into each respective service. Defaults map to the committed `.yaml` files but can be switched to `.local.yaml` copies without editing the Compose manifest.
- `SCYLLA_DEVELOPER_MODE` – Toggle Scylla developer mode (defaults to `1` to relax resource checks for local usage).
- `PG_ADDRESS` – PostgreSQL connection string consumed by the migrations helper. Defaults to the internal Citus master service.
- `CASSANDRA_ADDRESS` – ScyllaDB connection string for migrations. Defaults to the internal Scylla service with multi-statement support enabled.
- `GOCHAT_MIGRATIONS_REPO` – Git repository cloned by the migrations job. Defaults to the upstream GoChat repository.
- `GOCHAT_MIGRATIONS_BRANCH` – Branch checked out when fetching migrations. Defaults to `main`.
- `OPENSEARCH_INITIAL_ADMIN_PASSWORD` – Bootstrap password for the OpenSearch distribution.
- `OPENSEARCH_HOSTS` – Dashboard connection string targeting the OpenSearch service.
- `DISABLE_SECURITY_DASHBOARDS_PLUGIN` – Disable the security plugin in the OpenSearch Dashboards container (defaults to `true`).
- `COMPOSE_PROJECT_NAME` – Project name reused by the embedded Citus images to derive hostnames.
- `COORDINATOR_EXTERNAL_PORT` – Host port mapped to the Citus coordinator Postgres service (defaults to `5432`).
- `POSTGRES_USER` – Username used by Citus components and seeded in the database.
- `POSTGRES_PASSWORD` – Password for the configured PostgreSQL user.
- `POSTGRES_HOST_AUTH_METHOD` – Authentication strategy passed to the Postgres image (defaults to `trust` for local development).
- `DOCKER_SOCK` – Path to the Docker socket mounted into the Citus membership manager (defaults to `/var/run/docker.sock`).

All variables not listed inherit the Compose default values baked into `compose/docker-compose.yaml`.
