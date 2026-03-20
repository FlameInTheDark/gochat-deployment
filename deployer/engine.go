package deployer

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Engine struct {
	bundle           fs.FS
	httpClient       *http.Client
	githubAPIBaseURL string
	backendRepo      gitHubRepo
	frontendRepo     gitHubRepo
}

type preparedOptions struct {
	Options

	useTLS            bool
	useBundledTraefik bool
	httpScheme        string
	wsScheme          string

	appPublicURL         string
	apiPublicBaseURL     string
	wsPublicURL          string
	storageEndpointURL   string
	storagePublicBaseURL string
	storageAccessKey     string
	storageSecretKey     string
	storageRegion        string
	storageUseSSL        bool
	storageAlias         string
	minioConsoleAlias    string

	contentHosts            []string
	helmFullName            string
	composePGAddr           string
	composePGDSN            string
	composeCassandraAddr    string
	openObserveRootEmail    string
	openObserveRootPassword string
	openObserveOrg          string
	openObserveBasicAuth    string
	openObserveLogStream    string
	openObserveMetricStream string
	openObserveTraceStream  string
	sfuServiceID            string
	sfuWebhookToken         string
	backendTag              string
	frontendTag             string
	imageAPI                string
	imageAuth               string
	imageAttachments        string
	imageWS                 string
	imageWebhook            string
	imageIndexer            string
	imageEmbedder           string
	imageUI                 string
	imageMigrations         string
}

func NewEngine(bundle fs.FS) *Engine {
	return &Engine{
		bundle:           bundle,
		httpClient:       defaultHTTPClient(),
		githubAPIBaseURL: "https://api.github.com",
		backendRepo: gitHubRepo{
			Owner: "FlameInTheDark",
			Name:  "gochat",
		},
		frontendRepo: gitHubRepo{
			Owner: "FlameInTheDark",
			Name:  "gochat-react",
		},
	}
}

func (e *Engine) Check(ctx context.Context, opts Options) CheckReport {
	report := CheckReport{}

	add := func(name string, required bool, args ...string) {
		status := ToolStatus{Name: name, Required: required}
		path, err := exec.LookPath(name)
		if err != nil {
			status.Detail = err.Error()
			report.Items = append(report.Items, status)
			return
		}

		status.Present = true
		status.Detail = path
		if len(args) > 0 {
			cmd := exec.CommandContext(ctx, name, args...)
			if output, err := cmd.CombinedOutput(); err == nil {
				if text := strings.TrimSpace(string(output)); text != "" {
					status.Detail = text
				}
			}
		}
		report.Items = append(report.Items, status)
	}

	if opts.DeploymentType == DeploymentCompose {
		add("docker", true, "compose", "version")
		return report
	}
	if opts.DeploymentType == DeploymentHelm {
		add("helm", true, "version")
		add("kubectl", false, "version", "--client=true")
		return report
	}

	add("docker", false, "compose", "version")
	add("helm", false, "version")
	add("kubectl", false, "version", "--client=true")
	return report
}

func (e *Engine) ExportBundle(_ context.Context, workspaceRoot string) error {
	if workspaceRoot == "" {
		return fmt.Errorf("workspace root is required")
	}
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		return fmt.Errorf("create workspace root: %w", err)
	}

	return fs.WalkDir(e.bundle, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		if !isBundledPath(path) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		target := filepath.Join(workspaceRoot, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, readErr := fs.ReadFile(e.bundle, path)
		if readErr != nil {
			return fmt.Errorf("read embedded asset %s: %w", path, readErr)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create asset dir for %s: %w", target, err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("write embedded asset %s: %w", target, err)
		}
		return nil
	})
}

func (e *Engine) Render(ctx context.Context, opts Options, output io.Writer) (RenderResult, error) {
	prepared, err := e.prepareOptions(ctx, opts)
	if err != nil {
		return RenderResult{}, err
	}
	return e.renderPrepared(ctx, prepared, output)
}

