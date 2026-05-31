package deployer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	cli "github.com/urfave/cli/v3"
)

func newCLI(engine *Engine) *cli.Command {
	return &cli.Command{
		Name:  "gochat-deployer",
		Usage: "Deploy GoChat with an embedded single-binary workflow",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runWizard(ctx, engine, commandOptions(cmd))
		},
		Commands: []*cli.Command{
			{
				Name:  "check",
				Usage: "Check required deployment tools",
				Flags: commonFlags(false),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					opts := commandOptions(cmd)
					report := engine.Check(ctx, opts)
					if err := printCheckReportTo(os.Stdout, report); err != nil {
						return err
					}
					if missing := report.MissingRequired(); len(missing) > 0 {
						names := make([]string, 0, len(missing))
						for _, item := range missing {
							names = append(names, item.Name)
						}
						return fmt.Errorf("missing required tools: %s", strings.Join(names, ", "))
					}
					return nil
				},
			},
			{
				Name:  "export",
				Usage: "Export the embedded deployment assets into the workspace",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "workspace-root", Usage: "Workspace root for the exported bundle", Value: ".gochat-deployer/workspace"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					workspaceRoot := cmd.String("workspace-root")
					if err := engine.ExportBundle(ctx, workspaceRoot); err != nil {
						return err
					}
					if _, err := fmt.Fprintf(cmd.Writer, "Exported embedded assets to %s\n", workspaceRoot); err != nil {
						return fmt.Errorf("write export result: %w", err)
					}
					return nil
				},
			},
			tokensCommand(),
			{
				Name:  "render",
				Usage: "Render deployment config without running docker compose or helm",
				Flags: commonFlags(true),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					opts := commandOptions(cmd)
					opts.Action = ActionRender
					result, err := engine.Render(ctx, opts, cmd.Writer)
					if err != nil {
						return err
					}
					return printSummaryTo(os.Stdout, result, true)
				},
			},
			{
				Name:  "deploy",
				Usage: "Render deployment config and execute docker compose or helm",
				Flags: commonFlags(true),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					opts := commandOptions(cmd)
					opts.Action = ActionDeploy
					result, err := engine.Deploy(ctx, opts, cmd.Writer)
					if err != nil {
						return err
					}
					return printSummaryTo(os.Stdout, result, false)
				},
			},
			{
				Name:  "wizard",
				Usage: "Run the interactive terminal wizard",
				Flags: commonFlags(false),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runWizard(ctx, engine, commandOptions(cmd))
				},
			},
		},
	}
}

