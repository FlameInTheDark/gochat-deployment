package deployer

import (
	"fmt"
	"strings"
)

func (e *Engine) renderOutputs(prepared *preparedOptions) (map[string]string, string) {
	composeConfigs := map[string]string{
		"api_config.yaml":         renderAPIConfig(prepared, "scylla", "keydb:6379", prepared.composePGDSN, "http://opensearch:9200", "nats://nats:4222", "nats://indexer-nats:4222", "http://etcd:2379"),
		"auth_config.yaml":        renderAuthConfig(prepared, "keydb:6379", prepared.composePGDSN),
		"attachments_config.yaml": renderAttachmentsConfig(prepared, "scylla", "nats://nats:4222", "keydb:6379", prepared.composePGDSN),
		"ws_config.yaml":          renderWSConfig(prepared, "scylla", prepared.composePGDSN, "keydb:6379", "nats://nats:4222"),
		"webhook_config.yaml":     renderWebhookConfig(prepared, "scylla", "keydb:6379", "nats://nats:4222", "http://etcd:2379"),
		"indexer_config.yaml":     renderIndexerConfig(prepared, "nats://indexer-nats:4222", "http://opensearch:9200"),
		"embedder_config.yaml":    renderEmbedderConfig("scylla", "nats://nats:4222", "keydb:6379"),
	}

	helmPGDSN := fmt.Sprintf("host=%s-citus-master port=5432 user=postgres password=%s dbname=gochat sslmode=disable", prepared.helmFullName, prepared.PostgresPassword)
	helmValues := renderHelmValues(prepared, map[string]string{
		"api":         renderAPIConfig(prepared, prepared.helmFullName+"-scylla", prepared.helmFullName+"-keydb:6379", helmPGDSN, "http://"+prepared.helmFullName+"-opensearch:9200", "nats://"+prepared.helmFullName+"-nats:4222", "nats://"+prepared.helmFullName+"-indexer-nats:4222", "http://"+prepared.helmFullName+"-etcd:2379"),
		"auth":        renderAuthConfig(prepared, prepared.helmFullName+"-keydb:6379", helmPGDSN),
		"attachments": renderAttachmentsConfig(prepared, prepared.helmFullName+"-scylla", "nats://"+prepared.helmFullName+"-nats:4222", prepared.helmFullName+"-keydb:6379", helmPGDSN),
		"ws":          renderWSConfig(prepared, prepared.helmFullName+"-scylla", helmPGDSN, prepared.helmFullName+"-keydb:6379", "nats://"+prepared.helmFullName+"-nats:4222"),
		"webhook":     renderWebhookConfig(prepared, prepared.helmFullName+"-scylla", prepared.helmFullName+"-keydb:6379", "nats://"+prepared.helmFullName+"-nats:4222", "http://"+prepared.helmFullName+"-etcd:2379"),
		"indexer":     renderIndexerConfig(prepared, "nats://"+prepared.helmFullName+"-indexer-nats:4222", "http://"+prepared.helmFullName+"-opensearch:9200"),
		"embedder":    renderEmbedderConfig(prepared.helmFullName+"-scylla", "nats://"+prepared.helmFullName+"-nats:4222", prepared.helmFullName+"-keydb:6379"),
	})

	return composeConfigs, helmValues
}

