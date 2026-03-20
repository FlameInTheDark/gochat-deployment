package deployer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
}

func TestSyncMigrationsDownloadsTaggedFiles(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/FlameInTheDark/gochat/contents/db/postgres":
			if got := r.URL.Query().Get("ref"); got != "v1.10.0" {
				t.Fatalf("postgres ref = %q, want v1.10.0", got)
			}
			writeJSON(t, w, []gitHubContentItem{
				{
					Name:        "000001_initial.up.sql",
					Path:        "db/postgres/000001_initial.up.sql",
					Type:        "file",
					DownloadURL: server.URL + "/raw/postgres/000001_initial.up.sql",
				},
			})
		case "/repos/FlameInTheDark/gochat/contents/db/cassandra":
			if got := r.URL.Query().Get("ref"); got != "v1.10.0" {
				t.Fatalf("cassandra ref = %q, want v1.10.0", got)
			}
			writeJSON(t, w, []gitHubContentItem{
				{
					Name:        "000001_initial.up.cql",
					Path:        "db/cassandra/000001_initial.up.cql",
					Type:        "file",
					DownloadURL: server.URL + "/raw/cassandra/000001_initial.up.cql",
				},
			})
		case "/raw/postgres/000001_initial.up.sql":
			_, _ = w.Write([]byte("postgres migration"))
		case "/raw/cassandra/000001_initial.up.cql":
			_, _ = w.Write([]byte("cassandra migration"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	engine := NewEngine(nil)
	engine.httpClient = server.Client()
	engine.githubAPIBaseURL = server.URL

	targetRoot := t.TempDir()
	if err := engine.syncMigrations(context.Background(), "v1.10.0", targetRoot); err != nil {
		t.Fatalf("syncMigrations returned error: %v", err)
	}

	postgresPath := filepath.Join(targetRoot, "postgres", "000001_initial.up.sql")
	cassandraPath := filepath.Join(targetRoot, "cassandra", "000001_initial.up.cql")

	postgresData, err := os.ReadFile(postgresPath)
	if err != nil {
		t.Fatalf("read postgres migration: %v", err)
	}
	if string(postgresData) != "postgres migration" {
		t.Fatalf("postgres migration = %q", string(postgresData))
	}

	cassandraData, err := os.ReadFile(cassandraPath)
	if err != nil {
		t.Fatalf("read cassandra migration: %v", err)
	}
	if string(cassandraData) != "cassandra migration" {
		t.Fatalf("cassandra migration = %q", string(cassandraData))
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
