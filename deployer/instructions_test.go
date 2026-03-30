package deployer

import (
	"context"
	"strings"
	"testing"
)

func TestRenderDeploymentGuideIncludesInstructionsAndCredentials(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), Options{
		Action:                    ActionRender,
		DeploymentType:            DeploymentHelm,
		StorageMode:               StorageExternal,
		BaseDomain:                "example.com",
		BackendTag:                "v1.2.3",
		FrontendTag:               "v2.3.4",
		IngressClassName:          "nginx",
		Namespace:                 "gochat",
		ReleaseName:               "gochat",
		WorkspaceRoot:             ".tmp-guide",
		OpenObserveHost:           "observe.example.com",
		OpenObserveRootEmail:      "ops@example.com",
		OpenObserveRootPassword:   "Complexpass#123",
		ExternalS3Endpoint:        "https://s3.example.com",
		ExternalS3PublicBaseURL:   "https://cdn.example.com/gochat",
		ExternalS3AccessKeyID:     "access",
		ExternalS3SecretAccessKey: "secret",
		ExternalS3Region:          "eu-central-1",
		ExternalS3UseSSL:          true,
		EmailProvider:             "resend",
		ResendAPIKey:              "resend-key",
	})
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	result := RenderResult{
		WorkspaceRoot:        prepared.WorkspaceRoot,
		GeneratedRoot:        prepared.WorkspaceRoot + "/.generated",
		ComposeEnvPath:       prepared.WorkspaceRoot + "/.generated/compose/.env",
		ComposeConfigRoot:    prepared.WorkspaceRoot + "/.generated/compose/config",
		ComposeFilePath:      prepared.WorkspaceRoot + "/compose/docker-compose.yaml",
		HelmChartPath:        prepared.WorkspaceRoot + "/helm/gochat",
		HelmValuesPath:       prepared.WorkspaceRoot + "/.generated/helm/values.generated.yaml",
		ComposeDeployCommand: "docker compose up -d",
		HelmDeployCommand:    "helm upgrade --install gochat ./helm/gochat",
		BackendTag:           prepared.backendTag,
		FrontendTag:          prepared.frontendTag,
		MigrationsTag:        prepared.backendTag,
		AppPublicURL:         prepared.appPublicURL,
		APIPublicBaseURL:     prepared.apiPublicBaseURL,
		WSPublicURL:          prepared.wsPublicURL,
		TelemetryGatewayURL:  "https://telemetry.example.com",
		StoragePublicURL:     prepared.storagePublicBaseURL,
		OpenObserveURL:       "https://observe.example.com",
		InstructionsPath:     deploymentGuidePath(prepared.WorkspaceRoot),
	}

	guide := renderDeploymentGuide(prepared, result)
	for _, expected := range []string{
		"# GoChat Deployment Guide",
		"Warning: this file contains credentials",
		"- UI: `https://example.com`",
		"- WebSocket: `wss://example.com/ws`",
		"- Telemetry Gateway: `https://telemetry.example.com`",
		"- OpenObserve: `https://observe.example.com`",
		"- Helm update: `helm upgrade --install gochat ./helm/gochat`",
		"- Helm status: `helm status gochat -n gochat`",
		"- Telemetry gateway config: `",
		"telemetry_gateway_config.yaml`",
		"- Auth MFA recovery template: `",
		"mfa_recovery.tmpl`",
		"- OpenObserve email: `ops@example.com`",
		"- OpenObserve password: `Complexpass#123`",
		"- MFA encryption key: `",
		"- Access key ID: `access`",
		"- Secret access key: `secret`",
		"- Resend API key: `resend-key`",
		"## Observability",
		"- Telemetry gateway URL: `https://telemetry.example.com`",
		"- OTEL metric export interval: `60000`",
		"- OpenObserve org: `default`",
		"- Log stream: `gochat_logs`",
		"- Metric stream: `gochat-metrics`",
		"- Trace stream: `gochat_traces`",
		"- Pre-generated SFU service ID: `",
		"- Pre-generated SFU webhook token: `",
		"## External SFU Deployment",
		"- `webhook_url` should be the GoChat base origin",
		"- Reuse the same SFU JWT for heartbeat auth and OTLP auth",
		"git checkout v1.2.3",
		"gochat-deployer tokens sfu --secret",
		"$env:OTEL_EXPORTER_OTLP_ENDPOINT = \"https://telemetry.example.com\"",
		"$env:OTEL_EXPORTER_OTLP_HEADERS = \"Authorization=Bearer $($env:WEBHOOK_TOKEN)\"",
		"telemetry_metric_export_interval: \"60000\"",
		"$env:OTEL_METRIC_EXPORT_INTERVAL = \"60000\"",
		"- Helm renders the public websocket endpoint as `/ws`",
	} {
		if !strings.Contains(guide, expected) {
			t.Fatalf("deployment guide missing %q", expected)
		}
	}
}