func renderComposeEnv(prepared *preparedOptions) string {
	lines := []string{
		"APP_HOST=" + prepared.AppHost,
		"API_HOST=" + prepared.APIHost,
		"WS_HOST=" + prepared.WSHost,
		"STORAGE_HOST=" + prepared.StorageHost,
		"MINIO_CONSOLE_HOST=" + prepared.MinIOConsoleHost,
		"TRAEFIK_STORAGE_ALIAS=" + prepared.storageAlias,
		"TRAEFIK_MINIO_CONSOLE_ALIAS=" + prepared.minioConsoleAlias,
		"HTTP_PORT=80",
		"TRAEFIK_DASHBOARD_PORT=8080",
		"OPENSEARCH_DASHBOARDS_PORT=5601",
		"OPENOBSERVE_PORT=5080",
		"OTEL_COLLECTOR_HEALTH_PORT=13133",
		"OTEL_COLLECTOR_FLUENTD_PORT=24224",
		"OTEL_COLLECTOR_GRPC_PORT=4317",
		"OTEL_COLLECTOR_HTTP_PORT=4318",
		"",
		"POSTGRES_USER=postgres",
		"POSTGRES_PASSWORD=" + prepared.PostgresPassword,
		"POSTGRES_DB=gochat",
		"POSTGRES_HOST_AUTH_METHOD=md5",
		"PG_ADDRESS=" + prepared.composePGAddr,
		"CASSANDRA_ADDRESS=" + prepared.composeCassandraAddr,
		"",
		"ETCD_ROOT_PASSWORD=" + prepared.EtcdRootPassword,
		"OPENSEARCH_INITIAL_ADMIN_PASSWORD=" + prepared.OpensearchAdminPassword,
		"OPENOBSERVE_ROOT_EMAIL=" + prepared.openObserveRootEmail,
		"OPENOBSERVE_ROOT_PASSWORD=" + prepared.openObserveRootPassword,
		"OPENOBSERVE_ORG=" + prepared.openObserveOrg,
		"OPENOBSERVE_BASIC_AUTH=" + prepared.openObserveBasicAuth,
		"OPENOBSERVE_LOG_STREAM=" + prepared.openObserveLogStream,
		"OPENOBSERVE_METRIC_STREAM=" + prepared.openObserveMetricStream,
		"OPENOBSERVE_TRACE_STREAM=" + prepared.openObserveTraceStream,
		"",
		"MINIO_ROOT_USER=" + prepared.MinIORootUser,
		"MINIO_ROOT_PASSWORD=" + prepared.MinIORootPassword,
		"MINIO_BUCKET=" + prepared.StorageBucket,
		"",
		"GOCHAT_IMAGE_API=" + prepared.imageAPI,
		"GOCHAT_IMAGE_AUTH=" + prepared.imageAuth,
		"GOCHAT_IMAGE_ATTACHMENTS=" + prepared.imageAttachments,
		"GOCHAT_IMAGE_WS=" + prepared.imageWS,
		"GOCHAT_IMAGE_WEBHOOK=" + prepared.imageWebhook,
		"GOCHAT_IMAGE_INDEXER=" + prepared.imageIndexer,
		"GOCHAT_IMAGE_EMBEDDER=" + prepared.imageEmbedder,
		"GOCHAT_IMAGE_UI=" + prepared.imageUI,
		"GOCHAT_IMAGE_MIGRATIONS=" + prepared.imageMigrations,
	}
	return strings.Join(lines, "\n")
}

func renderAPIConfig(prepared *preparedOptions, scyllaHost, keydbAddr, pgDSN, opensearchAddr, natsAddr, indexerNatsAddr, etcdEndpoint string) string {
	contentHosts := make([]string, 0, len(prepared.contentHosts))
	for _, host := range prepared.contentHosts {
		contentHosts = append(contentHosts, fmt.Sprintf("  - %q", host))
	}

	return strings.Join([]string{
		"# App",
		`app_name: "GoChat"`,
		fmt.Sprintf("base_url: %q", prepared.appPublicURL),
		"content_hosts:",
		strings.Join(contentHosts, "\n"),
		"",
		"# API",
		"swagger: false",
		"api_log: true",
		`server_address: ":3100"`,
		"rate_limit_time: 1",
		"rate_limit_requests: 10",
		"idempotency_storage_lifetime: 10",
		"",
		"# Auth",
		fmt.Sprintf("auth_secret: %q", prepared.AuthSecret),
		"",
		"# Cassandra",
		fmt.Sprintf("cluster: [%q]", scyllaHost),
		`cluster_keyspace: "gochat"`,
		"",
		"# Redis",
		fmt.Sprintf("keydb: %q", keydbAddr),
		"",
		"# PostgreSQL",
		fmt.Sprintf("pg_dsn: %q", pgDSN),
		"pg_retries: 5",
		"",
		"# OpenSearch",
		"os_insecure_skip_verify: true",
		fmt.Sprintf("os_addresses: [%q]", opensearchAddr),
		`os_username: "admin"`,
		fmt.Sprintf("os_password: %q", prepared.OpensearchAdminPassword),
		"",
		"# NATS",
		fmt.Sprintf("nats_conn_string: %q", natsAddr),
		fmt.Sprintf("indexer_nats_conn_string: %q", indexerNatsAddr),
		"",
		"# Uploads / Attachments",
		"upload_limit: 50000000",
		"attachment_ttl_minutes: 10",
		"",
		`voice_region: "global"`,
		"voice_regions:",
		"  - id: global",
		`    name: "Global"`,
		"  - id: eu",
		`    name: "Europe (Frankfurt)"`,
		"  - id: us-east",
		`    name: "US East (Ashburn)"`,
		"",
		"# Discovery for manually managed SFU instances",
		"etcd_endpoints:",
		fmt.Sprintf("  - %q", etcdEndpoint),
		`etcd_prefix: "/gochat/sfu"`,
		`etcd_username: "root"`,
		fmt.Sprintf("etcd_password: %q", prepared.EtcdRootPassword),
	}, "\n")
}

