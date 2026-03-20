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
		"- OpenObserve: `https://observe.example.com`",
		"- Helm update: `helm upgrade --install gochat ./helm/gochat`",
		"- Helm status: `helm status gochat -n gochat`",
		"- OpenObserve email: `ops@example.com`",
		"- OpenObserve password: `Complexpass#123`",
		"- Access key ID: `access`",
		"- Secret access key: `secret`",
		"- Resend API key: `resend-key`",
		"- Pre-generated SFU service ID: `",
		"- Pre-generated SFU webhook token: `",
		"## External SFU Deployment",
		"- `webhook_url` should be the GoChat base origin",
		"git checkout v1.2.3",
		"gochat-deployer tokens sfu --secret",
		"$env:OTEL_EXPORTER_OTLP_TRACES_ENDPOINT = \"https://observe.example.com/api/default/v1/traces\"",
		"- Helm renders the public websocket endpoint as `/ws`",
	} {
		if !strings.Contains(guide, expected) {
			t.Fatalf("deployment guide missing %q", expected)
		}
	}
}
