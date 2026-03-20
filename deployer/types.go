package deployer

import "fmt"

type DeploymentType string

const (
	DeploymentCompose DeploymentType = "compose"
	DeploymentHelm    DeploymentType = "helm"
)

func (d DeploymentType) Valid() bool {
	return d == DeploymentCompose || d == DeploymentHelm
}

type StorageMode string

const (
	StorageMinIO    StorageMode = "minio"
	StorageExternal StorageMode = "external"
)

func (s StorageMode) Valid() bool {
	return s == StorageMinIO || s == StorageExternal
}

type ToggleMode string

const (
	ToggleAuto ToggleMode = "auto"
	ToggleOn   ToggleMode = "on"
	ToggleOff  ToggleMode = "off"
)

func (m ToggleMode) Valid() bool {
	return m == ToggleAuto || m == ToggleOn || m == ToggleOff
}

type ActionMode string

const (
	ActionDeploy ActionMode = "deploy"
	ActionRender ActionMode = "render"
)

func (a ActionMode) Valid() bool {
	return a == ActionDeploy || a == ActionRender
}

type Options struct {
	DeploymentType DeploymentType
	StorageMode    StorageMode
	Action         ActionMode

	BaseDomain       string
	AppHost          string
	APIHost          string
	WSHost           string
	StorageHost      string
	MinIOConsoleHost string
	Namespace        string
	ReleaseName      string
	WorkspaceRoot    string
	IngressClassName string
	TLSSecretName    string
	OpenObserveHost  string

	ImageRepositoryPrefix string
	BackendTag            string
	FrontendTag           string
	MigrationsImageRepo   string
	MigrationsImageTag    string
	UseTLS                ToggleMode
	UseBundledTraefik     ToggleMode

	AuthSecret              string
	WebhookJWTSecret        string
	PostgresPassword        string
	EtcdRootPassword        string
	OpensearchAdminPassword string
	OpenObserveRootEmail    string
	OpenObserveRootPassword string

	StorageBucket             string
	MinIORootUser             string
	MinIORootPassword         string
	ExternalS3Endpoint        string
	ExternalS3PublicBaseURL   string
	ExternalS3AccessKeyID     string
	ExternalS3SecretAccessKey string
	ExternalS3Region          string
	ExternalS3UseSSL          bool

	EmailProvider   string
	EmailSource     string
	EmailName       string
	SendpulseUserID string
	SendpulseSecret string
	ResendAPIKey    string
	DashaMailAPIKey string
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPUseTLS      bool
}

type ToolStatus struct {
	Name     string
	Required bool
	Present  bool
	Detail   string
}

type CheckReport struct {
	Items []ToolStatus
}

func (r CheckReport) MissingRequired() []ToolStatus {
	missing := make([]ToolStatus, 0)
	for _, item := range r.Items {
		if item.Required && !item.Present {
			missing = append(missing, item)
		}
	}
	return missing
}

type RenderResult struct {
	WorkspaceRoot        string
	GeneratedRoot        string
	ComposeEnvPath       string
	ComposeConfigRoot    string
	ComposeFilePath      string
	HelmChartPath        string
	HelmValuesPath       string
	ComposeDeployCommand string
	HelmDeployCommand    string
	InstructionsPath     string
	BackendTag           string
	FrontendTag          string
	MigrationsTag        string
	AppPublicURL         string
	APIPublicBaseURL     string
	WSPublicURL          string
	StoragePublicURL     string
	MinIOConsoleURL      string
	OpenObserveURL       string
}

func (r RenderResult) SummaryLines() []string {
	lines := []string{
		fmt.Sprintf("Workspace: %s", r.WorkspaceRoot),
		fmt.Sprintf("Backend:   %s", r.BackendTag),
		fmt.Sprintf("Frontend:  %s", r.FrontendTag),
		fmt.Sprintf("Migrate:   %s", r.MigrationsTag),
		fmt.Sprintf("UI:        %s", r.AppPublicURL),
		fmt.Sprintf("API:       %s", r.APIPublicBaseURL),
		fmt.Sprintf("WebSocket: %s", r.WSPublicURL),
	}
	if r.StoragePublicURL != "" {
		lines = append(lines, fmt.Sprintf("Storage:   %s", r.StoragePublicURL))
	}
	if r.MinIOConsoleURL != "" {
		lines = append(lines, fmt.Sprintf("Console:   %s", r.MinIOConsoleURL))
	}
	if r.OpenObserveURL != "" {
		lines = append(lines, fmt.Sprintf("Observe:   %s", r.OpenObserveURL))
	}
	if r.InstructionsPath != "" {
		lines = append(lines, fmt.Sprintf("Guide:     %s", r.InstructionsPath))
	}
	return lines
}