func renderAuthConfig(prepared *preparedOptions, keydbAddr, pgDSN string) string {
	return strings.Join([]string{
		"# App",
		`app_name: "GoChat"`,
		fmt.Sprintf("base_url: %q", prepared.appPublicURL),
		"",
		"# API",
		`server_address: ":3100"`,
		"api_log: true",
		"idempotency_storage_lifetime: 10",
		"rate_limit_time: 1",
		"swagger: false",
		"rate_limit_requests: 20",
		"",
		"# Auth",
		fmt.Sprintf("auth_secret: %q", prepared.AuthSecret),
		"",
		"# Email",
		fmt.Sprintf("email_source: %q", prepared.EmailSource),
		fmt.Sprintf("email_name: %q", prepared.EmailName),
		`email_template: "./email_notify.tmpl"`,
		`password_reset_template: "./password_reset.tmpl"`,
		fmt.Sprintf("email_provider: %q", prepared.EmailProvider),
		fmt.Sprintf("sendpulse_user_id: %q", prepared.SendpulseUserID),
		fmt.Sprintf("sendpulse_secret: %q", prepared.SendpulseSecret),
		fmt.Sprintf("resend_api_key: %q", prepared.ResendAPIKey),
		fmt.Sprintf("dashamail_api_key: %q", prepared.DashaMailAPIKey),
		fmt.Sprintf("smtp_host: %q", prepared.SMTPHost),
		fmt.Sprintf("smtp_port: %d", prepared.SMTPPort),
		fmt.Sprintf("smtp_username: %q", prepared.SMTPUsername),
		fmt.Sprintf("smtp_password: %q", prepared.SMTPPassword),
		fmt.Sprintf("smtp_use_tls: %s", boolText(prepared.SMTPUseTLS)),
		"",
		"# Redis",
		fmt.Sprintf("keydb: %q", keydbAddr),
		"",
		"# PostgreSQL",
		fmt.Sprintf("pg_dsn: %q", pgDSN),
	}, "\n")
}

func renderAttachmentsConfig(prepared *preparedOptions, scyllaHost, natsAddr, keydbAddr, pgDSN string) string {
	lines := []string{
		"# Attachments Service Config",
		"",
		"# Network",
		`server_address: ":3200"`,
		"",
		"# Auth",
		fmt.Sprintf("auth_secret: %q", prepared.AuthSecret),
		"",
		"# Cassandra",
		fmt.Sprintf("cluster: [%q]", scyllaHost),
		`cluster_keyspace: "gochat"`,
		"",
		"# NATS",
		fmt.Sprintf("nats_conn_string: %q", natsAddr),
		"",
		"# Redis",
		fmt.Sprintf("keydb: %q", keydbAddr),
		"",
		"# S3",
		fmt.Sprintf("s3_endpoint: %q", prepared.storageEndpointURL),
		fmt.Sprintf("s3_access_key_id: %q", prepared.storageAccessKey),
		fmt.Sprintf("s3_secret_access_key: %q", prepared.storageSecretKey),
		fmt.Sprintf("s3_bucket: %q", prepared.StorageBucket),
		fmt.Sprintf("s3_external_url: %q", prepared.storagePublicBaseURL),
		fmt.Sprintf("s3_use_ssl: %s", boolText(prepared.storageUseSSL)),
	}
	if prepared.storageRegion != "" {
		lines = append(lines, fmt.Sprintf("s3_region: %q", prepared.storageRegion))
	}
	lines = append(lines,
		"",
		"# PostgreSQL",
		fmt.Sprintf("pg_dsn: %q", pgDSN),
		"pg_retries: 5",
	)
	return strings.Join(lines, "\n")
}

