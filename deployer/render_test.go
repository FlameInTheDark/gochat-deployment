package deployer

import (
	"context"
	"strings"
	"testing"
)

func TestRenderHelmValuesIncludesFrontendURLs(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType:   DeploymentHelm,
		StorageMode:      StorageMinIO,
		BaseDomain:       "example.com",
		BackendTag:       "v1.2.3",
		FrontendTag:      "v2.3.4",
		IngressClassName: "nginx",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	configs := map[string]string{
		"api":         "api-config",
		"auth":        "auth-config",
		"attachments": "attachments-config",
		"ws":          "ws-config",
		"webhook":     "webhook-config",
		"indexer":     "indexer-config",
		"embedder":    "embedder-config",
	}

	values := renderHelmValues(prepared, configs)

	for _, expected := range []string{
		`VITE_API_BASE_URL: "https://example.com/api/v1"`,
		`VITE_WEBSOCKET_URL: "wss://example.com/ws"`,
		`VITE_BASE_PATH: "/"`,
		`repository: https://github.com/FlameInTheDark/gochat-react.git`,
		`ref: "v2.3.4"`,
	} {
		if !strings.Contains(values, expected) {
			t.Fatalf("rendered values missing %q", expected)
		}
	}
}

func TestRenderHelmValuesIncludesObservabilityAndWebsocketIngress(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), Options{
		DeploymentType:            DeploymentHelm,
		StorageMode:               StorageExternal,
		BaseDomain:                "example.com",
		BackendTag:                "v1.2.3",
		FrontendTag:               "v2.3.4",
		IngressClassName:          "nginx",
		TLSSecretName:             "wildcard-example",
		OpenObserveHost:           "observe.example.com",
		OpenObserveRootEmail:      "ops@example.com",
		OpenObserveRootPassword:   "Complexpass#123",
		ExternalS3Endpoint:        "https://s3.example.com",
		ExternalS3AccessKeyID:     "access",
		ExternalS3SecretAccessKey: "secret",
		ExternalS3UseSSL:          true,
	})
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	values := renderHelmValues(prepared, map[string]string{
		"api":         "api-config",
		"auth":        "auth-config",
		"attachments": "attachments-config",
		"ws":          "ws-config",
		"webhook":     "webhook-config",
		"indexer":     "indexer-config",
		"embedder":    "embedder-config",
	})

	for _, expected := range []string{
		`enabled: true`,
		`value: "kubernetes"`,
		`value: "http://gochat-otel-collector:4318"`,
		`proxyBodySize: "50m"`,
		`nginx.ingress.kubernetes.io/rewrite-target: /subscribe`,
		`websocket:`,
		`rootUserEmail: "ops@example.com"`,
		`rootUserPassword: "Complexpass#123"`,
		`host: "observe.example.com"`,
		`secretName: wildcard-example`,
	} {
		if !strings.Contains(values, expected) {
			t.Fatalf("rendered helm values missing %q", expected)
		}
	}
}

func TestRenderAuthConfigIncludesProviderSpecificSecrets(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType:  DeploymentCompose,
		StorageMode:     StorageMinIO,
		BaseDomain:      "example.com",
		BackendTag:      "v1.2.3",
		FrontendTag:     "v2.3.4",
		EmailProvider:   "sendpulse",
		SendpulseUserID: "user-id",
		SendpulseSecret: "sendpulse-secret",
		ResendAPIKey:    "resend-key",
		DashaMailAPIKey: "dashamail-key",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	config := renderAuthConfig(prepared, "keydb:6379", "host=citus-master")
	for _, expected := range []string{
		`email_provider: "sendpulse"`,
		`sendpulse_user_id: "user-id"`,
		`sendpulse_secret: "sendpulse-secret"`,
		`resend_api_key: "resend-key"`,
		`dashamail_api_key: "dashamail-key"`,
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("rendered auth config missing %q", expected)
		}
	}
}

func TestRenderAPIConfigIncludesAttachmentDefaults(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType: DeploymentCompose,
		StorageMode:    StorageMinIO,
		BaseDomain:     "example.com",
		BackendTag:     "v1.2.3",
		FrontendTag:    "v2.3.4",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	config := renderAPIConfig(prepared, "scylla", "keydb:6379", "host=citus-master", "http://opensearch:9200", "nats://nats:4222", "nats://indexer:4222", "http://etcd:2379")
	for _, expected := range []string{
		"upload_limit: 50000000",
		"attachment_ttl_minutes: 10",
		`- id: eu`,
		`name: "Europe (Frankfurt)"`,
		`- id: us-east`,
		`name: "US East (Ashburn)"`,
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("rendered api config missing %q", expected)
		}
	}
}

func TestRenderWSConfigIncludesNATSForHelm(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType: DeploymentHelm,
		StorageMode:    StorageMinIO,
		BaseDomain:     "example.com",
		BackendTag:     "v1.2.3",
		FrontendTag:    "v2.3.4",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	config := renderWSConfig(prepared, "gochat-scylla", "host=gochat-citus-master", "gochat-keydb:6379", "nats://gochat-nats:4222")
	for _, expected := range []string{
		`nats_conn_string: "nats://gochat-nats:4222"`,
		`cache_addr: "gochat-keydb:6379"`,
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("rendered ws config missing %q", expected)
		}
	}
}

func TestRenderComposeEnvIncludesObservabilityValues(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType: DeploymentCompose,
		StorageMode:    StorageMinIO,
		BaseDomain:     "example.com",
		BackendTag:     "v1.2.3",
		FrontendTag:    "v2.3.4",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	env := renderComposeEnv(prepared)
	for _, expected := range []string{
		"OPENOBSERVE_PORT=5080",
		"OTEL_COLLECTOR_HEALTH_PORT=13133",
		"OTEL_COLLECTOR_FLUENTD_PORT=24224",
		"OTEL_COLLECTOR_GRPC_PORT=4317",
		"OTEL_COLLECTOR_HTTP_PORT=4318",
		"OPENOBSERVE_ROOT_EMAIL=ops@example.com",
		"OPENOBSERVE_ROOT_PASSWORD=Complexpass#123",
		"OPENOBSERVE_ORG=default",
		"OPENOBSERVE_LOG_STREAM=gochat_logs",
		"OPENOBSERVE_METRIC_STREAM=gochat-metrics",
		"OPENOBSERVE_TRACE_STREAM=gochat_traces",
	} {
		if !strings.Contains(env, expected) {
			t.Fatalf("rendered compose env missing %q", expected)
		}
	}
}
