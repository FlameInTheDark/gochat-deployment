package deployer

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
)

func runWizard(ctx context.Context, engine *Engine, seed Options) error {
	state := newWizardState(seed)
	smtpPort := strconv.Itoa(state.SMTPPort)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[ActionMode]().
				Title("Action").
				Options(
					huh.NewOption("Deploy", ActionDeploy),
					huh.NewOption("Render Only", ActionRender),
				).
				Value(&state.Action),
			huh.NewSelect[DeploymentType]().
				Title("Deployment Type").
				Options(
					huh.NewOption("Docker Compose", DeploymentCompose),
					huh.NewOption("Helm", DeploymentHelm),
				).
				Value(&state.DeploymentType),
			huh.NewSelect[StorageMode]().
				Title("Storage Mode").
				Options(
					huh.NewOption("Bundled MinIO", StorageMinIO),
					huh.NewOption("External S3", StorageExternal),
				).
				Value(&state.StorageMode),
		).Title("Target"),
		huh.NewGroup(
			huh.NewInput().
				Title("Base Domain").
				Description("UI lives at the root domain. API uses /api/v1 and WS uses /ws.").
				Value(&state.BaseDomain).
				Validate(requiredText),
			huh.NewInput().
				Title("App Host Override").
				Description("Leave blank to use the base domain directly.").
				Value(&state.AppHost),
			huh.NewInput().
				Title("Telemetry Host Override").
				Description("Leave blank for telemetry.<domain>.").
				Value(&state.TelemetryHost),
			huh.NewInput().
				Title("Storage Host Override").
				Description("Leave blank for storage.<domain>.").
				Value(&state.StorageHost),
			huh.NewInput().
				Title("MinIO Console Host Override").
				Description("Leave blank for minio.<domain>.").
				Value(&state.MinIOConsoleHost),
			huh.NewInput().
				Title("Workspace Root").
				Value(&state.WorkspaceRoot).
				Validate(requiredText),
		).Title("Hosts"),
		huh.NewGroup(
			huh.NewInput().
				Title("Namespace").
				Value(&state.Namespace).
				Validate(requiredText),
			huh.NewInput().
				Title("Release Name").
				Value(&state.ReleaseName).
				Validate(requiredText),
			huh.NewSelect[ToggleMode]().
				Title("TLS Mode").
				Options(
					huh.NewOption("Auto", ToggleAuto),
					huh.NewOption("On", ToggleOn),
					huh.NewOption("Off", ToggleOff),
				).
				Value(&state.TLSMode),
			huh.NewSelect[ToggleMode]().
				Title("Traefik Mode").
				Options(
					huh.NewOption("Auto", ToggleAuto),
					huh.NewOption("On", ToggleOn),
					huh.NewOption("Off", ToggleOff),
				).
				Value(&state.TraefikMode),
			huh.NewInput().
				Title("Ingress Class").
				Value(&state.IngressClassName),
			huh.NewInput().
				Title("TLS Secret").
				Value(&state.TLSSecretName),
		).Title("Helm").
			WithHideFunc(func() bool { return state.DeploymentType != DeploymentHelm }),
		huh.NewGroup(
			huh.NewInput().
				Title("OpenObserve Host").
				Description("Leave blank to keep the OpenObserve panel internal-only. Only used for Helm.").
				Value(&state.OpenObserveHost),
		).Title("OpenObserve Host").
			WithHideFunc(func() bool { return state.DeploymentType != DeploymentHelm }),
		huh.NewGroup(
			huh.NewInput().
				Title("OpenObserve Email").
				Description("Enter the admin email explicitly. No default is injected here.").
				Value(&state.OpenObserveRootEmail).
				Validate(requiredText),
			huh.NewInput().
				Title("OpenObserve Password").
				Description("Enter the admin password explicitly.").
				Value(&state.OpenObserveRootPass).
				Validate(requiredText).
				EchoMode(huh.EchoModePassword),
		).Title("OpenObserve Credentials"),
		huh.NewGroup(
			huh.NewInput().
				Title("Image Repository Prefix").
				Value(&state.ImageRepositoryPrefix),
			huh.NewInput().
				Title("Backend Tag Override").
				Description("Leave blank to resolve the latest stable backend tag automatically.").
				Value(&state.BackendTag),
			huh.NewInput().
				Title("Frontend Tag Override").
				Description("Leave blank to resolve the latest stable frontend tag automatically.").
				Value(&state.FrontendTag),
			huh.NewInput().
				Title("Migrations Image Repository").
				Description("Leave blank to use <image-repository-prefix>/gochat-migrations.").
				Value(&state.MigrationsImageRepo),
			huh.NewInput().
				Title("Migrations Image Tag").
				Description("Leave blank to match the resolved backend tag.").
				Value(&state.MigrationsImageTag),
		).Title("Images"),
		huh.NewGroup(
			huh.NewInput().
				Title("Bucket").
				Value(&state.StorageBucket).
				Validate(requiredText),
			huh.NewInput().
				Title("MinIO Root User").
				Value(&state.MinIORootUser),
			huh.NewInput().
				Title("MinIO Root Password").
				Value(&state.MinIORootPassword).
				EchoMode(huh.EchoModePassword),
		).Title("MinIO").
			WithHideFunc(func() bool { return state.StorageMode != StorageMinIO }),
		huh.NewGroup(
			huh.NewInput().
				Title("External S3 Endpoint").
				Value(&state.ExternalEndpoint).
				Validate(requiredText),
			huh.NewInput().
				Title("External S3 Public Base URL").
				Value(&state.ExternalPublicURL),
			huh.NewInput().
				Title("External S3 Access Key").
				Value(&state.ExternalAccessKey).
				Validate(requiredText),
			huh.NewInput().
				Title("External S3 Secret Key").
				Value(&state.ExternalSecretKey).
				Validate(requiredText).
				EchoMode(huh.EchoModePassword),
			huh.NewInput().
				Title("External S3 Region").
				Value(&state.ExternalRegion).
				Validate(requiredText),
			huh.NewConfirm().
				Title("External S3 Uses SSL").
				Value(&state.ExternalUseSSL),
		).Title("External S3").
			WithHideFunc(func() bool { return state.StorageMode != StorageExternal }),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Email Provider").
				Options(
					huh.NewOption("Log", "log"),
					huh.NewOption("SMTP", "smtp"),
					huh.NewOption("SendPulse", "sendpulse"),
					huh.NewOption("Resend", "resend"),
					huh.NewOption("DashaMail", "dashamail"),
				).
				Value(&state.EmailProvider),
			huh.NewInput().
				Title("Email Source").
				Value(&state.EmailSource).
				Validate(requiredText),
			huh.NewInput().
				Title("Email Name").
				Value(&state.EmailName).
				Validate(requiredText),
		).Title("Email"),
		huh.NewGroup(
			huh.NewInput().
				Title("SMTP Host").
				Value(&state.SMTPHost).
				Validate(requiredWhen(func() bool { return state.EmailProvider == "smtp" })),
			huh.NewInput().
				Title("SMTP Port").
				Value(&smtpPort).
				Validate(validatePortWhen(func() bool { return state.EmailProvider == "smtp" })),
			huh.NewInput().
				Title("SMTP Username").
				Value(&state.SMTPUsername),
			huh.NewInput().
				Title("SMTP Password").
				Value(&state.SMTPPassword).
				EchoMode(huh.EchoModePassword),
			huh.NewConfirm().
				Title("SMTP Uses TLS").
				Value(&state.SMTPUseTLS),
		).Title("SMTP").
			WithHideFunc(func() bool { return state.EmailProvider != "smtp" }),
		huh.NewGroup(
			huh.NewInput().
				Title("SendPulse User ID").
				Value(&state.SendpulseUserID).
				Validate(requiredWhen(func() bool { return state.EmailProvider == "sendpulse" })),
			huh.NewInput().
				Title("SendPulse Secret").
				Value(&state.SendpulseSecret).
				Validate(requiredWhen(func() bool { return state.EmailProvider == "sendpulse" })).
				EchoMode(huh.EchoModePassword),
		).Title("SendPulse").
			WithHideFunc(func() bool { return state.EmailProvider != "sendpulse" }),
		huh.NewGroup(
			huh.NewInput().
				Title("Resend API Key").
				Value(&state.ResendAPIKey).
				Validate(requiredWhen(func() bool { return state.EmailProvider == "resend" })).
				EchoMode(huh.EchoModePassword),
		).Title("Resend").
			WithHideFunc(func() bool { return state.EmailProvider != "resend" }),
		huh.NewGroup(
			huh.NewInput().
				Title("DashaMail API Key").
				Value(&state.DashaMailAPIKey).
				Validate(requiredWhen(func() bool { return state.EmailProvider == "dashamail" })).
				EchoMode(huh.EchoModePassword),
		).Title("DashaMail").
			WithHideFunc(func() bool { return state.EmailProvider != "dashamail" }),
	).WithTheme(huh.ThemeCharm()).
		WithAccessible(accessibleWizardMode()).
		WithShowHelp(true)

	if err := form.RunWithContext(ctx); err != nil {
		return err
	}
	if state.EmailProvider == "smtp" {
		parsedPort, err := strconv.Atoi(strings.TrimSpace(smtpPort))
		if err != nil {
			return fmt.Errorf("invalid SMTP port: %w", err)
		}
		state.SMTPPort = parsedPort
	}

	opts := state.options()
	prepared, err := engine.prepareOptions(ctx, opts)
	if err != nil {
		return err
	}
	opts = prepared.Options

	report := engine.Check(ctx, opts)
	if missing := report.MissingRequired(); len(missing) > 0 {
		var details []string
		for _, item := range missing {
			details = append(details, fmt.Sprintf("%s: %s", item.Name, item.Detail))
		}
		return fmt.Errorf("missing required tools:\n%s", strings.Join(details, "\n"))
	}

	var confirmed bool
	review := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Review").
				Description(reviewText(prepared, report)),
			huh.NewConfirm().
				Title(confirmTitle(state.Action)).
				Affirmative("Run").
				Negative("Cancel").
				Value(&confirmed),
		).Title("Confirmation"),
	).WithTheme(huh.ThemeCharm()).
		WithAccessible(accessibleWizardMode()).
		WithShowHelp(true)

	if err := review.RunWithContext(ctx); err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("wizard cancelled")
	}

	fmt.Fprintln(os.Stdout)
	var result RenderResult
	if state.Action == ActionRender {
		result, err = engine.Render(ctx, opts, os.Stdout)
	} else {
		result, err = engine.Deploy(ctx, opts, os.Stdout)
	}
	if err != nil {
		return err
	}

	printSummaryTo(os.Stdout, result, state.Action == ActionRender)
	return nil
}