func renderWSConfig(prepared *preparedOptions, scyllaHost, pgDSN, keydbAddr, natsAddr string) string {
	return strings.Join([]string{
		"# Auth",
		fmt.Sprintf("auth_secret: %q", prepared.AuthSecret),
		"",
		"# Cassandra",
		fmt.Sprintf("cluster: [%q]", scyllaHost),
		`cluster_keyspace: "gochat"`,
		"",
		"# PostgreSQL",
		fmt.Sprintf("pg_dsn: %q", pgDSN),
		"",
		"# NATS",
		fmt.Sprintf("nats_conn_string: %q", natsAddr),
		"",
		"# Cache Address (KeyDB)",
		fmt.Sprintf("cache_addr: %q", keydbAddr),
	}, "\n")
}

func renderWebhookConfig(prepared *preparedOptions, scyllaHost, keydbAddr, natsAddr, etcdEndpoint string) string {
	return strings.Join([]string{
		`server_address: ":3200"`,
		"api_log: true",
		"swagger: false",
		"",
		"# Discovery for manually managed SFU heartbeats",
		"etcd_endpoints:",
		fmt.Sprintf("  - %q", etcdEndpoint),
		`etcd_prefix: "/gochat/sfu"`,
		`etcd_username: "root"`,
		fmt.Sprintf("etcd_password: %q", prepared.EtcdRootPassword),
		"",
		"# Cassandra",
		fmt.Sprintf("cluster: [%q]", scyllaHost),
		`cluster_keyspace: "gochat"`,
		"",
		"# Webhook JWT secret",
		fmt.Sprintf("jwt_secret: %q", prepared.WebhookJWTSecret),
		"",
		"# Redis",
		fmt.Sprintf("keydb: %q", keydbAddr),
		"",
		"# NATS",
		fmt.Sprintf("nats_conn_string: %q", natsAddr),
	}, "\n")
}

func renderIndexerConfig(prepared *preparedOptions, natsAddr, opensearchAddr string) string {
	return strings.Join([]string{
		"# NATS",
		fmt.Sprintf("nats_conn_string: %q", natsAddr),
		"",
		"# OpenSearch",
		"os_insecure_skip_verify: true",
		fmt.Sprintf("os_addresses: [%q]", opensearchAddr),
		`os_username: "admin"`,
		fmt.Sprintf("os_password: %q", prepared.OpensearchAdminPassword),
	}, "\n")
}

func renderEmbedderConfig(scyllaHost, natsAddr, keydbAddr string) string {
	return strings.Join([]string{
		"# Cassandra",
		fmt.Sprintf("cluster: [%q]", scyllaHost),
		`cluster_keyspace: "gochat"`,
		"",
		"# NATS",
		fmt.Sprintf("nats_conn_string: %q", natsAddr),
		"",
		"# Redis",
		fmt.Sprintf("keydb: %q", keydbAddr),
		`cache_ttl: "6h"`,
		`negative_cache_ttl: "30m"`,
		"excluded_url_patterns: []",
		"",
		"# Fetching",
		`fetch_timeout: "10s"`,
		"max_body_bytes: 2097152",
		"allow_private_hosts: false",
		`youtube_oembed_endpoint: "https://www.youtube.com/oembed"`,
		`youtube_embed_base_url: "https://www.youtube.com/embed"`,
	}, "\n")
}

