# Helm Inputs

For Kubernetes deployments, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1 -DeploymentType helm
```

The wrapper generates `.generated/helm/values.generated.yaml` and installs the chart from [helm/gochat](/H:/Projects/Deployment/gochat-deployment/helm/gochat).

## Required Operator Inputs

- image repository prefix and tag for pulled images
- base domain or explicit public hosts
- namespace and release name
- storage mode: bundled MinIO or external S3
- ingress controller strategy:
  - existing ingress controller
  - bundled Traefik

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

When bundled Traefik is enabled, the chart creates the strip-prefix resources needed for `/ws`. If you use another ingress controller, you need to reproduce that rewrite behavior yourself.

SFU is intentionally excluded from the Helm chart. If you need voice, deploy SFU separately with the required direct networking and reuse the automated `webhook` plus `etcd` components for registration.

Example app endpoints for `example.com`:

- UI: `https://example.com`
- API: `https://example.com/api/v1`
- WS: `wss://example.com/ws`

## Values Rendered By The Wrapper

The generated override file pins:

- image repository and tag for every application container
- the migrations container image, which defaults to `gochat-migrations:<backend-tag>`
- `routing.appHost`
- rendered config blocks for API/auth/attachments/ws/webhook/indexer/embedder
- PostgreSQL, etcd, and OpenSearch secrets
- ingress host rules for app, storage, and MinIO console
- MinIO credentials and bucket settings when enabled

The wrapper does not edit the base chart in place at runtime. It renders an override file and applies it with `helm upgrade --install`.
