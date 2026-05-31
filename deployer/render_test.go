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
		"api":              "api-config",
		"auth":             "auth-config",
		"attachments":      "attachments-config",
		"search":           "search-config",
		"ws":               "ws-config",
		"webhook":          "webhook-config",
		"indexer":          "indexer-config",
		"embedder":         "embedder-config",
		"telemetryGateway": "telemetry-gateway-config",
	}

	values := renderHelmValues(prepared, configs)

	for _, expected := range []string{
		`VITE_API_BASE_URL: "https://example.com/api/v1"`,
		`VITE_WEBSOCKET_URL: "wss://example.com/ws"`,
		`VITE_BASE_PATH: "/"`,
		`repository: https://github.com/FlameInTheDark/gochat-react.git`,
		`ref: "v2.3.4"`,
		"migrations:\n  scope: \"all\"\n  image:\n    repository: ghcr.io/flameinthedark/gochat-migrations\n    tag: \"v1.2.3\"",
		"relational:\n  provider: \"yugabyte\"\n  yugabyte:\n    host: \"yb-tservers.gochat-yb.svc.cluster.local\"\n    port: 5433",
		"    colocation: false",
		"yugabyteInit:\n  enabled: true\n  image:\n    repository: yugabytedb/yugabyte\n    tag: \"2025.2.2.2-b11\"",
		"scylla:\n  enabled: false",
		"      host: \"gochat-scylla-client.gochat-scylla.svc.cluster.local\"",
		"      replicationClass: \"NetworkTopologyStrategy\"",
		"      datacenter: \"gochat-dc\"",
		"      replicationFactor: 3",
		"  cassandraAddress: \"cassandra://gochat-scylla-client.gochat-scylla.svc.cluster.local:9042/gochat?x-multi-statement=true\"",
	} {
		if !strings.Contains(values, expected) {
			t.Fatalf("rendered values missing %q", expected)
		}
	}
}

func TestRenderDatabaseChartValuesIncludeTopology(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType:             DeploymentHelm,
		StorageMode:                StorageMinIO,
		BaseDomain:                 "example.com",
		BackendTag:                 "v1.2.3",
		FrontendTag:                "v2.3.4",
		ScyllaNodeCount:            5,
		ScyllaReplicationFactor:    3,
		YugabyteTServerCount:       5,
		YugabyteReplicationFactor:  3,
		YugabyteTServerStorageSize: "20Gi",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	scyllaValues := renderScyllaValues(prepared)
	for _, expected := range []string{
		`fullnameOverride: "gochat-scylla"`,
		`datacenter: "gochat-dc"`,
		`  - name: rack1`,
		`    members: 2`,
		`  - name: rack2`,
		`    members: 2`,
		`  - name: rack3`,
		`    members: 1`,
		`      capacity: 50Gi`,
	} {
		if !strings.Contains(scyllaValues, expected) {
			t.Fatalf("rendered scylla values missing %q", expected)
		}
	}

	yugabyteValues := renderYugabyteValues(prepared)
	for _, expected := range []string{
		`  tag: "2025.2.3.0-b149"`,
		"replicas:\n  master: 3\n  tserver: 5\n  totalMasters: 3",
		`    size: 20Gi`,
		`      cpu: "2"`,
		`      memory: 2Gi`,
	} {
		if !strings.Contains(yugabyteValues, expected) {
			t.Fatalf("rendered yugabyte values missing %q", expected)
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
		"api":              "api-config",
		"auth":             "auth-config",
		"attachments":      "attachments-config",
		"search":           "search-config",
		"ws":               "ws-config",
		"webhook":          "webhook-config",
		"indexer":          "indexer-config",
		"embedder":         "embedder-config",
		"telemetryGateway": "telemetry-gateway-config",
	})

	for _, expected := range []string{
		`enabled: true`,
		`value: "kubernetes"`,
		`value: "http://gochat-otel-collector:4318"`,
		`OTEL_METRIC_EXPORT_INTERVAL`,
		`value: "60000"`,
		`proxyBodySize: "50m"`,
		`nginx.ingress.kubernetes.io/rewrite-target: /subscribe`,
		`websocket:`,
		`host: telemetry.example.com`,
		`service: telemetry-gateway`,
		`repository: ghcr.io/flameinthedark/gochat-telemetry-gateway`,
		`telemetry-gateway-config`,
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

	config := renderAuthConfig(prepared, "keydb:6379", prepared.composePGDSN, "nats://nats:4222")
	for _, expected := range []string{
		`email_provider: "sendpulse"`,
		`mfa_encryption_key: "`,
		`nats_conn_string: "nats://nats:4222"`,
		`mfa_recovery_template: "./mfa_recovery.tmpl"`,
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

	config := renderAPIConfig(prepared, "scylla", "keydb:6379", prepared.composePGDSN, "http://opensearch:9200", "nats://nats:4222", "nats://indexer:4222", "http://etcd:2379")
	for _, expected := range []string{
		"upload_limit: 50000000",
		"attachment_ttl_minutes: 10",
		`- id: eu`,
		`name: "Europe (Frankfurt)"`,
		`- id: us-east`,
		`name: "US East (Ashburn)"`,
		`stream_etcd_prefix: "/gochat/stream"`,
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("rendered api config missing %q", expected)
		}
	}
	if strings.Contains(config, `stream_auth_secret`) {
		t.Fatal("rendered api config must not include stream_auth_secret")
	}
}

func TestRenderStreamConfigUsesSharedAuthAndStreamToken(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType:       DeploymentCompose,
		StorageMode:          StorageMinIO,
		BaseDomain:           "example.com",
		BackendTag:           "v1.2.3",
		FrontendTag:          "v2.3.4",
		AuthSecret:           "app-secret",
		WebhookJWTSecret:     "webhook-secret",
		OpenObserveRootEmail: "ops@example.com",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	config := renderStreamConfig(prepared, "eu", "https://stream-eu.example.com", "https://example.com", "https://telemetry.example.com")
	for _, expected := range []string{
		`server_address: ":3310"`,
		`auth_secret: "app-secret"`,
		`region: "eu"`,
		`public_base_url: "https://stream-eu.example.com"`,
		`webhook_url: "https://example.com"`,
		`dave_required_default: true`,
		`dave_allow_av1: false`,
		`max_audio_bitrate_kbps: 256`,
		`max_video_bitrate_kbps: 100000`,
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("rendered stream config missing %q", expected)
		}
	}
	if strings.Contains(config, "stream-secret") || strings.Contains(config, "stream_auth_secret") {
		t.Fatal("stream config must not render a separate stream auth secret")
	}
	if !strings.Contains(config, prepared.streamWebhookToken) {
		t.Fatalf("stream config missing generated stream webhook token")
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

	config := renderWSConfig(prepared, "gochat-scylla", prepared.yugabyteDSN, "gochat-keydb:6379", "nats://gochat-nats:4222")
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
		"TELEMETRY_HOST=telemetry.example.com",
		"TRAEFIK_TELEMETRY_ALIAS=telemetry.example.com",
		"OPENOBSERVE_ROOT_EMAIL=ops@example.com",
		"OPENOBSERVE_ROOT_PASSWORD=Complexpass#123",
		"OPENOBSERVE_ORG=default",
		"OPENOBSERVE_LOG_STREAM=gochat_logs",
		"OPENOBSERVE_METRIC_STREAM=gochat-metrics",
		"OPENOBSERVE_TRACE_STREAM=gochat_traces",
		"GOCHAT_IMAGE_TELEMETRY_GATEWAY=ghcr.io/flameinthedark/gochat-telemetry-gateway:v1.2.3",
		"GOCHAT_IMAGE_SEARCH=ghcr.io/flameinthedark/gochat-search:v1.2.3",
		"GOCHAT_IMAGE_MIGRATIONS=ghcr.io/flameinthedark/gochat-migrations:v1.2.3",
		"YUGABYTE_HOST=yugabyte",
		"YUGABYTE_PORT=5433",
		"YUGABYTE_USER=yugabyte",
		"YUGABYTE_PASSWORD=yugabyte",
		"YUGABYTE_DB=gochat",
		"YUGABYTE_COLOCATION=false",
		"YUGABYTE_ADDRESS=postgres://yugabyte:yugabyte@yugabyte:5433/gochat?sslmode=disable",
		"PG_ADDRESS=postgres://yugabyte:yugabyte@yugabyte:5433/gochat?sslmode=disable",
		"CASSANDRA_ADDRESS=cassandra://scylla:9042/gochat?x-multi-statement=true",
	} {
		if !strings.Contains(env, expected) {
			t.Fatalf("rendered compose env missing %q", expected)
		}
	}
}