func commonFlags(requireDeployment bool) []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{Name: "deployment-type", Usage: "Deployment type: compose or helm"},
		&cli.StringFlag{Name: "storage-mode", Usage: "Storage mode: minio or external"},
		&cli.StringFlag{Name: "base-domain", Usage: "Base domain used to derive app and storage hosts"},
		&cli.StringFlag{Name: "app-host", Usage: "Public app host"},
		&cli.StringFlag{Name: "api-host", Usage: "Public API host override"},
		&cli.StringFlag{Name: "ws-host", Usage: "Public websocket host override"},
		&cli.StringFlag{Name: "telemetry-host", Usage: "Public telemetry gateway host"},
		&cli.StringFlag{Name: "storage-host", Usage: "Public storage host"},
		&cli.StringFlag{Name: "minio-console-host", Usage: "Public MinIO console host"},
		&cli.StringFlag{Name: "namespace", Usage: "Kubernetes namespace", Value: "gochat"},
		&cli.StringFlag{Name: "release-name", Usage: "Helm release name", Value: "gochat"},
		&cli.StringFlag{Name: "workspace-root", Usage: "Workspace root used for embedded assets and generated files", Value: ".gochat-deployer/workspace"},
		&cli.StringFlag{Name: "openobserve-host", Usage: "Public OpenObserve host for Helm; leave empty to keep it internal"},
		&cli.StringFlag{Name: "image-repository-prefix", Usage: "Image repository prefix", Value: "ghcr.io/flameinthedark"},
		&cli.StringFlag{Name: "backend-tag", Usage: "Backend image tag override; latest stable backend tag is resolved automatically when omitted"},
		&cli.StringFlag{Name: "frontend-tag", Usage: "Frontend image tag override; latest stable frontend tag is resolved automatically when omitted"},
		&cli.StringFlag{Name: "migrations-image-repository", Usage: "Migrations image repository override; defaults to <image-repository-prefix>/gochat-migrations"},
		&cli.StringFlag{Name: "migrations-image-tag", Usage: "Migrations image tag override; defaults to the resolved backend tag"},
		&cli.BoolFlag{Name: "tls", Usage: "Force TLS-aware public URLs for Helm"},
		&cli.BoolFlag{Name: "no-tls", Usage: "Disable TLS-aware public URLs for Helm"},
		&cli.BoolFlag{Name: "bundled-traefik", Usage: "Force bundled Traefik for Helm"},
		&cli.BoolFlag{Name: "no-bundled-traefik", Usage: "Disable bundled Traefik for Helm"},
		&cli.StringFlag{Name: "ingress-class-name", Usage: "Ingress class name for Helm"},
		&cli.StringFlag{Name: "tls-secret-name", Usage: "TLS secret name for Helm"},
		&cli.StringFlag{Name: "auth-secret", Usage: "Application auth secret"},
		&cli.StringFlag{Name: "mfa-encryption-key", Usage: "Base64-encoded 32-byte MFA encryption key"},
		&cli.StringFlag{Name: "webhook-jwt-secret", Usage: "Webhook JWT secret"},
		&cli.StringFlag{Name: "yugabyte-host", Usage: "YugabyteDB YSQL host; defaults to yugabyte for Compose and yb-tservers.gochat-yb.svc.cluster.local for Helm"},
		&cli.IntFlag{Name: "yugabyte-port", Usage: "YugabyteDB YSQL port", Value: 5433},
		&cli.StringFlag{Name: "yugabyte-user", Usage: "YugabyteDB YSQL user", Value: "yugabyte"},
		&cli.StringFlag{Name: "yugabyte-password", Usage: "YugabyteDB YSQL password", Value: "yugabyte"},
		&cli.StringFlag{Name: "yugabyte-database", Usage: "YugabyteDB YSQL database", Value: "gochat"},
		&cli.StringFlag{Name: "yugabyte-sslmode", Usage: "YugabyteDB YSQL sslmode", Value: "disable"},
		&cli.StringFlag{Name: "yugabyte-namespace", Usage: "Kubernetes namespace for the YugabyteDB Helm release", Value: defaultYugabyteNamespace},
		&cli.StringFlag{Name: "yugabyte-release-name", Usage: "YugabyteDB Helm release name", Value: defaultYugabyteReleaseName},
		&cli.StringFlag{Name: "yugabyte-chart-version", Usage: "YugabyteDB Helm chart version", Value: defaultYugabyteChartVersion},
		&cli.StringFlag{Name: "yugabyte-image-tag", Usage: "YugabyteDB image tag", Value: defaultYugabyteImageTag},
		&cli.IntFlag{Name: "yugabyte-tserver-count", Usage: "YugabyteDB tserver node count", Value: defaultYugabyteTServerCount},
		&cli.IntFlag{Name: "yugabyte-replication-factor", Usage: "YugabyteDB replication factor; also controls master count", Value: defaultYugabyteReplicationFactor},
		&cli.StringFlag{Name: "yugabyte-storage-class", Usage: "YugabyteDB storage class; empty uses the cluster default"},
		&cli.StringFlag{Name: "yugabyte-master-storage-size", Usage: "YugabyteDB master PVC size", Value: defaultYugabyteMasterStorageSize},
		&cli.StringFlag{Name: "yugabyte-tserver-storage-size", Usage: "YugabyteDB tserver PVC size", Value: defaultYugabyteTServerStorageSize},
		&cli.StringFlag{Name: "yugabyte-master-cpu", Usage: "YugabyteDB master CPU request", Value: defaultYugabyteMasterCPU},
		&cli.StringFlag{Name: "yugabyte-master-memory", Usage: "YugabyteDB master memory request/limit", Value: defaultYugabyteMasterMemory},
		&cli.StringFlag{Name: "yugabyte-tserver-cpu", Usage: "YugabyteDB tserver CPU request", Value: defaultYugabyteTServerCPU},
		&cli.StringFlag{Name: "yugabyte-tserver-memory", Usage: "YugabyteDB tserver memory request/limit", Value: defaultYugabyteTServerMemory},
		&cli.StringFlag{Name: "scylla-hosts", Usage: "Comma-separated ScyllaDB contact points; Helm defaults to the operator-managed client service"},
		&cli.StringFlag{Name: "scylla-namespace", Usage: "Kubernetes namespace for the ScyllaDB Helm release", Value: defaultScyllaNamespace},
		&cli.StringFlag{Name: "scylla-release-name", Usage: "ScyllaDB Helm release name and fullname override", Value: defaultScyllaReleaseName},
		&cli.StringFlag{Name: "scylla-chart-version", Usage: "ScyllaDB Helm chart and operator version", Value: defaultScyllaChartVersion},
		&cli.StringFlag{Name: "scylla-image-tag", Usage: "ScyllaDB image tag", Value: defaultScyllaImageTag},
		&cli.IntFlag{Name: "scylla-node-count", Usage: "ScyllaDB node count", Value: defaultScyllaNodeCount},
		&cli.IntFlag{Name: "scylla-replication-factor", Usage: "ScyllaDB keyspace replication factor", Value: defaultScyllaReplicationFactor},
		&cli.StringFlag{Name: "scylla-datacenter", Usage: "ScyllaDB datacenter name for NetworkTopologyStrategy", Value: defaultScyllaDatacenter},
		&cli.StringFlag{Name: "scylla-storage-class", Usage: "ScyllaDB storage class; empty uses the cluster default"},
		&cli.StringFlag{Name: "scylla-storage-size", Usage: "ScyllaDB PVC size per node", Value: defaultScyllaStorageSize},
		&cli.StringFlag{Name: "scylla-cpu", Usage: "ScyllaDB CPU request/limit per node", Value: defaultScyllaCPU},
		&cli.StringFlag{Name: "scylla-memory", Usage: "ScyllaDB memory request/limit per node", Value: defaultScyllaMemory},
		&cli.StringFlag{Name: "etcd-root-password", Usage: "etcd root password"},
		&cli.StringFlag{Name: "opensearch-admin-password", Usage: "OpenSearch admin password"},
		&cli.StringFlag{Name: "openobserve-root-email", Usage: "OpenObserve admin email; required for render and deploy"},
		&cli.StringFlag{Name: "openobserve-root-password", Usage: "OpenObserve admin password; required for render and deploy"},
		&cli.StringFlag{Name: "storage-bucket", Usage: "Storage bucket name", Value: "gochat"},
		&cli.StringFlag{Name: "minio-root-user", Usage: "Bundled MinIO root user"},
		&cli.StringFlag{Name: "minio-root-password", Usage: "Bundled MinIO root password"},
		&cli.StringFlag{Name: "external-s3-endpoint", Usage: "External S3 endpoint"},
		&cli.StringFlag{Name: "external-s3-public-base-url", Usage: "External S3 public base URL"},
		&cli.StringFlag{Name: "external-s3-access-key-id", Usage: "External S3 access key id"},
		&cli.StringFlag{Name: "external-s3-secret-access-key", Usage: "External S3 secret access key"},
		&cli.StringFlag{Name: "external-s3-region", Usage: "External S3 region", Value: "us-east-1"},
		&cli.BoolFlag{Name: "external-s3-use-ssl", Usage: "Use HTTPS when normalizing the external S3 endpoint"},
		&cli.StringFlag{Name: "email-provider", Usage: "Email provider: log, smtp, sendpulse, resend, or dashamail", Value: "log"},
		&cli.StringFlag{Name: "email-source", Usage: "Email sender address", Value: "no-reply@example.com"},
		&cli.StringFlag{Name: "email-name", Usage: "Email sender display name", Value: "GoChat"},
		&cli.StringFlag{Name: "sendpulse-user-id", Usage: "SendPulse user ID"},
		&cli.StringFlag{Name: "sendpulse-secret", Usage: "SendPulse secret"},
		&cli.StringFlag{Name: "resend-api-key", Usage: "Resend API key"},
		&cli.StringFlag{Name: "dashamail-api-key", Usage: "DashaMail API key"},
		&cli.StringFlag{Name: "smtp-host", Usage: "SMTP host"},
		&cli.IntFlag{Name: "smtp-port", Usage: "SMTP port", Value: 2525},
		&cli.StringFlag{Name: "smtp-username", Usage: "SMTP username"},
		&cli.StringFlag{Name: "smtp-password", Usage: "SMTP password"},
		&cli.BoolFlag{Name: "smtp-use-tls", Usage: "Use TLS for SMTP"},
	}

	if requireDeployment {
		flags = append(flags, &cli.BoolFlag{Name: "non-interactive", Usage: "Marker flag for explicit CLI deployment"})
	}
	return flags
}

