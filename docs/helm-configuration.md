# Helm configuration and environment variables

The Helm chart exposes most tunables through `values.yaml`. Override them with `--set` or an additional values file when installing or upgrading the release.

## Required values

- `ingress.hosts[].host` – Replace `example.com` with the fully qualified domain you plan to expose through your ingress controller.
- `ingress.hosts[].paths` – Ensure the `/` path points at the UI service. Adjust or add additional prefixes for the API (`/api/v1`) and websocket (`/ws`) routes as needed.

## Optional values and related environment variables

- `global.imageVariant` – Changes the default tag (`latest` or `dev`) applied to the GoChat workloads.
- `ui.image.*` – Override the SPA image repository, tag or pull policy.
- `migrations.enabled` – Controls whether the Helm hook job runs database migrations on install/upgrade.
- `migrations.image.*` – Choose the container that executes the migrations helper script.
- `migrations.pgAddress` – PostgreSQL connection string passed to the migrations job. Defaults to the in-cluster Citus master when available.
- `migrations.cassandraAddress` – Cassandra/Scylla connection string for the migrations job. Defaults to the in-cluster Scylla service when enabled.
- `migrations.repo` – Git repository cloned to retrieve migration files. Defaults to the upstream GoChat repository.
- `migrations.branch` – Branch name checked out when fetching migrations (defaults to `main`).
- `ingress.enabled` – Enables the bundled ingress definition. Disable if you prefer to manage routing separately.
- `traefik.enabled` – Deploys Traefik as part of the release. Turn it off if you already run an ingress controller.
- `citus.enabled` / `scylla.enabled` – Control the lifecycle of the bundled database services. If you disable them, supply external endpoints through the `migrations` block and service-specific configuration sections.
- `imagePullSecrets` – Provide credentials when pulling images from private registries.

Refer to the inline comments in `helm/gochat/values.yaml` for exhaustive service-specific knobs such as resource requests, replica counts and storage options.
