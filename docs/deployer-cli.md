# Deployer CLI

The deployer binary is the primary interface for this repository.

If you run `gochat-deployer` with no subcommand, it starts the interactive wizard.

## Commands

### `gochat-deployer`

Starts the Charm-based wizard.

### `gochat-deployer wizard`

Starts the Charm-based wizard explicitly.

### `gochat-deployer check`

Checks the external tools needed for the selected deployment target.

Examples:

```bash
gochat-deployer check --deployment-type compose
gochat-deployer check --deployment-type helm
```

### `gochat-deployer tokens sfu`

Generates the webhook JWT used by an external SFU node.

Examples:

```bash
gochat-deployer tokens sfu --secret WEBHOOK_SECRET
gochat-deployer tokens sfu --secret WEBHOOK_SECRET --id sfu-eu-1 --format json --header
```

If `--id` is omitted, the deployer generates a UUIDv4 service id automatically.

### `gochat-deployer render`

Renders generated config into the workspace without running Docker Compose or Helm.

Examples:

```bash
gochat-deployer render \
  --deployment-type compose \
  --storage-mode minio \
  --openobserve-root-email ops@example.com \
  --openobserve-root-password 'StrongPassword123!' \
  --base-domain example.com
```

```bash
gochat-deployer render \
  --deployment-type helm \
  --storage-mode external \
  --base-domain example.com \
  --openobserve-root-email ops@example.com \
  --openobserve-root-password 'StrongPassword123!' \
  --external-s3-endpoint https://s3.example.com \
  --external-s3-access-key-id ACCESS_KEY \
  --external-s3-secret-access-key SECRET_KEY
```

`render` prints the exact `docker compose` and `helm upgrade --install` commands that can be run against the generated workspace.
It also writes `.generated/deployment-guide.md`, a Markdown handoff file with URLs, commands, credentials, and a standalone SFU deployment section for the rendered deployment.

### `gochat-deployer deploy`

Renders config and then runs Docker Compose or Helm.

Examples:

```bash
gochat-deployer deploy \
  --deployment-type compose \
  --storage-mode minio \
  --openobserve-root-email ops@example.com \
  --openobserve-root-password 'StrongPassword123!' \
  --base-domain example.com
```

```bash
gochat-deployer deploy \
  --deployment-type helm \
  --storage-mode external \
  --base-domain example.com \
  --namespace gochat \
  --release-name gochat \
  --openobserve-root-email ops@example.com \
  --openobserve-root-password 'StrongPassword123!' \
  --external-s3-endpoint https://s3.example.com \
  --external-s3-public-base-url https://cdn.example.com/gochat \
  --external-s3-access-key-id ACCESS_KEY \
  --external-s3-secret-access-key SECRET_KEY
```

`deploy` also refreshes `.generated/deployment-guide.md` so the workspace keeps a current Markdown handoff after the deployment finishes.

### `gochat-deployer export`

Exports the embedded bundle into a workspace without rendering deployment-specific values.

Example:

```bash
gochat-deployer export --workspace-root .gochat-deployer/workspace
```

## Common Flags

These flags are used by `check`, `render`, `deploy`, and as wizard seed values.

### Deployment Selection

- `--deployment-type compose|helm`
- `--storage-mode minio|external`
- `--workspace-root PATH`

### Public Routing

- `--base-domain DOMAIN`
- `--app-host HOST`
- `--api-host HOST`
- `--ws-host HOST`
- `--storage-host HOST`
- `--minio-console-host HOST`
- `--openobserve-host HOST`

The deployer follows the upstream single-domain router shape:

- UI: `https://example.com`
- API: `https://example.com/api/v1`
- WS: `wss://example.com/ws`

### Images

- `--image-repository-prefix REPO`
- `--backend-tag TAG`
- `--frontend-tag TAG`
- `--migrations-image-repository REPO`
- `--migrations-image-tag TAG`

If `--backend-tag` is omitted, the deployer resolves the latest stable tag from `FlameInTheDark/gochat`.

