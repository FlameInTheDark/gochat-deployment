# GoChat Deployment

> Deploy GoChat with one deployer binary on Docker Compose or Kubernetes.

This repository packages the GoChat deployment logic into a single Go-based deployer. It embeds the Compose stack, Helm chart, helper assets, and deployment templates so you can render or apply a deployment without manually stitching the platform together.

Upstream application repositories:

- backend: `https://github.com/FlameInTheDark/gochat`
- frontend: `https://github.com/FlameInTheDark/gochat-react`

You do not need to prepare those repositories manually before deploying. The deployer resolves backend and frontend versions, selects the matching migrations container, and generates a ready-to-apply workspace.

## Highlights

- Interactive terminal wizard and explicit CLI modes
- Docker Compose for single-host installs
- Helm for Kubernetes installs
- Automatic backend and frontend tag resolution
- Version-matched `gochat-migrations` container for deploy and update flows
- Built-in OpenObserve and OpenTelemetry Collector wiring
- Generated `.generated/deployment-guide.md` with URLs, commands, credentials, and SFU instructions

## Quick Start

### Run The Wizard

If you already have the deployer binary:

```bash
gochat-deployer
```

Accessible mode:

```bash
GOCHAT_DEPLOYER_ACCESSIBLE=1 gochat-deployer
```

### Download A Release Binary

Linux or macOS:

```bash
arch=$(uname -m); case "$arch" in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) echo "unsupported arch: $arch" >&2; exit 1 ;; esac; os=$(uname -s | tr '[:upper:]' '[:lower:]'); curl -fsSL "https://github.com/FlameInTheDark/gochat-deployment/releases/latest/download/gochat-deployer_${os}_${arch}.tar.gz" | tar -xz && ./gochat-deployer
```

Windows:

```powershell
$arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower() -eq 'arm64') { 'arm64' } else { 'amd64' }; $asset = "gochat-deployer_windows_${arch}.zip"; Invoke-WebRequest "https://github.com/FlameInTheDark/gochat-deployment/releases/latest/download/$asset" -OutFile $asset; Expand-Archive $asset . -Force; .\gochat-deployer.exe
```

### Use The Wrapper Scripts

Linux or macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/FlameInTheDark/gochat-deployment/main/scripts/deploy.sh | bash
```

Windows:

```powershell
irm https://raw.githubusercontent.com/FlameInTheDark/gochat-deployment/main/scripts/deploy.ps1 | iex
```

The wrappers build the local deployer when you run them from a cloned repository and `go` is available. Otherwise they download a release binary and run it. Set `GOCHAT_DEPLOYER_USE_RELEASE=1` if you always want the wrapper to download a release binary.

### Build From Source

```bash
go build -o ./bin/gochat-deployer .
./bin/gochat-deployer
```

For pinned release downloads and the full CLI surface, see [docs/deployer-cli.md](docs/deployer-cli.md).

## Deployer Workflow

The recommended flow is:

1. Check prerequisites for your target.
2. Render a reviewable workspace.
3. Apply it with the deployer or the printed manual command.
4. Use `.generated/deployment-guide.md` for post-deploy access details.

Check prerequisites:

```bash
gochat-deployer check --deployment-type helm
gochat-deployer check --deployment-type compose
```

The deployer supports these primary commands:

- `wizard`
- `check`
- `tokens`
- `render`
- `deploy`
- `export`

## Common Deployment Flows

### Kubernetes With External S3

Preview first:

```bash
gochat-deployer render \
  --deployment-type helm \
  --workspace-root ./.gochat-deployer/k8s \
  --namespace gochat \
  --release-name gochat \
  --base-domain example.com \
  --ingress-class-name nginx \
  --tls-secret-name wildcard-example \
  --openobserve-host observe.example.com \
  --openobserve-root-email ops@example.com \
  --openobserve-root-password 'StrongPassword123!' \
  --storage-mode external \
  --external-s3-endpoint https://s3.example.com \
  --external-s3-public-base-url https://cdn.example.com/gochat \
  --external-s3-access-key-id ACCESS_KEY \
  --external-s3-secret-access-key SECRET_KEY \
  --email-provider resend \
  --resend-api-key RESEND_API_KEY