func (e *Engine) renderPrepared(ctx context.Context, prepared *preparedOptions, output io.Writer) (RenderResult, error) {
	if output == nil {
		output = io.Discard
	}

	if err := e.ExportBundle(ctx, prepared.WorkspaceRoot); err != nil {
		return RenderResult{}, err
	}

	composeGeneratedRoot := filepath.Join(prepared.WorkspaceRoot, ".generated", "compose")
	composeConfigRoot := filepath.Join(composeGeneratedRoot, "config")
	helmGeneratedRoot := filepath.Join(prepared.WorkspaceRoot, ".generated", "helm")
	workspaceMigrationsRoot := filepath.Join(prepared.WorkspaceRoot, "helm", "gochat", "files", "migrations")
	for _, dir := range []string{composeGeneratedRoot, composeConfigRoot, helmGeneratedRoot, workspaceMigrationsRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return RenderResult{}, fmt.Errorf("create generated dir %s: %w", dir, err)
		}
	}
	if err := e.syncMigrations(ctx, prepared.backendTag, workspaceMigrationsRoot); err != nil {
		return RenderResult{}, err
	}

	composeConfigs, helmValues := e.renderOutputs(prepared)
	for name, content := range composeConfigs {
		target := filepath.Join(composeConfigRoot, name)
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return RenderResult{}, fmt.Errorf("write %s: %w", target, err)
		}
	}

	composeEnvPath := filepath.Join(composeGeneratedRoot, ".env")
	if err := os.WriteFile(composeEnvPath, []byte(renderComposeEnv(prepared)), 0o644); err != nil {
		return RenderResult{}, fmt.Errorf("write %s: %w", composeEnvPath, err)
	}

	helmValuesPath := filepath.Join(helmGeneratedRoot, "values.generated.yaml")
	if err := os.WriteFile(helmValuesPath, []byte(helmValues), 0o644); err != nil {
		return RenderResult{}, fmt.Errorf("write %s: %w", helmValuesPath, err)
	}

	fmt.Fprintf(output, "[gochat] Rendered bundle into %s\n", prepared.WorkspaceRoot)

	result := RenderResult{
		WorkspaceRoot:     prepared.WorkspaceRoot,
		GeneratedRoot:     filepath.Join(prepared.WorkspaceRoot, ".generated"),
		ComposeEnvPath:    composeEnvPath,
		ComposeConfigRoot: composeConfigRoot,
		ComposeFilePath:   filepath.Join(prepared.WorkspaceRoot, "compose", "docker-compose.yaml"),
		HelmChartPath:     filepath.Join(prepared.WorkspaceRoot, "helm", "gochat"),
		HelmValuesPath:    helmValuesPath,
		BackendTag:        prepared.backendTag,
		FrontendTag:       prepared.frontendTag,
		MigrationsTag:     prepared.backendTag,
		AppPublicURL:      prepared.appPublicURL,
		APIPublicBaseURL:  prepared.apiPublicBaseURL,
		WSPublicURL:       prepared.wsPublicURL,
	}
	result.ComposeDeployCommand = composeDeployCommand(prepared, result)
	result.HelmDeployCommand = helmDeployCommand(prepared, result)
	result.InstructionsPath = deploymentGuidePath(prepared.WorkspaceRoot)

	if prepared.storagePublicBaseURL != "" {
		result.StoragePublicURL = prepared.storagePublicBaseURL
	}
	if prepared.StorageMode == StorageMinIO {
		result.MinIOConsoleURL = fmt.Sprintf("%s://%s", prepared.httpScheme, prepared.MinIOConsoleHost)
	}
	if prepared.OpenObserveHost != "" {
		result.OpenObserveURL = fmt.Sprintf("%s://%s", prepared.httpScheme, prepared.OpenObserveHost)
	}
	if err := writeDeploymentGuide(result.InstructionsPath, prepared, result); err != nil {
		return RenderResult{}, fmt.Errorf("write deployment guide: %w", err)
	}

	return result, nil
}

