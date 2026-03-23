package deployer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const deploymentGuideFileName = "deployment-guide.md"

func deploymentGuidePath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".generated", deploymentGuideFileName)
}

func writeDeploymentGuide(path string, prepared *preparedOptions, result RenderResult) error {
	if path == "" {
		return fmt.Errorf("deployment guide path is required")
	}
	content := renderDeploymentGuide(prepared, result)
	return os.WriteFile(path, []byte(content), 0o600)
}

func renderDeploymentGuide(prepared *preparedOptions, result RenderResult) string {
	lines := []string{
		"# GoChat Deployment Guide",
		"",
		"> Warning: this file contains credentials and should be treated as sensitive.",
		"",
		"## Overview",
		"",
		fmt.Sprintf("- Action: `%s`", prepared.Action),
		fmt.Sprintf("- Deployment Type: `%s`", prepared.DeploymentType),
		fmt.Sprintf("- Storage Mode: `%s`", prepared.StorageMode),
		fmt.Sprintf("- Workspace: `%s`", result.WorkspaceRoot),
		fmt.Sprintf("- Backend Tag: `%s`", result.BackendTag),
		fmt.Sprintf("- Frontend Tag: `%s`", result.FrontendTag),
		fmt.Sprintf("- Migrations Tag: `%s`", result.MigrationsTag),
	}
	if prepared.DeploymentType == DeploymentHelm {
		lines = append(lines,
			fmt.Sprintf("- Namespace: `%s`", prepared.Namespace),
			fmt.Sprintf("- Release: `%s`", prepared.ReleaseName),
		)
	}

	lines = append(lines,
		"",
		"## Access",
		"",
		fmt.Sprintf("- UI: `%s`", result.AppPublicURL),
		fmt.Sprintf("- API: `%s`", result.APIPublicBaseURL),
		fmt.Sprintf("- WebSocket: `%s`", result.WSPublicURL),
	)
	if result.StoragePublicURL != "" {
		lines = append(lines, fmt.Sprintf("- Storage: `%s`", result.StoragePublicURL))
	}
	if result.MinIOConsoleURL != "" {
		lines = append(lines, fmt.Sprintf("- MinIO Console: `%s`", result.MinIOConsoleURL))
	}
	if result.OpenObserveURL != "" {
		lines = append(lines, fmt.Sprintf("- OpenObserve: `%s`", result.OpenObserveURL))
	}

	lines = append(lines,
		"",
		"## Commands",
		"",
		fmt.Sprintf("- Docker Compose update: `%s`", result.ComposeDeployCommand),
		fmt.Sprintf("- Helm update: `%s`", result.HelmDeployCommand),
	)
	if prepared.DeploymentType == DeploymentCompose {
		lines = append(lines,
			fmt.Sprintf("- Compose status: `docker compose --env-file %q -f %q ps`", result.ComposeEnvPath, result.ComposeFilePath),
			fmt.Sprintf("- Compose logs: `docker compose --env-file %q -f %q logs -f`", result.ComposeEnvPath, result.ComposeFilePath),
		)
	} else {
		lines = append(lines,
			fmt.Sprintf("- Helm status: `helm status %s -n %s`", prepared.ReleaseName, prepared.Namespace),
			fmt.Sprintf("- Pods: `kubectl get pods -n %s`", prepared.Namespace),
			fmt.Sprintf("- Ingress: `kubectl get ingress -n %s`", prepared.Namespace),
		)
	}

	lines = append(lines,
		"",
		"## Generated Files",
		"",
		fmt.Sprintf("- Compose env: `%s`", result.ComposeEnvPath),
		fmt.Sprintf("- Compose config dir: `%s`", result.ComposeConfigRoot),
		fmt.Sprintf("- Compose file: `%s`", result.ComposeFilePath),
		fmt.Sprintf("- Helm chart: `%s`", result.HelmChartPath),
		fmt.Sprintf("- Helm values: `%s`", result.HelmValuesPath),
	)

	lines = append(lines,
		"",
		"## Core Credentials",
		"",
		fmt.Sprintf("- Auth secret: `%s`", prepared.AuthSecret),
		fmt.Sprintf("- Webhook JWT secret: `%s`", prepared.WebhookJWTSecret),
		"- PostgreSQL user: `postgres`",
		fmt.Sprintf("- PostgreSQL password: `%s`", prepared.PostgresPassword),
		"- PostgreSQL database: `gochat`",
		"- etcd user: `root`",
		fmt.Sprintf("- etcd password: `%s`", prepared.EtcdRootPassword),
		"- OpenSearch user: `admin`",
		fmt.Sprintf("- OpenSearch password: `%s`", prepared.OpensearchAdminPassword),
		fmt.Sprintf("- OpenObserve email: `%s`", prepared.openObserveRootEmail),
		fmt.Sprintf("- OpenObserve password: `%s`", prepared.openObserveRootPassword),
		fmt.Sprintf("- Pre-generated SFU service ID: `%s`", prepared.sfuServiceID),
		fmt.Sprintf("- Pre-generated SFU webhook token: `%s`", prepared.sfuWebhookToken),
	)

	lines = append(lines,
		"",
		"## Storage Credentials",
		"",
		fmt.Sprintf("- Bucket: `%s`", prepared.StorageBucket),
	)
	if prepared.StorageMode == StorageMinIO {
		lines = append(lines,
			"- Mode: `bundled-minio`",
			fmt.Sprintf("- MinIO root user: `%s`", prepared.MinIORootUser),
			fmt.Sprintf("- MinIO root password: `%s`", prepared.MinIORootPassword),
		)
	} else {
		lines = append(lines,
			"- Mode: `external-s3`",
			fmt.Sprintf("- Endpoint: `%s`", prepared.storageEndpointURL),
			fmt.Sprintf("- Access key ID: `%s`", prepared.ExternalS3AccessKeyID),
			fmt.Sprintf("- Secret access key: `%s`", prepared.ExternalS3SecretAccessKey),
			fmt.Sprintf("- Region: `%s`", prepared.ExternalS3Region),
			fmt.Sprintf("- Use SSL: `%s`", boolText(prepared.storageUseSSL)),
		)
	}

	lines = append(lines,
		"",
		"## Email Configuration",
		"",
		fmt.Sprintf("- Provider: `%s`", prepared.EmailProvider),
		fmt.Sprintf("- Source: `%s`", prepared.EmailSource),
		fmt.Sprintf("- Name: `%s`", prepared.EmailName),
	)
	switch prepared.EmailProvider {
	case "smtp":
		lines = append(lines,
			fmt.Sprintf("- SMTP host: `%s`", prepared.SMTPHost),
			fmt.Sprintf("- SMTP port: `%d`", prepared.SMTPPort),
			fmt.Sprintf("- SMTP username: `%s`", prepared.SMTPUsername),
			fmt.Sprintf("- SMTP password: `%s`", prepared.SMTPPassword),
			fmt.Sprintf("- SMTP uses TLS: `%s`", boolText(prepared.SMTPUseTLS)),
		)
	case "sendpulse":
		lines = append(lines,
			fmt.Sprintf("- SendPulse user ID: `%s`", prepared.SendpulseUserID),
			fmt.Sprintf("- SendPulse secret: `%s`", prepared.SendpulseSecret),
		)
	case "resend":
		lines = append(lines, fmt.Sprintf("- Resend API key: `%s`", prepared.ResendAPIKey))
	case "dashamail":
		lines = append(lines, fmt.Sprintf("- DashaMail API key: `%s`", prepared.DashaMailAPIKey))
	}

	lines = append(lines, renderSFUDeploymentSection(prepared, result)...)

	lines = append(lines,
		"",
		"## Notes",
		"",
		"- Schema migrations run from the version-matched `gochat-migrations` image by default.",
		"- OpenObserve bootstrap assets live in `monitoring/openobserve/`.",
	)
	if prepared.DeploymentType == DeploymentHelm {
		lines = append(lines,
			"- Helm renders app OTEL env, an OpenObserve instance, and the collector by default.",
			"- Helm renders the public websocket endpoint as `/ws` and enables the alias ingress needed for ingress-nginx style setups.",
			"- Helm renders the UI as an in-cluster build from the frontend repo tag instead of depending on a prebuilt UI image.",
		)
	} else {
		lines = append(lines,
			"- Compose keeps using the published frontend image, so the baked frontend env still matters there.",
			"- Compose exposes OpenObserve on port `5080` and the collector health endpoint on port `13133` by default.",
		)
	}

	return strings.Join(lines, "\n") + "\n"
}

