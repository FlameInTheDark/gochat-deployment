package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
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