func TestPrepareOptionsDefaultsMigrationsImageToBackendTag(t *testing.T) {
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

	if prepared.MigrationsImageRepo != "ghcr.io/flameinthedark/gochat-migrations" {
		t.Fatalf("unexpected migrations repository: %s", prepared.MigrationsImageRepo)
	}
	if prepared.MigrationsImageTag != "v1.2.3" {
		t.Fatalf("unexpected migrations tag: %s", prepared.MigrationsImageTag)
	}
	if prepared.imageMigrations != "ghcr.io/flameinthedark/gochat-migrations:v1.2.3" {
		t.Fatalf("unexpected migrations image ref: %s", prepared.imageMigrations)
	}
	if prepared.imageTelemetryGateway != "ghcr.io/flameinthedark/gochat-telemetry-gateway:v1.2.3" {
		t.Fatalf("unexpected telemetry gateway image ref: %s", prepared.imageTelemetryGateway)
	}
	if prepared.imageSearch != "ghcr.io/flameinthedark/gochat-search:v1.2.3" {
		t.Fatalf("unexpected search image ref: %s", prepared.imageSearch)
	}
}

func TestRenderSearchConfigUsesIndexerNATSAndStores(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType: DeploymentCompose,
		StorageMode:    StorageMinIO,
		BaseDomain:     "example.com",
		BackendTag:     "v1.2.3",
		FrontendTag:    "v2.3.4",
		AuthSecret:     "app-secret",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	config := renderSearchConfig(prepared, "scylla", "keydb:6379", prepared.composePGDSN, "http://opensearch:9200", "nats://indexer-nats:4222")
	for _, expected := range []string{
		`server_address: ":3100"`,
		`auth_secret: "app-secret"`,
		`cluster: ["scylla"]`,
		`keydb: "keydb:6379"`,
		`pg_dsn: "host=yugabyte port=5433 user=yugabyte password=yugabyte dbname=gochat sslmode=disable"`,
		`os_addresses: ["http://opensearch:9200"]`,
		`nats_conn_string: "nats://indexer-nats:4222"`,
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("rendered search config missing %q", expected)
		}
	}
}
