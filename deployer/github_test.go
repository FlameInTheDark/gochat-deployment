package deployer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrepareOptionsResolvesLatestStableTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/FlameInTheDark/gochat/tags":
			switch r.URL.Query().Get("page") {
			case "1":
				writeJSON(t, w, []gitHubTag{{Name: "v1.9.0-beta.1"}, {Name: "not-a-version"}, {Name: "v1.2.0"}})
			case "2":
				writeJSON(t, w, []gitHubTag{{Name: "v1.10.0"}, {Name: "v1.3.4"}})
			default:
				writeJSON(t, w, []gitHubTag{})
			}
		case "/repos/FlameInTheDark/gochat-react/tags":
			switch r.URL.Query().Get("page") {
			case "1":
				writeJSON(t, w, []gitHubTag{{Name: "v0.9.0-rc.1"}, {Name: "v0.9.0"}})
			default:
				writeJSON(t, w, []gitHubTag{})
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	engine := NewEngine(nil)
	engine.httpClient = server.Client()
	engine.githubAPIBaseURL = server.URL

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType: DeploymentCompose,
		StorageMode:    StorageMinIO,
		BaseDomain:     "example.com",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	if prepared.backendTag != "v1.10.0" {
		t.Fatalf("backendTag = %q, want v1.10.0", prepared.backendTag)
	}
	if prepared.frontendTag != "v0.9.0" {
		t.Fatalf("frontendTag = %q, want v0.9.0", prepared.frontendTag)
	}
	if prepared.imageAPI != "ghcr.io/flameinthedark/gochat-api:v1.10.0" {
		t.Fatalf("imageAPI = %q", prepared.imageAPI)
	}
	if prepared.imageUI != "ghcr.io/flameinthedark/gochat-react:v0.9.0" {
		t.Fatalf("imageUI = %q", prepared.imageUI)
	}
	if prepared.MigrationsImageRepo != "ghcr.io/flameinthedark/gochat-migrations" {
		t.Fatalf("migrationsImageRepo = %q", prepared.MigrationsImageRepo)
	}
	if prepared.MigrationsImageTag != "v1.10.0" {
		t.Fatalf("migrationsImageTag = %q", prepared.MigrationsImageTag)
	}
}

func TestPrepareOptionsExplicitTagOverridesSkipRemoteResolution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected remote lookup: %s", r.URL.String())
	}))
	defer server.Close()

	engine := NewEngine(nil)
	engine.httpClient = server.Client()
	engine.githubAPIBaseURL = server.URL

	prepared, err := engine.prepareOptions(context.Background(), withTestOpenObserve(Options{
		DeploymentType: DeploymentCompose,
		StorageMode:    StorageMinIO,
		BaseDomain:     "example.com",
		BackendTag:     "v9.9.9",
		FrontendTag:    "v8.8.8",
	}))
	if err != nil {
		t.Fatalf("prepareOptions returned error: %v", err)
	}

	if prepared.backendTag != "v9.9.9" {
		t.Fatalf("backendTag = %q, want v9.9.9", prepared.backendTag)
	}
	if prepared.frontendTag != "v8.8.8" {
		t.Fatalf("frontendTag = %q, want v8.8.8", prepared.frontendTag)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode json: %v", err)
	}
}
