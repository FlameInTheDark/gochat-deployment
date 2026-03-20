package deployer

import (
	"context"
	"strings"
	"testing"
)

func TestPrepareOptionsComposeMinIOUsesRootDomainRouting(t *testing.T) {
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

	if prepared.AppHost != "example.com" {
		t.Fatalf("AppHost = %q, want example.com", prepared.AppHost)
	}
	if prepared.APIHost != "example.com" {
		t.Fatalf("APIHost = %q, want example.com", prepared.APIHost)
	}
	if prepared.WSHost != "example.com" {
		t.Fatalf("WSHost = %q, want example.com", prepared.WSHost)
	}
	if prepared.appPublicURL != "http://example.com" {
		t.Fatalf("appPublicURL = %q, want http://example.com", prepared.appPublicURL)
	}
	if prepared.apiPublicBaseURL != "http://example.com/api/v1" {
		t.Fatalf("apiPublicBaseURL = %q, want http://example.com/api/v1", prepared.apiPublicBaseURL)
	}
	if prepared.wsPublicURL != "ws://example.com/ws/subscribe" {
		t.Fatalf("wsPublicURL = %q, want ws://example.com/ws/subscribe", prepared.wsPublicURL)
	}
	if prepared.StorageHost != "storage.example.com" {
		t.Fatalf("StorageHost = %q, want storage.example.com", prepared.StorageHost)
	}
	if prepared.MinIOConsoleHost != "minio.example.com" {
		t.Fatalf("MinIOConsoleHost = %q, want minio.example.com", prepared.MinIOConsoleHost)
	}
	if prepared.useTLS {
		t.Fatal("compose deployment should not enable TLS-aware URLs")
	}
	if !prepared.useBundledTraefik {
		t.Fatal("compose deployment should force bundled Traefik")
	}
	if prepared.openObserveRootEmail != "ops@example.com" {
		t.Fatalf("openObserveRootEmail = %q, want ops@example.com", prepared.openObserveRootEmail)
	}
	if prepared.openObserveRootPassword != "Complexpass#123" {
		t.Fatalf("openObserveRootPassword = %q, want Complexpass#123", prepared.openObserveRootPassword)
	}
	if prepared.openObserveOrg != "default" {
		t.Fatalf("openObserveOrg = %q, want default", prepared.openObserveOrg)
	}
	if prepared.openObserveBasicAuth == "" {
		t.Fatal("openObserveBasicAuth should not be empty")
	}
	if prepared.openObserveLogStream != "gochat_logs" {
		t.Fatalf("openObserveLogStream = %q, want gochat_logs", prepared.openObserveLogStream)
	}
	if prepared.openObserveMetricStream != "gochat-metrics" {
		t.Fatalf("openObserveMetricStream = %q, want gochat-metrics", prepared.openObserveMetricStream)
	}
	if prepared.openObserveTraceStream != "gochat_traces" {
		t.Fatalf("openObserveTraceStream = %q, want gochat_traces", prepared.openObserveTraceStream)
	}
}

func TestPrepareOptionsHelmExternalStorageRequiresS3Values(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType:            DeploymentHelm,
		StorageMode:               StorageExternal,
		BaseDomain:                "example.com",
		BackendTag:                "v1.2.3",
		FrontendTag:               "v2.3.4",
		ExternalS3Endpoint:        "objects.example.net",
		ExternalS3AccessKeyID:     "access",
		ExternalS3SecretAccessKey: "secret",
		ExternalS3UseSSL:          true,
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	if !prepared.useTLS {
		t.Fatal("helm deployment should default to TLS-aware URLs")
	}
	if prepared.wsPublicURL != "wss://example.com/ws" {
		t.Fatalf("wsPublicURL = %q, want wss://example.com/ws", prepared.wsPublicURL)
	}
	if prepared.storageEndpointURL != "https://objects.example.net" {
		t.Fatalf("storageEndpointURL = %q, want https://objects.example.net", prepared.storageEndpointURL)
	}
	if prepared.storagePublicBaseURL != "https://objects.example.net/gochat" {
		t.Fatalf("storagePublicBaseURL = %q, want https://objects.example.net/gochat", prepared.storagePublicBaseURL)
	}
	if prepared.storageAlias != "storage.alias.internal.invalid" {
		t.Fatalf("storageAlias = %q, want storage.alias.internal.invalid", prepared.storageAlias)
	}
	if prepared.minioConsoleAlias != "minio.alias.internal.invalid" {
		t.Fatalf("minioConsoleAlias = %q, want minio.alias.internal.invalid", prepared.minioConsoleAlias)
	}
}