func (e *Engine) Deploy(ctx context.Context, opts Options, output io.Writer) (RenderResult, error) {
	prepared, err := e.prepareOptions(ctx, opts)
	if err != nil {
		return RenderResult{}, err
	}
	if output == nil {
		output = io.Discard
	}

	report := e.Check(ctx, prepared.Options)
	if missing := report.MissingRequired(); len(missing) > 0 {
		names := make([]string, 0, len(missing))
		for _, item := range missing {
			names = append(names, item.Name)
		}
		sort.Strings(names)
		return RenderResult{}, fmt.Errorf("missing required tools: %s", strings.Join(names, ", "))
	}

	result, err := e.renderPrepared(ctx, prepared, output)
	if err != nil {
		return RenderResult{}, err
	}

	switch prepared.DeploymentType {
	case DeploymentCompose:
		args := []string{
			"compose",
			"--env-file", result.ComposeEnvPath,
			"-f", result.ComposeFilePath,
		}
		if prepared.StorageMode == StorageMinIO {
			args = append(args, "--profile", "minio")
		}
		if err := runCommand(ctx, prepared.WorkspaceRoot, nil, output, "docker", append(args, "pull")...); err != nil {
			return RenderResult{}, err
		}
		if err := runCommand(ctx, prepared.WorkspaceRoot, nil, output, "docker", append(args, "up", "-d")...); err != nil {
			return RenderResult{}, err
		}
	case DeploymentHelm:
		args := []string{
			"upgrade",
			"--install",
			prepared.ReleaseName,
			result.HelmChartPath,
			"--namespace", prepared.Namespace,
			"--create-namespace",
			"--values", result.HelmValuesPath,
		}
		if err := runCommand(ctx, prepared.WorkspaceRoot, helmCommandEnv(prepared.WorkspaceRoot), output, "helm", args...); err != nil {
			return RenderResult{}, err
		}
	default:
		return RenderResult{}, fmt.Errorf("unsupported deployment type %q", prepared.DeploymentType)
	}

	return result, nil
}