func reviewText(prepared *preparedOptions, report CheckReport) string {
	lines := []string{
		fmt.Sprintf("Action: %s", prepared.Action),
		fmt.Sprintf("Type: %s", prepared.DeploymentType),
		fmt.Sprintf("Storage: %s", prepared.StorageMode),
		fmt.Sprintf("Workspace: %s", prepared.WorkspaceRoot),
		fmt.Sprintf("Backend Tag: %s", prepared.backendTag),
		fmt.Sprintf("Frontend Tag: %s", prepared.frontendTag),
		fmt.Sprintf("Migrations Tag: %s", prepared.MigrationsImageTag),
		fmt.Sprintf("UI: %s", prepared.appPublicURL),
		fmt.Sprintf("API: %s", prepared.apiPublicBaseURL),
		fmt.Sprintf("WebSocket: %s", prepared.wsPublicURL),
		fmt.Sprintf("Telemetry: %s", prepared.telemetryPublicURL),
		fmt.Sprintf("Email Provider: %s", prepared.EmailProvider),
	}
	if prepared.storagePublicBaseURL != "" {
		lines = append(lines, fmt.Sprintf("Storage URL: %s", prepared.storagePublicBaseURL))
	}
	lines = append(lines, "", "Tools:")
	for _, item := range report.Items {
		status := "missing"
		if item.Present {
			status = "ok"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", item.Name, status))
	}
	return strings.Join(lines, "\n")
}

func confirmTitle(action ActionMode) string {
	if action == ActionRender {
		return "Render configuration now?"
	}
	return "Deploy now?"
}

func accessibleWizardMode() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GOCHAT_DEPLOYER_ACCESSIBLE"))) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	}
	return false
}

func requiredText(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("this field is required")
	}
	return nil
}

func requiredWhen(enabled func() bool) func(string) error {
	return func(value string) error {
		if enabled == nil || !enabled() {
			return nil
		}
		return requiredText(value)
	}
}

func validatePort(value string) error {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port <= 0 || port > 65535 {
		return fmt.Errorf("enter a valid port")
	}
	return nil
}

func validatePortWhen(enabled func() bool) func(string) error {
	return func(value string) error {
		if enabled == nil || !enabled() {
			return nil
		}
		return validatePort(value)
	}
}
