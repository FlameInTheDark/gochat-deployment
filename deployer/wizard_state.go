package deployer

type wizardState struct {
	Action                ActionMode
	DeploymentType        DeploymentType
	StorageMode           StorageMode
	BaseDomain            string
	AppHost               string
	TelemetryHost         string
	StorageHost           string
	MinIOConsoleHost      string
	Namespace             string
	ReleaseName           string
	WorkspaceRoot         string
	OpenObserveHost       string
	ImageRepositoryPrefix string
	BackendTag            string
	FrontendTag           string
	MigrationsImageRepo   string
	MigrationsImageTag    string
	TLSMode               ToggleMode
	TraefikMode           ToggleMode
	IngressClassName      string
	TLSSecretName         string
	OpenObserveRootEmail  string
	OpenObserveRootPass   string
	MFAEncryptionKey      string
	StorageBucket         string
	MinIORootUser         string
	MinIORootPassword     string
	ExternalEndpoint      string
	ExternalPublicURL     string
	ExternalAccessKey     string
	ExternalSecretKey     string
	ExternalRegion        string
	ExternalUseSSL        bool
	EmailProvider         string
	EmailSource           string
	EmailName             string
	SendpulseUserID       string
	SendpulseSecret       string
	ResendAPIKey          string
	DashaMailAPIKey       string
	SMTPHost              string
	SMTPPort              int
	SMTPUsername          string
	SMTPPassword          string
	SMTPUseTLS            bool
}

func newWizardState(seed Options) wizardState {
	state := wizardState{
		Action:                ActionDeploy,
		DeploymentType:        seed.DeploymentType,
		StorageMode:           seed.StorageMode,
		BaseDomain:            seed.BaseDomain,
		AppHost:               seed.AppHost,
		TelemetryHost:         seed.TelemetryHost,
		StorageHost:           seed.StorageHost,
		MinIOConsoleHost:      seed.MinIOConsoleHost,
		Namespace:             seed.Namespace,
		ReleaseName:           seed.ReleaseName,
		WorkspaceRoot:         seed.WorkspaceRoot,
		OpenObserveHost:       seed.OpenObserveHost,
		ImageRepositoryPrefix: seed.ImageRepositoryPrefix,
		BackendTag:            seed.BackendTag,
		FrontendTag:           seed.FrontendTag,
		MigrationsImageRepo:   seed.MigrationsImageRepo,
		MigrationsImageTag:    seed.MigrationsImageTag,
		TLSMode:               seed.UseTLS,
		TraefikMode:           seed.UseBundledTraefik,
		IngressClassName:      seed.IngressClassName,
		TLSSecretName:         seed.TLSSecretName,
		OpenObserveRootEmail:  seed.OpenObserveRootEmail,
		OpenObserveRootPass:   seed.OpenObserveRootPassword,
		MFAEncryptionKey:      seed.MFAEncryptionKey,
		StorageBucket:         seed.StorageBucket,
		MinIORootUser:         seed.MinIORootUser,
		MinIORootPassword:     seed.MinIORootPassword,
		ExternalEndpoint:      seed.ExternalS3Endpoint,
		ExternalPublicURL:     seed.ExternalS3PublicBaseURL,
		ExternalAccessKey:     seed.ExternalS3AccessKeyID,
		ExternalSecretKey:     seed.ExternalS3SecretAccessKey,
		ExternalRegion:        seed.ExternalS3Region,
		ExternalUseSSL:        seed.ExternalS3UseSSL,
		EmailProvider:         seed.EmailProvider,
		EmailSource:           seed.EmailSource,
		EmailName:             seed.EmailName,
		SendpulseUserID:       seed.SendpulseUserID,
		SendpulseSecret:       seed.SendpulseSecret,
		ResendAPIKey:          seed.ResendAPIKey,
		DashaMailAPIKey:       seed.DashaMailAPIKey,
		SMTPHost:              seed.SMTPHost,
		SMTPPort:              seed.SMTPPort,
		SMTPUsername:          seed.SMTPUsername,
		SMTPPassword:          seed.SMTPPassword,
		SMTPUseTLS:            seed.SMTPUseTLS,
	}

	if !state.DeploymentType.Valid() {
		state.DeploymentType = DeploymentCompose
	}
	if !state.StorageMode.Valid() {
		state.StorageMode = StorageMinIO
	}
	if !state.TLSMode.Valid() {
		state.TLSMode = ToggleAuto
	}
	if !state.TraefikMode.Valid() {
		state.TraefikMode = ToggleAuto
	}
	if state.Namespace == "" {
		state.Namespace = "gochat"
	}
	if state.ReleaseName == "" {
		state.ReleaseName = "gochat"
	}
	if state.WorkspaceRoot == "" {
		state.WorkspaceRoot = ".gochat-deployer/workspace"
	}
	if state.ImageRepositoryPrefix == "" {
		state.ImageRepositoryPrefix = "ghcr.io/flameinthedark"
	}
	if state.StorageBucket == "" {
		state.StorageBucket = "gochat"
	}
	if state.ExternalRegion == "" {
		state.ExternalRegion = "us-east-1"
	}
	if state.EmailProvider == "" {
		state.EmailProvider = "log"
	}
	if state.EmailSource == "" {
		state.EmailSource = "no-reply@example.com"
	}
	if state.EmailName == "" {
		state.EmailName = "GoChat"
	}
	if state.SMTPPort == 0 {
		state.SMTPPort = 2525
	}

	return state
}