func renderHelmValues(prepared *preparedOptions, configs map[string]string) string {
	ingressHosts := []string{
		fmt.Sprintf(`  - host: %s
    paths:
      - path: /api/v1/auth
        pathType: Prefix
        service: auth
        port: 3100
      - path: /api/v1/upload
        pathType: Prefix
        service: attachments
        port: 3200
      - path: /emoji
        pathType: Prefix
        service: attachments
        port: 3200
      - path: /api/v1/webhook
        pathType: Prefix
        service: webhook
        port: 3200
      - path: /api/v1
        pathType: Prefix
        service: api
        port: 3100
      - path: /
        pathType: Prefix
        service: ui
        port: 80`, prepared.AppHost),
	}
	if prepared.StorageMode == StorageMinIO {
		ingressHosts = append(ingressHosts,
			fmt.Sprintf(`  - host: %s
    paths:
      - path: /
        pathType: Prefix
        service: minio
        port: 9000`, prepared.StorageHost),
			fmt.Sprintf(`  - host: %s
    paths:
      - path: /
        pathType: Prefix
        service: minio-console
        port: 9001`, prepared.MinIOConsoleHost),
		)
	}

	tlsBlock := "  tls: []"
	if prepared.useTLS && prepared.TLSSecretName != "" {
		hosts := uniqueStrings([]string{prepared.AppHost, prepared.APIHost, prepared.WSHost})
		if prepared.StorageMode == StorageMinIO {
			hosts = append(hosts, prepared.StorageHost, prepared.MinIOConsoleHost)
		}
		tlsLines := make([]string, 0, len(hosts))
		for _, host := range uniqueStrings(hosts) {
			tlsLines = append(tlsLines, "      - "+host)
		}
		tlsBlock = strings.Join([]string{
			"  tls:",
			"    - secretName: " + prepared.TLSSecretName,
			"      hosts:",
			strings.Join(tlsLines, "\n"),
		}, "\n")
	}

	lines := []string{
		"auth:",
		"  image:",
		"    repository: " + imageRepository(prepared.imageAuth),
		fmt.Sprintf("    tag: %q", prepared.backendTag),
		renderHelmOtelEnvBlock(prepared),
		"  config: |",
		indent(configs["auth"], 4),
		"",
		"api:",
		"  image:",
		"    repository: " + imageRepository(prepared.imageAPI),
		fmt.Sprintf("    tag: %q", prepared.backendTag),
		renderHelmOtelEnvBlock(prepared),
		"  config: |",
		indent(configs["api"], 4),
		"",
		"attachments:",
		"  image:",
		"    repository: " + imageRepository(prepared.imageAttachments),
		fmt.Sprintf("    tag: %q", prepared.backendTag),
		renderHelmOtelEnvBlock(prepared),
		"  config: |",
		indent(configs["attachments"], 4),
		"",
		"ws:",
		"  image:",
		"    repository: " + imageRepository(prepared.imageWS),
		fmt.Sprintf("    tag: %q", prepared.backendTag),
		renderHelmOtelEnvBlock(prepared),
		"  config: |",
		indent(configs["ws"], 4),
		"",
		"webhook:",
		"  image:",
		"    repository: " + imageRepository(prepared.imageWebhook),
		fmt.Sprintf("    tag: %q", prepared.backendTag),
		renderHelmOtelEnvBlock(prepared),
		"  config: |",
		indent(configs["webhook"], 4),
		"",
		"indexer:",
		"  image:",
		"    repository: " + imageRepository(prepared.imageIndexer),
		fmt.Sprintf("    tag: %q", prepared.backendTag),
		renderHelmOtelEnvBlock(prepared),
		"  config: |",
		indent(configs["indexer"], 4),
		"",
		"embedder:",
		"  image:",
		"    repository: " + imageRepository(prepared.imageEmbedder),
		fmt.Sprintf("    tag: %q", prepared.backendTag),
		renderHelmOtelEnvBlock(prepared),
		"  config: |",
		indent(configs["embedder"], 4),
		"",
		"ui:",
		"  image:",
		"    repository: nginx",
		`    tag: "alpine"`,
		"  build:",
		"    enabled: true",
		"    repository: https://github.com/FlameInTheDark/gochat-react.git",
		fmt.Sprintf("    ref: %q", prepared.frontendTag),
		"    command: |",
		"      bun install --frozen-lockfile",
		"      bun run build",
		"  env:",
		fmt.Sprintf("    VITE_WEBSOCKET_URL: %q", prepared.wsPublicURL),
		fmt.Sprintf("    VITE_API_BASE_URL: %q", prepared.apiPublicBaseURL),
		`    VITE_BASE_PATH: "/"`,
		"",
		"migrations:",
		"  image:",
		"    repository: " + imageRepository(prepared.imageMigrations),
		fmt.Sprintf("    tag: %q", prepared.MigrationsImageTag),
		"",
		"citus:",
		"  auth:",
		fmt.Sprintf("    postgresPassword: %q", prepared.PostgresPassword),
		"    hostAuthMethod: md5",
		"",
		"etcd:",
		"  env:",
		fmt.Sprintf("    rootPassword: %q", prepared.EtcdRootPassword),
		"",
		"opensearch:",
		"  env:",
		`    OPENSEARCH_JAVA_OPTS: -Xms1g -Xmx1g`,
		fmt.Sprintf("    OPENSEARCH_INITIAL_ADMIN_PASSWORD: %q", prepared.OpensearchAdminPassword),
		"",
		"minio:",
		fmt.Sprintf("  enabled: %s", boolText(prepared.StorageMode == StorageMinIO)),
		fmt.Sprintf("  rootUser: %q", prepared.MinIORootUser),
		fmt.Sprintf("  rootPassword: %q", prepared.MinIORootPassword),
		fmt.Sprintf("  bucket: %q", prepared.StorageBucket),
		"",
		"routing:",
		fmt.Sprintf("  appHost: %q", prepared.AppHost),
		"",
		"traefik:",
		fmt.Sprintf("  enabled: %s", boolText(prepared.useBundledTraefik)),
		"",
		"ingress:",
		"  enabled: true",
		fmt.Sprintf("  className: %q", prepared.IngressClassName),
		"  annotations: {}",
		"  hosts:",
		strings.Join(ingressHosts, "\n"),
	}

	if !prepared.useBundledTraefik {
		lines = append(lines, renderHelmWebsocketIngressBlock(prepared))
	}

	lines = append(lines,
		tlsBlock,
		"",
		"observability:",
		"  enabled: true",
		"  deploymentEnv: kubernetes",
		"  openobserve:",
		"    env:",
		fmt.Sprintf("      rootUserEmail: %q", prepared.openObserveRootEmail),
		fmt.Sprintf("      rootUserPassword: %q", prepared.openObserveRootPassword),
		renderOpenObserveIngressBlock(prepared),
		"  otelCollector:",
		"    env:",
		fmt.Sprintf("      openobserveOrg: %s", prepared.openObserveOrg),
		fmt.Sprintf("      openobserveBasicAuth: %s", prepared.openObserveBasicAuth),
		fmt.Sprintf("      openobserveLogStream: %s", prepared.openObserveLogStream),
		fmt.Sprintf("      openobserveMetricStream: %s", prepared.openObserveMetricStream),
		fmt.Sprintf("      openobserveTraceStream: %s", prepared.openObserveTraceStream),
	)

	return strings.Join(lines, "\n")
}