func commandOptions(cmd *cli.Command) Options {
	return Options{
		DeploymentType:             DeploymentType(cmd.String("deployment-type")),
		StorageMode:                StorageMode(cmd.String("storage-mode")),
		BaseDomain:                 cmd.String("base-domain"),
		AppHost:                    cmd.String("app-host"),
		APIHost:                    cmd.String("api-host"),
		WSHost:                     cmd.String("ws-host"),
		TelemetryHost:              cmd.String("telemetry-host"),
		StorageHost:                cmd.String("storage-host"),
		MinIOConsoleHost:           cmd.String("minio-console-host"),
		Namespace:                  cmd.String("namespace"),
		ReleaseName:                cmd.String("release-name"),
		WorkspaceRoot:              cmd.String("workspace-root"),
		OpenObserveHost:            cmd.String("openobserve-host"),
		IngressClassName:           cmd.String("ingress-class-name"),
		TLSSecretName:              cmd.String("tls-secret-name"),
		ImageRepositoryPrefix:      cmd.String("image-repository-prefix"),
		BackendTag:                 cmd.String("backend-tag"),
		FrontendTag:                cmd.String("frontend-tag"),
		MigrationsImageRepo:        cmd.String("migrations-image-repository"),
		MigrationsImageTag:         cmd.String("migrations-image-tag"),
		UseTLS:                     boolToToggle(cmd.Bool("tls"), cmd.Bool("no-tls")),
		UseBundledTraefik:          boolToToggle(cmd.Bool("bundled-traefik"), cmd.Bool("no-bundled-traefik")),
		AuthSecret:                 cmd.String("auth-secret"),
		MFAEncryptionKey:           cmd.String("mfa-encryption-key"),
		WebhookJWTSecret:           cmd.String("webhook-jwt-secret"),
		YugabyteHost:               cmd.String("yugabyte-host"),
		YugabytePort:               cmd.Int("yugabyte-port"),
		YugabyteUser:               cmd.String("yugabyte-user"),
		YugabytePassword:           cmd.String("yugabyte-password"),
		YugabyteDatabase:           cmd.String("yugabyte-database"),
		YugabyteSSLMode:            cmd.String("yugabyte-sslmode"),
		YugabyteNamespace:          cmd.String("yugabyte-namespace"),
		YugabyteReleaseName:        cmd.String("yugabyte-release-name"),
		YugabyteChartVersion:       cmd.String("yugabyte-chart-version"),
		YugabyteImageTag:           cmd.String("yugabyte-image-tag"),
		YugabyteTServerCount:       cmd.Int("yugabyte-tserver-count"),
		YugabyteReplicationFactor:  cmd.Int("yugabyte-replication-factor"),
		YugabyteStorageClass:       cmd.String("yugabyte-storage-class"),
		YugabyteMasterStorageSize:  cmd.String("yugabyte-master-storage-size"),
		YugabyteTServerStorageSize: cmd.String("yugabyte-tserver-storage-size"),
		YugabyteMasterCPU:          cmd.String("yugabyte-master-cpu"),
		YugabyteMasterMemory:       cmd.String("yugabyte-master-memory"),
		YugabyteTServerCPU:         cmd.String("yugabyte-tserver-cpu"),
		YugabyteTServerMemory:      cmd.String("yugabyte-tserver-memory"),
		ScyllaHosts:                cmd.String("scylla-hosts"),
		ScyllaNamespace:            cmd.String("scylla-namespace"),
		ScyllaReleaseName:          cmd.String("scylla-release-name"),
		ScyllaChartVersion:         cmd.String("scylla-chart-version"),
		ScyllaImageTag:             cmd.String("scylla-image-tag"),
		ScyllaNodeCount:            cmd.Int("scylla-node-count"),
		ScyllaReplicationFactor:    cmd.Int("scylla-replication-factor"),
		ScyllaDatacenter:           cmd.String("scylla-datacenter"),
		ScyllaStorageClass:         cmd.String("scylla-storage-class"),
		ScyllaStorageSize:          cmd.String("scylla-storage-size"),
		ScyllaCPU:                  cmd.String("scylla-cpu"),
		ScyllaMemory:               cmd.String("scylla-memory"),
		EtcdRootPassword:           cmd.String("etcd-root-password"),
		OpensearchAdminPassword:    cmd.String("opensearch-admin-password"),
		OpenObserveRootEmail:       cmd.String("openobserve-root-email"),
		OpenObserveRootPassword:    cmd.String("openobserve-root-password"),
		StorageBucket:              cmd.String("storage-bucket"),
		MinIORootUser:              cmd.String("minio-root-user"),
		MinIORootPassword:          cmd.String("minio-root-password"),
		ExternalS3Endpoint:         cmd.String("external-s3-endpoint"),
		ExternalS3PublicBaseURL:    cmd.String("external-s3-public-base-url"),
		ExternalS3AccessKeyID:      cmd.String("external-s3-access-key-id"),
		ExternalS3SecretAccessKey:  cmd.String("external-s3-secret-access-key"),
		ExternalS3Region:           cmd.String("external-s3-region"),
		ExternalS3UseSSL:           cmd.Bool("external-s3-use-ssl"),
		EmailProvider:              cmd.String("email-provider"),
		EmailSource:                cmd.String("email-source"),
		EmailName:                  cmd.String("email-name"),
		SendpulseUserID:            cmd.String("sendpulse-user-id"),
		SendpulseSecret:            cmd.String("sendpulse-secret"),
		ResendAPIKey:               cmd.String("resend-api-key"),
		DashaMailAPIKey:            cmd.String("dashamail-api-key"),
		SMTPHost:                   cmd.String("smtp-host"),
		SMTPPort:                   cmd.Int("smtp-port"),
		SMTPUsername:               cmd.String("smtp-username"),
		SMTPPassword:               cmd.String("smtp-password"),
		SMTPUseTLS:                 cmd.Bool("smtp-use-tls"),
	}
}