func renderSFUDeploymentSection(prepared *preparedOptions, result RenderResult) []string {
	observeBaseURL := sfuObserveBaseURL(prepared, result)
	observeBaseValue := "<set-a-reachable-openobserve-url>"
	if observeBaseURL != "" {
		observeBaseValue = observeBaseURL
	}

	lines := []string{
		"",
		"## External SFU Deployment",
		"",
		"- The deployer does not ship the SFU itself. Deploy the backend tag above as a standalone `cmd/sfu` binary or container on separate infrastructure.",
		"- Expose the SFU over WSS at `/signal` for clients and keep `/admin/channel/close` reachable from the GoChat API/control plane if you want region-move and forced close flows to work.",
		"- Choose an SFU region that matches the API allowlist currently rendered by this deployer: `global`, `eu`, or `us-east`.",
		"- `webhook_url` should be the GoChat base origin, not the full heartbeat path. The current SFU code appends `/api/v1/webhook/sfu/heartbeat`, `/api/v1/webhook/sfu/voice/join`, and `/api/v1/webhook/sfu/voice/leave` itself.",
		fmt.Sprintf("- GoChat base origin for SFU callbacks: `%s`", result.AppPublicURL),
		fmt.Sprintf("- Expected heartbeat endpoint once configured: `%s/webhook/sfu/heartbeat`", result.APIPublicBaseURL),
		"- Discovery prefix already configured on the GoChat side: `/gochat/sfu`.",
		"",
		"### Build",
		"",
		"```bash",
		"git clone https://github.com/FlameInTheDark/gochat.git",
		"cd gochat",
		fmt.Sprintf("git checkout %s", result.BackendTag),
		"go build -o ./bin/gochat-sfu ./cmd/sfu",
		"```",
		"",
		"### Minimal Config",
		"",
		"```yaml",
		`server_address: ":3300"`,
		fmt.Sprintf("auth_secret: %q", prepared.AuthSecret),
		"stun_servers:",
		`  - "stun:stun.l.google.com:19302"`,
		`region: "global"`,
		`public_base_url: "https://sfu-global.example.com"`,
		fmt.Sprintf("webhook_url: %q", result.AppPublicURL),
		fmt.Sprintf("webhook_token: %q", prepared.sfuWebhookToken),
		fmt.Sprintf("service_id: %q", prepared.sfuServiceID),
		"max_audio_bitrate_kbps: 0",
		"enforce_audio_bitrate: false",
		"audio_bitrate_margin_percent: 15",
		"```",
		"",
		"### Observability Environment",
		"",
		"```powershell",
		fmt.Sprintf("$env:GOCHAT_DEPLOYMENT_ENV = %q", sfuDeploymentEnv(prepared)),
		`$env:SFU_REGION = "global"`,
		fmt.Sprintf("$env:SFU_SERVICE_ID = %q", prepared.sfuServiceID),
		`$env:SFU_PUBLIC_BASE_URL = "https://sfu-global.example.com"`,
		fmt.Sprintf("$env:WEBHOOK_URL = %q", result.AppPublicURL),
		fmt.Sprintf("$env:WEBHOOK_TOKEN = %q", prepared.sfuWebhookToken),
		fmt.Sprintf("$env:AUTH_SECRET = %q", prepared.AuthSecret),
		fmt.Sprintf("$env:OTEL_EXPORTER_OTLP_TRACES_ENDPOINT = %q", observeBaseValue+"/api/"+prepared.openObserveOrg+"/v1/traces"),
		fmt.Sprintf("$env:OTEL_EXPORTER_OTLP_TRACES_HEADERS = %q", "Authorization=Basic "+prepared.openObserveBasicAuth),
		fmt.Sprintf("$env:OTEL_EXPORTER_OTLP_METRICS_ENDPOINT = %q", observeBaseValue+"/api/"+prepared.openObserveOrg+"/v1/metrics"),
		fmt.Sprintf("$env:OTEL_EXPORTER_OTLP_METRICS_HEADERS = %q", "Authorization=Basic "+prepared.openObserveBasicAuth),
		`$env:OPENOBSERVE_LOGS_ENABLED = "true"`,
		fmt.Sprintf("$env:OPENOBSERVE_LOGS_ENDPOINT = %q", observeBaseValue+"/api/"+prepared.openObserveOrg),
		fmt.Sprintf("$env:OPENOBSERVE_LOGS_AUTH = %q", "Basic "+prepared.openObserveBasicAuth),
		fmt.Sprintf("$env:OPENOBSERVE_LOGS_STREAM = %q", prepared.openObserveLogStream),
		"```",
		"",
		"### Token Generation",
		"",
		"- The pre-generated SFU credentials above can be used for the first node immediately.",
		"- Generate additional SFU credentials with the deployer when you add more nodes or want a different `service_id`.",
		"",
		"```bash",
		fmt.Sprintf("gochat-deployer tokens sfu --secret %s --id %s --format json", quoteCommandArg(prepared.WebhookJWTSecret), quoteCommandArg(prepared.sfuServiceID)),
		"```",
		"",
		"### Network Notes",
		"",
		"- Upstream docs require outbound access from the SFU host to the GoChat webhook origin and the OpenObserve OTLP/log ingestion endpoints.",
		"- STUN is configured by default. If your users are behind restrictive NAT, add TURN servers in `stun_servers` or your surrounding WebRTC config.",
		"- Inference from the current `cmd/sfu` code: there is no configurable fixed RTP port-range setting today, so plan host/firewall rules for normal ICE/WebRTC candidate traffic instead of exposing only port `3300`.",
	}
	if observeBaseURL == "" {
		lines = append(lines,
			"- OpenObserve is not published as a direct public URL in this render. Replace `<set-a-reachable-openobserve-url>` with a URL the external SFU host can actually reach.",
		)
	}
	return lines
}

func sfuObserveBaseURL(prepared *preparedOptions, result RenderResult) string {
	if result.OpenObserveURL != "" {
		return strings.TrimRight(result.OpenObserveURL, "/")
	}
	if prepared.DeploymentType == DeploymentCompose {
		return fmt.Sprintf("http://%s:5080", prepared.AppHost)
	}
	return ""
}

func sfuDeploymentEnv(prepared *preparedOptions) string {
	if prepared.DeploymentType == DeploymentHelm {
		return "kubernetes"
	}
	return "compose"
}
