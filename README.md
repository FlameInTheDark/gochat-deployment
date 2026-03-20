# GoChat Deployment

This repository deploys GoChat by pulling published images from:

- backend: `https://github.com/FlameInTheDark/gochat`
- frontend: `https://github.com/FlameInTheDark/gochat-react`

It does not clone, build, or push those application repositories during deployment. The `research/` directory is reference material only.

The primary entrypoint is the Go deployer binary. It embeds the Compose stack, Helm chart, and helper assets into a single executable. At render or deploy time it resolves the latest stable backend and frontend tags, then fetches backend migrations for the resolved backend tag so schema files stay aligned with the backend release.

## Quick Start

### Release Binary

Linux or macOS, latest release:

```bash
arch=$(uname -m); case "$arch" in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) echo "unsupported arch: $arch" >&2; exit 1 ;; esac; os=$(uname -s | tr '[:upper:]' '[:lower:]'); curl -fsSL "https://github.com/FlameInTheDark/gochat-deployment/releases/latest/download/gochat-deployer_${os}_${arch}.tar.gz" | tar -xz && ./gochat-deployer
```

Windows, latest release:

```powershell
$arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower() -eq 'arm64') { 'arm64' } else { 'amd64' }; $asset = "gochat-deployer_windows_${arch}.zip"; Invoke-WebRequest "https://github.com/FlameInTheDark/gochat-deployment/releases/latest/download/$asset" -OutFile $asset; Expand-Archive $asset . -Force; .\gochat-deployer.exe
```

Linux or macOS, pinned release:

```bash
version=v1.2.3; arch=$(uname -m); case "$arch" in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) echo "unsupported arch: $arch" >&2; exit 1 ;; esac; os=$(uname -s | tr '[:upper:]' '[:lower:]'); curl -fsSL "https://github.com/FlameInTheDark/gochat-deployment/releases/download/${version}/gochat-deployer_${os}_${arch}.tar.gz" | tar -xz && ./gochat-deployer
```

Windows, pinned release:

```powershell
$version = 'v1.2.3'; $arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower() -eq 'arm64') { 'arm64' } else { 'amd64' }; $asset = "gochat-deployer_windows_${arch}.zip"; Invoke-WebRequest "https://github.com/FlameInTheDark/gochat-deployment/releases/download/$version/$asset" -OutFile $asset; Expand-Archive $asset . -Force; .\gochat-deployer.exe
```

### Script Wrapper

Linux or macOS, latest script:

```bash
curl -fsSL https://raw.githubusercontent.com/FlameInTheDark/gochat-deployment/main/scripts/deploy.sh | bash
```

Windows, latest script:

```powershell
irm https://raw.githubusercontent.com/FlameInTheDark/gochat-deployment/main/scripts/deploy.ps1 | iex
```

Linux or macOS, pinned script:

```bash
curl -fsSL https://raw.githubusercontent.com/FlameInTheDark/gochat-deployment/v1.2.3/scripts/deploy.sh | GOCHAT_DEPLOYER_VERSION=v1.2.3 bash
```

Windows, pinned script:

```powershell
$env:GOCHAT_DEPLOYER_VERSION='v1.2.3'; irm https://raw.githubusercontent.com/FlameInTheDark/gochat-deployment/v1.2.3/scripts/deploy.ps1 | iex
```

The scripts are thin wrappers around the deployer binary. When run inside a cloned repo, they build the local binary if `go` is available. Otherwise they download a release binary and run it.

Set `GOCHAT_DEPLOYER_USE_RELEASE=1` if you want the wrapper to download a release binary even inside a cloned repo.

### Local Repository

Build and run from source:

```bash
go build -o ./bin/gochat-deployer .
./bin/gochat-deployer
```

or use the wrapper scripts:

```bash
./scripts/deploy.sh
```

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1
```

## CLI

The deployer supports:

- `wizard`
- `check`
- `tokens`
- `render`
- `deploy`
- `export`

Separate CLI reference:

- [docs/deployer-cli.md](docs/deployer-cli.md)

Version behavior:

- backend services default to the latest stable tag from `FlameInTheDark/gochat`
- frontend defaults to the latest stable tag from `FlameInTheDark/gochat-react`
- backend migrations are fetched from `FlameInTheDark/gochat` for the resolved backend tag
- `--backend-tag` and `--frontend-tag` override auto-resolution when needed

Accessible wizard mode:

```bash
GOCHAT_DEPLOYER_ACCESSIBLE=1 gochat-deployer
```

## Deployment Modes

Supported targets:

- Docker Compose for single-host deployment
- Helm for Kubernetes deployment

Supported storage modes:

- bundled MinIO with automatic bucket creation, anonymous download, and permissive CORS
- external S3-compatible storage

Supported auth email providers:

- `log`
- `smtp`
- `sendpulse`
- `resend`
- `dashamail`

SFU is intentionally excluded from automated deployment. It should be deployed manually with its own direct network exposure and port range handling.

## Public Routing

The deployer follows the upstream backend router shape on a single app domain:

- UI: `https://example.com`
- API: `https://example.com/api/v1`
- WS: `wss://example.com/ws`

Bundled MinIO uses separate public hosts:

- storage API/public objects: `https://storage.example.com`
- MinIO console: `https://minio.example.com`

The `/ws` prefix is stripped by the router before requests reach the WebSocket service, matching the upstream Compose deployment.

## Notes

- Compose forces bundled Traefik and plain HTTP public URLs.
- Compose also exposes OpenSearch Dashboards on `${OPENSEARCH_DASHBOARDS_PORT:-5601}` for search troubleshooting.
- Compose now also includes the upstream local observability path: OpenObserve on `${OPENOBSERVE_PORT:-5080}` plus an OpenTelemetry Collector with OTLP and Fluentd ingress.
- The collector health endpoint is exposed on `${OTEL_COLLECTOR_HEALTH_PORT:-13133}`. OTLP gRPC/HTTP are exposed on `${OTEL_COLLECTOR_GRPC_PORT:-4317}` and `${OTEL_COLLECTOR_HTTP_PORT:-4318}`.
- OpenObserve dashboard and alert seed assets are vendored in this repo under `monitoring/openobserve/`.
  Bootstrap them with `go run ./research/gochat/cmd/tools observability bootstrap --dashboard-dir ./monitoring/openobserve/bootstrap/dashboards --alerts-file ./monitoring/openobserve/bootstrap/alerts/alerts.seed.yaml ...`.
- Helm defaults to TLS-aware public URLs, bundled OpenObserve/OTEL, and an in-cluster UI build from the frontend repo tag.
- If you set an ingress class, the deployer auto-disables bundled Traefik and renders the nginx-style websocket ingresses we validated in-cluster.
- The deployer pulls published container images by default from `ghcr.io/flameinthedark`.
- Compose still uses the published frontend image. It must be published with same-host defaults like `/api/v1` and `/ws/subscribe`, or with deployment-specific public URLs baked in at image build time.
- `render` and `deploy` require GitHub access so the deployer can resolve release tags and fetch backend migrations. Set `GITHUB_TOKEN` or `GH_TOKEN` if you need authenticated GitHub API access.
- Generated runtime files are written under the selected workspace and can be inspected before deployment.
- The deployer also writes `.generated/deployment-guide.md` with URLs, commands, credentials, and an external SFU deployment playbook.
- OpenObserve admin email and password must be supplied explicitly when you run `render`, `deploy`, or the wizard.
- `gochat-deployer tokens sfu --secret <webhook-jwt-secret>` generates webhook credentials for additional SFU nodes.
- `render` prints exact manual deploy commands for the generated Compose and Helm outputs.