func renderHelmOtelEnvBlock(prepared *preparedOptions) string {
	return strings.Join([]string{
		"  env:",
		"    - name: GOCHAT_DEPLOYMENT_ENV",
		`      value: "kubernetes"`,
		"    - name: OTEL_EXPORTER_OTLP_ENDPOINT",
		fmt.Sprintf("      value: %q", "http://"+prepared.helmFullName+"-otel-collector:4318"),
		"    - name: OTEL_EXPORTER_OTLP_PROTOCOL",
		`      value: "http/protobuf"`,
	}, "\n")
}

func renderHelmWebsocketIngressBlock(prepared *preparedOptions) string {
	lines := []string{
		"  websocket:",
		"    enabled: true",
	}
	if strings.EqualFold(prepared.IngressClassName, "nginx") {
		lines = append(lines,
			"    annotations:",
			`      nginx.ingress.kubernetes.io/proxy-http-version: "1.1"`,
			`      nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"`,
			`      nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"`,
			`      nginx.ingress.kubernetes.io/rewrite-target: /subscribe`,
		)
	} else {
		lines = append(lines, "    annotations: {}")
	}

	lines = append(lines,
		"  websocketAlias:",
		"    enabled: true",
	)
	if strings.EqualFold(prepared.IngressClassName, "nginx") {
		lines = append(lines,
			"    annotations:",
			`      nginx.ingress.kubernetes.io/proxy-http-version: "1.1"`,
			`      nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"`,
			`      nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"`,
			`      nginx.ingress.kubernetes.io/rewrite-target: /subscribe`,
		)
	} else {
		lines = append(lines, "    annotations: {}")
	}

	return strings.Join(lines, "\n")
}

func renderOpenObserveIngressBlock(prepared *preparedOptions) string {
	if prepared.OpenObserveHost == "" {
		return strings.Join([]string{
			"    ingress:",
			"      enabled: false",
		}, "\n")
	}

	lines := []string{
		"    ingress:",
		"      enabled: true",
		fmt.Sprintf("      className: %q", prepared.IngressClassName),
		"      annotations: {}",
		fmt.Sprintf("      host: %q", prepared.OpenObserveHost),
		`      path: "/"`,
		`      pathType: "Prefix"`,
	}
	if prepared.useTLS && prepared.TLSSecretName != "" {
		lines = append(lines,
			"      tls:",
			"        - secretName: "+prepared.TLSSecretName,
			"          hosts:",
			"            - "+prepared.OpenObserveHost,
		)
	}
	return strings.Join(lines, "\n")
}