func boolToToggle(on, off bool) ToggleMode {
	if on {
		return ToggleOn
	}
	if off {
		return ToggleOff
	}
	return ToggleAuto
}

func printCheckReportTo(output io.Writer, report CheckReport) error {
	if output == nil {
		output = io.Discard
	}
	if _, err := fmt.Fprintln(output, "Tool                  Required  Status  Detail"); err != nil {
		return fmt.Errorf("write check report header: %w", err)
	}
	if _, err := fmt.Fprintln(output, "----                  --------  ------  ------"); err != nil {
		return fmt.Errorf("write check report separator: %w", err)
	}
	for _, item := range report.Items {
		required := "no"
		if item.Required {
			required = "yes"
		}
		status := "missing"
		if item.Present {
			status = "ok"
		}
		if _, err := fmt.Fprintf(output, "%-20s  %-8s  %-6s  %s\n", item.Name, required, status, item.Detail); err != nil {
			return fmt.Errorf("write check report item: %w", err)
		}
	}
	return nil
}

func printSummaryTo(output io.Writer, result RenderResult, includeDeployHints bool) error {
	if output == nil {
		output = io.Discard
	}
	if _, err := fmt.Fprintln(output); err != nil {
		return fmt.Errorf("write summary spacer: %w", err)
	}
	for _, line := range result.SummaryLines() {
		if _, err := fmt.Fprintln(output, line); err != nil {
			return fmt.Errorf("write summary line: %w", err)
		}
	}
	for _, line := range []string{
		"Compose env: " + result.ComposeEnvPath,
		"Compose config: " + result.ComposeConfigRoot,
		"Helm values: " + result.HelmValuesPath,
		"Scylla values: " + result.ScyllaValuesPath,
		"Yugabyte values: " + result.YugabyteValuesPath,
	} {
		if _, err := fmt.Fprintln(output, line); err != nil {
			return fmt.Errorf("write summary path: %w", err)
		}
	}
	if includeDeployHints {
		for _, line := range []string{
			"",
			"Deploy from rendered data:",
			"Scylla Operator: " + result.ScyllaOperatorDeployCommand,
			"ScyllaDB: " + result.ScyllaDeployCommand,
			"YugabyteDB: " + result.YugabyteDeployCommand,
			"Docker Compose: " + result.ComposeDeployCommand,
			"Helm: " + result.HelmDeployCommand,
		} {
			if _, err := fmt.Fprintln(output, line); err != nil {
				return fmt.Errorf("write deployment hint: %w", err)
			}
		}
	}
	return nil
}