func (e *Engine) prepareOptions(ctx context.Context, opts Options) (*preparedOptions, error) {
	if opts.Action == "" {
		opts.Action = ActionDeploy
	}
	if !opts.Action.Valid() {
		return nil, fmt.Errorf("action must be deploy or render")
	}
	if !opts.DeploymentType.Valid() {
		return nil, fmt.Errorf("deployment type must be compose or helm")
	}
	if !opts.StorageMode.Valid() {
		return nil, fmt.Errorf("storage mode must be minio or external")
	}
	if opts.WorkspaceRoot == "" {
		opts.WorkspaceRoot = filepath.Join(".gochat-deployer", "workspace")
	}
	if opts.Namespace == "" {
		opts.Namespace = "gochat"
	}
	if opts.ReleaseName == "" {
		opts.ReleaseName = "gochat"
	}
	if opts.ImageRepositoryPrefix == "" {
		opts.ImageRepositoryPrefix = "ghcr.io/flameinthedark"
	}
	if opts.MigrationsImageRepo == "" {
		opts.MigrationsImageRepo = "migrate/migrate"
	}
	if opts.MigrationsImageTag == "" {
		opts.MigrationsImageTag = "v4.18.3"
	}
	if opts.StorageBucket == "" {
		opts.StorageBucket = "gochat"
	}
	if opts.ExternalS3Region == "" {
		opts.ExternalS3Region = "us-east-1"
	}
	if opts.EmailProvider == "" {
		opts.EmailProvider = "log"
	}
	opts.EmailProvider = strings.ToLower(strings.TrimSpace(opts.EmailProvider))
	if opts.EmailSource == "" {
		opts.EmailSource = "no-reply@example.com"
	}
	if opts.EmailName == "" {
		opts.EmailName = "GoChat"
	}
	opts.OpenObserveRootEmail = strings.TrimSpace(opts.OpenObserveRootEmail)
	opts.BackendTag = strings.TrimSpace(opts.BackendTag)
	opts.FrontendTag = strings.TrimSpace(opts.FrontendTag)
	if opts.SMTPPort == 0 {
		opts.SMTPPort = 2525
	}
	if opts.BaseDomain == "" {
		if opts.DeploymentType == DeploymentCompose {
			opts.BaseDomain = "gochat.local"
		} else {
			opts.BaseDomain = "example.com"
		}
	}

	opts.BaseDomain = normalizeHost(opts.BaseDomain)
	if opts.BaseDomain == "" && opts.AppHost != "" {
		opts.BaseDomain = normalizeHost(opts.AppHost)
	}
	if opts.AppHost == "" {
		opts.AppHost = opts.BaseDomain
	}
	if opts.APIHost == "" {
		opts.APIHost = opts.AppHost
	}
	if opts.WSHost == "" {
		opts.WSHost = opts.AppHost
	}
	if opts.StorageHost == "" && opts.BaseDomain != "" {
		opts.StorageHost = "storage." + opts.BaseDomain
	}
	if opts.MinIOConsoleHost == "" && opts.BaseDomain != "" {
		opts.MinIOConsoleHost = "minio." + opts.BaseDomain
	}

	opts.AppHost = normalizeHost(opts.AppHost)
	opts.APIHost = opts.AppHost
	opts.WSHost = opts.AppHost
	opts.StorageHost = normalizeHost(opts.StorageHost)
	opts.MinIOConsoleHost = normalizeHost(opts.MinIOConsoleHost)
	opts.OpenObserveHost = normalizeHost(opts.OpenObserveHost)
	if opts.AppHost == "" {
		return nil, fmt.Errorf("app host is required")
	}
	if opts.OpenObserveRootEmail == "" {
		return nil, fmt.Errorf("openobserve root email is required")
	}
	if strings.TrimSpace(opts.OpenObserveRootPassword) == "" {
		return nil, fmt.Errorf("openobserve root password is required")
	}

	if opts.AuthSecret == "" {
		opts.AuthSecret = randomSecret(48)
	}
	if opts.WebhookJWTSecret == "" {
		opts.WebhookJWTSecret = randomSecret(48)
	}
	if opts.PostgresPassword == "" {
		opts.PostgresPassword = randomSecret(32)
	}
	if opts.EtcdRootPassword == "" {
		opts.EtcdRootPassword = randomSecret(24)
	}
	if opts.OpensearchAdminPassword == "" {
		opts.OpensearchAdminPassword = randomStrongPassword(24)
	}
	if opts.BackendTag == "" {
		resolvedTag, err := e.resolveLatestStableTag(ctx, e.backendRepo)
		if err != nil {
			return nil, err
		}
		opts.BackendTag = resolvedTag
	}
	if opts.FrontendTag == "" {
		resolvedTag, err := e.resolveLatestStableTag(ctx, e.frontendRepo)
		if err != nil {
			return nil, err
		}
		opts.FrontendTag = resolvedTag
	}

	switch opts.StorageMode {
	case StorageMinIO:
		if opts.MinIORootUser == "" {
			opts.MinIORootUser = "gochat"
		}
		if opts.MinIORootPassword == "" {
			opts.MinIORootPassword = randomSecret(32)
		}
	case StorageExternal:
		if opts.ExternalS3Endpoint == "" {
			return nil, fmt.Errorf("external s3 endpoint is required for external storage mode")
		}
		if opts.ExternalS3AccessKeyID == "" {
			return nil, fmt.Errorf("external s3 access key id is required for external storage mode")
		}
		if opts.ExternalS3SecretAccessKey == "" {
			return nil, fmt.Errorf("external s3 secret access key is required for external storage mode")
		}
	}
	switch opts.EmailProvider {
	case "log":
	case "smtp":
		if strings.TrimSpace(opts.SMTPHost) == "" {
			return nil, fmt.Errorf("smtp host is required when email provider is smtp")
		}
	case "sendpulse":
		if strings.TrimSpace(opts.SendpulseUserID) == "" {
			return nil, fmt.Errorf("sendpulse user id is required when email provider is sendpulse")
		}
		if strings.TrimSpace(opts.SendpulseSecret) == "" {
			return nil, fmt.Errorf("sendpulse secret is required when email provider is sendpulse")
		}
	case "resend":
		if strings.TrimSpace(opts.ResendAPIKey) == "" {
			return nil, fmt.Errorf("resend api key is required when email provider is resend")
		}
	case "dashamail":
		if strings.TrimSpace(opts.DashaMailAPIKey) == "" {
			return nil, fmt.Errorf("dashamail api key is required when email provider is dashamail")
		}
	default:
		return nil, fmt.Errorf("email provider must be one of: log, smtp, sendpulse, resend, dashamail")
	}

	prepared := &preparedOptions{Options: opts}
	switch opts.DeploymentType {
	case DeploymentCompose:
		prepared.useTLS = false
		prepared.useBundledTraefik = true
	case DeploymentHelm:
		prepared.useTLS = toggleValue(opts.UseTLS, true)
		defaultTraefik := true
		if strings.TrimSpace(opts.IngressClassName) != "" {
			defaultTraefik = false
		}
		prepared.useBundledTraefik = toggleValue(opts.UseBundledTraefik, defaultTraefik)
	}

	if prepared.useTLS {
		prepared.httpScheme = "https"
		prepared.wsScheme = "wss"
	} else {
		prepared.httpScheme = "http"
		prepared.wsScheme = "ws"
	}

	prepared.appPublicURL = fmt.Sprintf("%s://%s", prepared.httpScheme, prepared.AppHost)
	prepared.apiPublicBaseURL = fmt.Sprintf("%s://%s/api/v1", prepared.httpScheme, prepared.AppHost)
	wsPublicPath := "/ws/subscribe"
	if prepared.DeploymentType == DeploymentHelm {
		wsPublicPath = "/ws"
	}
	prepared.wsPublicURL = fmt.Sprintf("%s://%s%s", prepared.wsScheme, prepared.AppHost, wsPublicPath)

	if prepared.StorageMode == StorageMinIO {
		prepared.storageEndpointURL = fmt.Sprintf("%s://%s", prepared.httpScheme, prepared.StorageHost)
		prepared.storagePublicBaseURL = strings.TrimRight(prepared.storageEndpointURL, "/") + "/" + prepared.StorageBucket
		prepared.storageAccessKey = prepared.MinIORootUser
		prepared.storageSecretKey = prepared.MinIORootPassword
		prepared.storageRegion = ""
		prepared.storageUseSSL = prepared.useTLS
		prepared.storageAlias = prepared.StorageHost
		prepared.minioConsoleAlias = prepared.MinIOConsoleHost
	} else {
		defaultScheme := "http"
		if prepared.ExternalS3UseSSL {
			defaultScheme = "https"
		}
		prepared.storageEndpointURL = normalizeURL(prepared.ExternalS3Endpoint, defaultScheme)
		if prepared.ExternalS3PublicBaseURL != "" {
			prepared.storagePublicBaseURL = normalizeURL(prepared.ExternalS3PublicBaseURL, defaultScheme)
		} else {
			prepared.storagePublicBaseURL = strings.TrimRight(prepared.storageEndpointURL, "/") + "/" + prepared.StorageBucket
		}
		prepared.storageAccessKey = prepared.ExternalS3AccessKeyID
		prepared.storageSecretKey = prepared.ExternalS3SecretAccessKey
		prepared.storageRegion = prepared.ExternalS3Region
		prepared.storageUseSSL = strings.HasPrefix(strings.ToLower(prepared.storageEndpointURL), "https://") || prepared.ExternalS3UseSSL
		prepared.storageAlias = "storage.alias.internal.invalid"
		prepared.minioConsoleAlias = "minio.alias.internal.invalid"
	}

	prepared.contentHosts = uniqueStrings([]string{
		originFromURL(prepared.storagePublicBaseURL),
		originFromURL(prepared.apiPublicBaseURL),
	})
	prepared.helmFullName = helmFullName(prepared.ReleaseName)
	prepared.backendTag = prepared.BackendTag
	prepared.frontendTag = prepared.FrontendTag
	prepared.imageAPI = imageRef(prepared.ImageRepositoryPrefix, "api", prepared.backendTag)
	prepared.imageAuth = imageRef(prepared.ImageRepositoryPrefix, "auth", prepared.backendTag)
	prepared.imageAttachments = imageRef(prepared.ImageRepositoryPrefix, "attachments", prepared.backendTag)
	prepared.imageWS = imageRef(prepared.ImageRepositoryPrefix, "ws", prepared.backendTag)
	prepared.imageWebhook = imageRef(prepared.ImageRepositoryPrefix, "webhook", prepared.backendTag)
	prepared.imageIndexer = imageRef(prepared.ImageRepositoryPrefix, "indexer", prepared.backendTag)
	prepared.imageEmbedder = imageRef(prepared.ImageRepositoryPrefix, "embedder", prepared.backendTag)
	prepared.imageUI = uiImageRef(prepared.ImageRepositoryPrefix, prepared.frontendTag)
	prepared.imageMigrations = fmt.Sprintf("%s:%s", strings.TrimRight(prepared.MigrationsImageRepo, ":"), prepared.MigrationsImageTag)
	prepared.composePGAddr = fmt.Sprintf("postgres://postgres:%s@citus-master:5432/gochat?sslmode=disable", prepared.PostgresPassword)
	prepared.composeCassandraAddr = "cassandra://scylla:9042/gochat?x-multi-statement=true"
	prepared.composePGDSN = fmt.Sprintf("host=citus-master port=5432 user=postgres password=%s dbname=gochat sslmode=disable", prepared.PostgresPassword)
	prepared.openObserveRootEmail = prepared.OpenObserveRootEmail
	prepared.openObserveRootPassword = prepared.OpenObserveRootPassword
	prepared.openObserveOrg = "default"
	prepared.openObserveBasicAuth = base64.StdEncoding.EncodeToString([]byte(prepared.openObserveRootEmail + ":" + prepared.openObserveRootPassword))
	prepared.openObserveLogStream = "gochat_logs"
	prepared.openObserveMetricStream = "gochat-metrics"
	prepared.openObserveTraceStream = "gochat_traces"
	sfuServiceID, err := newUUIDv4()
	if err != nil {
		return nil, err
	}
	prepared.sfuServiceID = sfuServiceID
	sfuWebhookToken, err := generateServiceToken(prepared.WebhookJWTSecret, sfuServiceType, prepared.sfuServiceID)
	if err != nil {
		return nil, err
	}
	prepared.sfuWebhookToken = sfuWebhookToken

	return prepared, nil
}