func (s wizardState) options() Options {
	return Options{
		Action:                    s.Action,
		DeploymentType:            s.DeploymentType,
		StorageMode:               s.StorageMode,
		BaseDomain:                s.BaseDomain,
		AppHost:                   s.AppHost,
		TelemetryHost:             s.TelemetryHost,
		StorageHost:               s.StorageHost,
		MinIOConsoleHost:          s.MinIOConsoleHost,
		Namespace:                 s.Namespace,
		ReleaseName:               s.ReleaseName,
		WorkspaceRoot:             s.WorkspaceRoot,
		OpenObserveHost:           s.OpenObserveHost,
		ImageRepositoryPrefix:     s.ImageRepositoryPrefix,
		BackendTag:                s.BackendTag,
		FrontendTag:               s.FrontendTag,
		MigrationsImageRepo:       s.MigrationsImageRepo,
		MigrationsImageTag:        s.MigrationsImageTag,
		UseTLS:                    s.TLSMode,
		UseBundledTraefik:         s.TraefikMode,
		IngressClassName:          s.IngressClassName,
		TLSSecretName:             s.TLSSecretName,
		OpenObserveRootEmail:      s.OpenObserveRootEmail,
		OpenObserveRootPassword:   s.OpenObserveRootPass,
		MFAEncryptionKey:          s.MFAEncryptionKey,
		StorageBucket:             s.StorageBucket,
		MinIORootUser:             s.MinIORootUser,
		MinIORootPassword:         s.MinIORootPassword,
		ExternalS3Endpoint:        s.ExternalEndpoint,
		ExternalS3PublicBaseURL:   s.ExternalPublicURL,
		ExternalS3AccessKeyID:     s.ExternalAccessKey,
		ExternalS3SecretAccessKey: s.ExternalSecretKey,
		ExternalS3Region:          s.ExternalRegion,
		ExternalS3UseSSL:          s.ExternalUseSSL,
		EmailProvider:             s.EmailProvider,
		EmailSource:               s.EmailSource,
		EmailName:                 s.EmailName,
		SendpulseUserID:           s.SendpulseUserID,
		SendpulseSecret:           s.SendpulseSecret,
		ResendAPIKey:              s.ResendAPIKey,
		DashaMailAPIKey:           s.DashaMailAPIKey,
		SMTPHost:                  s.SMTPHost,
		SMTPPort:                  s.SMTPPort,
		SMTPUsername:              s.SMTPUsername,
		SMTPPassword:              s.SMTPPassword,
		SMTPUseTLS:                s.SMTPUseTLS,
	}
}