func TestPrepareOptionsHelmAutoDisablesBundledTraefikWhenIngressClassIsSet(t *testing.T) {
	engine := NewEngine(nil)

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType:       DeploymentHelm,
		StorageMode:          StorageMinIO,
		BaseDomain:           "example.com",
		BackendTag:           "v1.2.3",
		FrontendTag:          "v2.3.4",
		IngressClassName:     "nginx",
		OpenObserveHost:      "observe.example.com",
		OpenObserveRootEmail: "ops@example.com",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	if prepared.useBundledTraefik {
		t.Fatal("helm deployment should disable bundled Traefik automatically when an ingress class is set")
	}
	if prepared.OpenObserveHost != "observe.example.com" {
		t.Fatalf("OpenObserveHost = %q, want observe.example.com", prepared.OpenObserveHost)
	}
	if prepared.openObserveRootEmail != "ops@example.com" {
		t.Fatalf("openObserveRootEmail = %q, want ops@example.com", prepared.openObserveRootEmail)
	}
}

func TestPrepareOptionsEmailProviders(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name        string
		opts        Options
		wantErrText string
	}{
		{
			name: "smtp missing host",
			opts: Options{
				DeploymentType: DeploymentCompose,
				StorageMode:    StorageMinIO,
				BaseDomain:     "example.com",
				FrontendTag:    "v2.3.4",
				EmailProvider:  "smtp",
			},
			wantErrText: "smtp host is required",
		},
		{
			name: "sendpulse missing credentials",
			opts: Options{
				DeploymentType: DeploymentCompose,
				StorageMode:    StorageMinIO,
				BaseDomain:     "example.com",
				FrontendTag:    "v2.3.4",
				EmailProvider:  "sendpulse",
			},
			wantErrText: "sendpulse user id is required",
		},
		{
			name: "resend missing key",
			opts: Options{
				DeploymentType: DeploymentCompose,
				StorageMode:    StorageMinIO,
				BaseDomain:     "example.com",
				FrontendTag:    "v2.3.4",
				EmailProvider:  "resend",
			},
			wantErrText: "resend api key is required",
		},
		{
			name: "dashamail missing key",
			opts: Options{
				DeploymentType: DeploymentCompose,
				StorageMode:    StorageMinIO,
				BaseDomain:     "example.com",
				FrontendTag:    "v2.3.4",
				EmailProvider:  "dashamail",
			},
			wantErrText: "dashamail api key is required",
		},
		{
			name: "sendpulse accepted",
			opts: Options{
				DeploymentType:  DeploymentCompose,
				StorageMode:     StorageMinIO,
				BaseDomain:      "example.com",
				FrontendTag:     "v2.3.4",
				EmailProvider:   "sendpulse",
				SendpulseUserID: "user",
				SendpulseSecret: "secret",
			},
		},
		{
			name: "resend accepted",
			opts: Options{
				DeploymentType: DeploymentCompose,
				StorageMode:    StorageMinIO,
				BaseDomain:     "example.com",
				FrontendTag:    "v2.3.4",
				EmailProvider:  "resend",
				ResendAPIKey:   "secret",
			},
		},
		{
			name: "dashamail accepted",
			opts: Options{
				DeploymentType:  DeploymentCompose,
				StorageMode:     StorageMinIO,
				BaseDomain:      "example.com",
				FrontendTag:     "v2.3.4",
				EmailProvider:   "dashamail",
				DashaMailAPIKey: "secret",
			},
		},
		{
			name: "provider normalized to lowercase",
			opts: Options{
				DeploymentType: DeploymentCompose,
				StorageMode:    StorageMinIO,
				BaseDomain:     "example.com",
				FrontendTag:    "v2.3.4",
				EmailProvider:  "SMTP",
				SMTPHost:       "mail.example.com",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.opts.BackendTag == "" {
				test.opts.BackendTag = "v1.2.3"
			}
			prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(test.opts))
			if test.wantErrText != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrText) {
					t.Fatalf("prepareOptions error = %v, want substring %q", err, test.wantErrText)
				}
				return
			}
			if err != nil {
				t.Fatalf("prepareOptions returned error: %v", err)
			}
			if test.name == "provider normalized to lowercase" && prepared.EmailProvider != "smtp" {
				t.Fatalf("EmailProvider = %q, want smtp", prepared.EmailProvider)
			}
		})
	}
}

func TestPrepareOptionsRequiresExplicitOpenObserveCredentials(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name        string
		opts        Options
		wantErrText string
	}{
		{
			name: "missing email",
			opts: Options{
				DeploymentType:          DeploymentCompose,
				StorageMode:             StorageMinIO,
				BaseDomain:              "example.com",
				BackendTag:              "v1.2.3",
				FrontendTag:             "v2.3.4",
				OpenObserveRootPassword: "Complexpass#123",
			},
			wantErrText: "openobserve root email is required",
		},
		{
			name: "missing password",
			opts: Options{
				DeploymentType:       DeploymentCompose,
				StorageMode:          StorageMinIO,
				BaseDomain:           "example.com",
				BackendTag:           "v1.2.3",
				FrontendTag:          "v2.3.4",
				OpenObserveRootEmail: "ops@example.com",
			},
			wantErrText: "openobserve root password is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := engine.prepareOptions(context.Background(), test.opts)
			if err == nil || !strings.Contains(err.Error(), test.wantErrText) {
				t.Fatalf("prepareOptions error = %v, want substring %q", err, test.wantErrText)
			}
		})
	}
}