func runCommand(ctx context.Context, dir string, extraEnv []string, output io.Writer, name string, args ...string) error {
	if output == nil {
		output = io.Discard
	}

	fmt.Fprintf(output, "[gochat] Running: %s %s\n", name, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}

func helmCommandEnv(workspaceRoot string) []string {
	if os.Getenv("DOCKER_CONFIG") != "" {
		return nil
	}

	dockerConfigRoot := filepath.Join(workspaceRoot, ".generated", "docker-config")
	if err := os.MkdirAll(dockerConfigRoot, 0o755); err != nil {
		return nil
	}
	return []string{"DOCKER_CONFIG=" + dockerConfigRoot}
}

func composeDeployCommand(prepared *preparedOptions, result RenderResult) string {
	args := []string{
		"docker", "compose",
		"--env-file", quoteCommandArg(result.ComposeEnvPath),
		"-f", quoteCommandArg(result.ComposeFilePath),
	}
	if prepared.StorageMode == StorageMinIO {
		args = append(args, "--profile", "minio")
	}
	args = append(args, "up", "-d")
	return strings.Join(args, " ")
}

func helmDeployCommand(prepared *preparedOptions, result RenderResult) string {
	args := []string{
		"helm", "upgrade", "--install",
		prepared.ReleaseName,
		quoteCommandArg(result.HelmChartPath),
		"--namespace", prepared.Namespace,
		"--create-namespace",
		"--values", quoteCommandArg(result.HelmValuesPath),
	}
	return strings.Join(args, " ")
}

func quoteCommandArg(value string) string {
	if value == "" {
		return `""`
	}
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func isBundledPath(path string) bool {
	return path == "compose" ||
		path == "helm" ||
		path == "scripts" ||
		strings.HasPrefix(path, "compose/") ||
		strings.HasPrefix(path, "helm/") ||
		strings.HasPrefix(path, "scripts/")
}

func toggleValue(mode ToggleMode, defaultValue bool) bool {
	switch mode {
	case ToggleOn:
		return true
	case ToggleOff:
		return false
	default:
		return defaultValue
	}
}

func normalizeHost(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "://") {
		if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
			value = parsed.Host
		}
	}
	return strings.Trim(strings.TrimSpace(value), "/")
}

