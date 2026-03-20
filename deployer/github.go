package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type gitHubRepo struct {
	Owner string
	Name  string
}

type gitHubTag struct {
	Name string `json:"name"`
}

type gitHubContentItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
}

type stableSemver struct {
	Raw   string
	Major int
	Minor int
	Patch int
}

var stableSemverPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func parseStableSemver(value string) (stableSemver, bool) {
	matches := stableSemverPattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(matches) != 4 {
		return stableSemver{}, false
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return stableSemver{}, false
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return stableSemver{}, false
	}
	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return stableSemver{}, false
	}

	return stableSemver{
		Raw:   strings.TrimSpace(value),
		Major: major,
		Minor: minor,
		Patch: patch,
	}, true
}

func compareStableSemver(left, right stableSemver) int {
	switch {
	case left.Major != right.Major:
		return left.Major - right.Major
	case left.Minor != right.Minor:
		return left.Minor - right.Minor
	default:
		return left.Patch - right.Patch
	}
}

func (e *Engine) resolveLatestStableTag(ctx context.Context, repo gitHubRepo) (string, error) {
	var (
		best    stableSemver
		found   bool
		page    = 1
		baseURL = strings.TrimRight(e.githubAPIBaseURL, "/")
	)

	for {
		url := fmt.Sprintf("%s/repos/%s/%s/tags?per_page=100&page=%d", baseURL, repo.Owner, repo.Name, page)
		var tags []gitHubTag
		if err := e.getJSON(ctx, url, &tags); err != nil {
			return "", fmt.Errorf("resolve latest tag for %s/%s: %w", repo.Owner, repo.Name, err)
		}
		if len(tags) == 0 {
			break
		}

		for _, tag := range tags {
			version, ok := parseStableSemver(tag.Name)
			if !ok {
				continue
			}
			if !found || compareStableSemver(version, best) > 0 {
				best = version
				found = true
			}
		}
		page++
	}

	if !found {
		return "", fmt.Errorf("no stable semver tags found for %s/%s", repo.Owner, repo.Name)
	}
	return best.Raw, nil
}

func (e *Engine) syncMigrations(ctx context.Context, backendTag, migrationsRoot string) error {
	if err := e.syncMigrationDir(ctx, e.backendRepo, backendTag, "db/postgres", filepath.Join(migrationsRoot, "postgres")); err != nil {
		return err
	}
	if err := e.syncMigrationDir(ctx, e.backendRepo, backendTag, "db/cassandra", filepath.Join(migrationsRoot, "cassandra")); err != nil {
		return err
	}
	return nil
}

func (e *Engine) syncMigrationDir(ctx context.Context, repo gitHubRepo, ref, sourceDir, targetDir string) error {
	baseURL := strings.TrimRight(e.githubAPIBaseURL, "/")
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", baseURL, repo.Owner, repo.Name, sourceDir, ref)

	var items []gitHubContentItem
	if err := e.getJSON(ctx, url, &items); err != nil {
		return fmt.Errorf("list %s at %s: %w", sourceDir, ref, err)
	}

	files := make([]gitHubContentItem, 0, len(items))
	for _, item := range items {
		if item.Type != "file" || strings.TrimSpace(item.DownloadURL) == "" {
			continue
		}
		files = append(files, item)
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files found in %s at %s", sourceDir, ref)
	}

	slices.SortFunc(files, func(left, right gitHubContentItem) int {
		return strings.Compare(left.Name, right.Name)
	})

	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("reset migration dir %s: %w", targetDir, err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create migration dir %s: %w", targetDir, err)
	}

	for _, file := range files {
		data, err := e.getBytes(ctx, file.DownloadURL)
		if err != nil {
			return fmt.Errorf("download migration %s: %w", file.Path, err)
		}
		targetPath := filepath.Join(targetDir, file.Name)
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return fmt.Errorf("write migration %s: %w", targetPath, err)
		}
	}

	return nil
}

func (e *Engine) getJSON(ctx context.Context, url string, target any) error {
	data, err := e.getBytes(ctx, url)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}

func (e *Engine) getBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gochat-deployer")
	if token := strings.TrimSpace(githubToken()); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := strings.TrimSpace(string(body))
		if detail == "" {
			detail = resp.Status
		}
		return nil, fmt.Errorf("%s returned %s: %s", url, resp.Status, detail)
	}
	return body, nil
}

func githubToken() string {
	if token := os.Getenv("GITHUB_TOKEN"); strings.TrimSpace(token) != "" {
		return token
	}
	return os.Getenv("GH_TOKEN")
}