```

Apply directly:

```bash
gochat-deployer deploy \
  --deployment-type helm \
  --workspace-root ./.gochat-deployer/k8s \
  --namespace gochat \
  --release-name gochat \
  --base-domain example.com \
  --ingress-class-name nginx \
  --tls-secret-name wildcard-example \
  --openobserve-host observe.example.com \
  --openobserve-root-email ops@example.com \
  --openobserve-root-password 'StrongPassword123!' \
  --storage-mode external \
  --external-s3-endpoint https://s3.example.com \
  --external-s3-public-base-url https://cdn.example.com/gochat \
  --external-s3-access-key-id ACCESS_KEY \
  --external-s3-secret-access-key SECRET_KEY \
  --email-provider resend \
  --resend-api-key RESEND_API_KEY
```

### Single Host With Bundled MinIO

```bash
gochat-deployer deploy \
  --deployment-type compose \
  --workspace-root ./.gochat-deployer/compose \
  --base-domain example.com \
  --storage-mode minio \
  --openobserve-root-email ops@example.com \
  --openobserve-root-password 'StrongPassword123!' \
  --email-provider resend \
  --resend-api-key RESEND_API_KEY
```

`render` writes the generated bundle to `--workspace-root`, prints the exact `docker compose` or `helm upgrade --install` command, and refreshes `.generated/deployment-guide.md`. `deploy` does the same render step and then applies the deployment immediately.

## Public Routing

The deployer follows the upstream single-domain router shape:

| Surface | URL |
| --- | --- |
| UI | `https://example.com` |
| API | `https://example.com/api/v1` |
| WebSocket | `wss://example.com/ws` |

Bundled MinIO uses separate public hosts:

| Surface | URL |
| --- | --- |
| Public objects / storage API | `https://storage.example.com` |
| MinIO console | `https://minio.example.com` |

For nginx-based Helm deployments, the public websocket endpoint is `/ws`, and legacy `/ws/subscribe` requests are rewritten to the backend's `/subscribe` endpoint for compatibility.

## Supported Targets

| Target | Typical use | Frontend delivery | Routing |
| --- | --- | --- | --- |
| Docker Compose | Single host | Published frontend image | Bundled Traefik, HTTP URLs |
| Helm | Kubernetes | In-cluster UI build from frontend repo tag | Bundled Traefik or existing ingress controller |

Supported storage modes:

- bundled MinIO with automatic bucket creation, public reads, and permissive CORS
- external S3-compatible storage

Supported email providers:

- `log`
- `smtp`
- `sendpulse`
- `resend`
- `dashamail`

## What The Deployer Generates

Inside the selected workspace root, the deployer writes:

- the embedded Compose or Helm bundle
- rendered service configs and environment files
- a migrations image reference that matches the deployed backend release by default
- exact manual deployment commands
- `.generated/deployment-guide.md`

The generated guide contains:

- service URLs
- rendered credentials
- post-deploy commands
- OpenObserve access details
- external SFU deployment and token-generation instructions

## Notes

- Compose uses published container images from `ghcr.io/flameinthedark`.
- Compose still uses the published frontend image, so deployment-specific frontend URL behavior must be present in that image.
- Helm defaults to TLS-aware public URLs, bundled OpenObserve and OTEL, and an in-cluster UI build from the frontend repo tag.
- If you set an ingress class and do not force bundled Traefik on, the deployer auto-disables bundled Traefik and renders ingress-nginx friendly websocket routing.
- OpenObserve admin email and password must be supplied explicitly for `render`, `deploy`, and the wizard.
- Compose exposes OpenSearch Dashboards on `${OPENSEARCH_DASHBOARDS_PORT:-5601}`.
- Compose also includes the upstream local observability path: OpenObserve on `${OPENOBSERVE_PORT:-5080}` and OTEL collector ports and health endpoints.
- Vendored OpenObserve dashboard and alert assets live under `monitoring/openobserve/`.
- `render` and `deploy` need GitHub access only when backend or frontend tags are omitted and the deployer has to resolve the latest releases. Set `GITHUB_TOKEN` or `GH_TOKEN` if you need authenticated GitHub API access.
- SFU is intentionally not deployed automatically. The deployer generates the credentials and instructions you need to deploy it separately.

## Documentation

- CLI reference: [docs/deployer-cli.md](docs/deployer-cli.md)
- Helm configuration: [docs/helm-configuration.md](docs/helm-configuration.md)
- Compose environment: [docs/compose-environment.md](docs/compose-environment.md)