func normalizeURL(value, defaultScheme string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "://") {
		return strings.TrimRight(value, "/")
	}
	return defaultScheme + "://" + strings.Trim(value, "/")
}

func originFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return parsed.Scheme + "://" + parsed.Host
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func randomSecret(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if length <= 0 {
		return ""
	}

	var builder strings.Builder
	builder.Grow(length)
	max := big.NewInt(int64(len(chars)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			builder.WriteByte(chars[i%len(chars)])
			continue
		}
		builder.WriteByte(chars[n.Int64()])
	}

	return builder.String()
}

func randomStrongPassword(length int) string {
	const (
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitChars   = "0123456789"
		specialChars = "!@#$%^&*_-+=?"
	)
	if length < 4 {
		length = 4
	}

	allChars := lowerChars + upperChars + digitChars + specialChars
	password := []byte{
		randomCharsetByte(lowerChars),
		randomCharsetByte(upperChars),
		randomCharsetByte(digitChars),
		randomCharsetByte(specialChars),
	}
	for len(password) < length {
		password = append(password, randomCharsetByte(allChars))
	}

	for i := len(password) - 1; i > 0; i-- {
		j := randomInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

func randomCharsetByte(chars string) byte {
	return chars[randomInt(len(chars))]
}

func randomInt(max int) int {
	if max <= 1 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func imageRef(prefix, name, tag string) string {
	prefix = strings.TrimRight(prefix, "/")
	if prefix == "" {
		return fmt.Sprintf("gochat-%s:%s", name, tag)
	}
	return fmt.Sprintf("%s/gochat-%s:%s", prefix, name, tag)
}

func uiImageRef(prefix, tag string) string {
	prefix = strings.TrimRight(prefix, "/")
	if prefix == "" {
		return "gochat-react:" + tag
	}
	return prefix + "/gochat-react:" + tag
}

func helmFullName(release string) string {
	if strings.Contains(release, "gochat") {
		return release
	}
	return release + "-gochat"
}

func imageRepository(imageRef string) string {
	index := strings.LastIndex(imageRef, ":")
	if index == -1 {
		return imageRef
	}
	return imageRef[:index]
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func indent(text string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