If `--frontend-tag` is omitted, the deployer resolves the latest stable tag from `FlameInTheDark/gochat-react`.

If `--migrations-image-repository` is omitted, the deployer uses `<image-repository-prefix>/gochat-migrations`.

If `--migrations-image-tag` is omitted, the deployer matches the migrations image tag to the resolved backend tag.

### Secrets

- `--auth-secret VALUE`
- `--webhook-jwt-secret VALUE`
- `--postgres-password VALUE`
- `--etcd-root-password VALUE`
- `--opensearch-admin-password VALUE`
- `--openobserve-root-email VALUE`
- `--openobserve-root-password VALUE`

OpenObserve admin email and password must be supplied explicitly for `render` and `deploy`.

## Compose Target

`check --deployment-type compose` requires:

- `docker`

Compose always uses bundled Traefik and HTTP public URLs.

## Helm Target

Helm-specific flags:

- `--namespace NAME`
- `--release-name NAME`
- `--tls`
- `--no-tls`
- `--bundled-traefik`
- `--no-bundled-traefik`
- `--ingress-class-name NAME`
- `--tls-secret-name NAME`
- `--openobserve-host HOST`

`check --deployment-type helm` requires:

- `helm`

Optional:

- `kubectl`

Helm defaults to TLS-aware public URLs, bundled OpenObserve/OTEL, and an in-cluster UI build from the frontend repo tag.

If `--ingress-class-name` is set and `--bundled-traefik` is not forced on, the deployer auto-disables bundled Traefik and renders the separate websocket ingresses needed for ingress-nginx style clusters.

## Storage Flags

### Bundled MinIO

- `--storage-mode minio`
- `--storage-bucket NAME`
- `--minio-root-user USER`
- `--minio-root-password PASSWORD`

Bundled MinIO is configured for public object access.

### External S3

- `--storage-mode external`
- `--external-s3-endpoint URL`
- `--external-s3-public-base-url URL`
- `--external-s3-access-key-id KEY`
- `--external-s3-secret-access-key SECRET`
- `--external-s3-region REGION`
- `--external-s3-use-ssl`

For external S3, `endpoint`, `access key id`, and `secret access key` are required.

## Email Provider Flags

Shared email flags:

- `--email-provider log|smtp|sendpulse|resend|dashamail`
- `--email-source ADDRESS`
- `--email-name NAME`

Provider-specific flags:

- SMTP:
  `--smtp-host`, `--smtp-port`, `--smtp-username`, `--smtp-password`, `--smtp-use-tls`
- SendPulse:
  `--sendpulse-user-id`, `--sendpulse-secret`
- Resend:
  `--resend-api-key`
- DashaMail:
  `--dashamail-api-key`

The deployer validates the required fields for the selected provider before rendering or deploying.

## GitHub Access

`render` and `deploy` need outbound GitHub access because the deployer:

- resolves the latest stable backend and frontend tags when overrides are not provided
- fetches backend migration files for the backend tag that will be deployed

Optional authentication:

- `GITHUB_TOKEN`
- `GH_TOKEN`

If either variable is set, the deployer sends it as a bearer token to GitHub API requests.

## Wizard

The wizard uses Charm tooling and is the default entrypoint.

Accessible mode:

```bash
GOCHAT_DEPLOYER_ACCESSIBLE=1 gochat-deployer
```

## Script Wrappers

The repository also includes:

- `scripts/deploy.sh`
- `scripts/deploy.ps1`

These scripts no longer implement deployment logic themselves. They either:

1. build the local deployer binary from the current repo, or
2. download a release binary and run it.

Wrapper-specific environment variables:

- `GOCHAT_DEPLOYER_VERSION`
  `latest` by default, or a pinned tag like `v1.2.3`
- `GOCHAT_DEPLOYER_USE_RELEASE`
  set to `1` or `true` to force the wrapper to download a release binary even inside a cloned repo
- `GOCHAT_DEPLOYER_REPO`
  overrides the GitHub repository slug used for release downloads
