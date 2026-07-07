// Package agentskills installs Stripe agent skills natively, reproducing what
// `npx skills add https://docs.stripe.com` does without any Node/npx dependency.
// It fetches the skills index from docs.stripe.com and writes each skill's files
// to a destination directory, preserving the skill/relative-path layout.
package agentskills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IndexURL is the canonical Stripe skills index. It is a var (not a const) so
// tests can point it at an httptest server; individual file URLs are derived
// from it, so overriding it redirects file fetches too.
var IndexURL = "https://docs.stripe.com/.well-known/skills/index.json"

const requestTimeout = 10 * time.Second

// Index is the top-level response from IndexURL.
type Index struct {
	Skills []Skill `json:"skills"`
}

// Skill is one entry in the index. Files lists the skill's artifacts as paths
// relative to the skill directory (e.g. "SKILL.md", "references/billing.md").
type Skill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
}

func clientOrDefault(httpClient *http.Client) *http.Client {
	if httpClient != nil {
		return httpClient
	}
	return &http.Client{Timeout: requestTimeout}
}

// filesBaseURL is the directory the index lives in, e.g.
// "https://docs.stripe.com/.well-known/skills/". Skill files are fetched from
// filesBaseURL + "<skill>/<file>".
func filesBaseURL() string {
	return strings.TrimSuffix(IndexURL, "index.json")
}

// FetchIndex retrieves the skills index from docs.stripe.com.
func FetchIndex(ctx context.Context, httpClient *http.Client) (*Index, error) {
	body, err := get(ctx, clientOrDefault(httpClient), IndexURL)
	if err != nil {
		return nil, fmt.Errorf("fetching skills index: %w", err)
	}

	var index Index
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("parsing skills index: %w", err)
	}
	return &index, nil
}

// Install fetches every file of every skill in the index and writes it under
// destDir preserving the skill/relative-path layout (destDir/<skill>/<file>).
// Installation is best-effort: a file that fails to fetch or write is skipped.
// It returns the names of skills that had at least one file written.
func Install(ctx context.Context, httpClient *http.Client, destDir string) ([]string, error) {
	client := clientOrDefault(httpClient)

	index, err := FetchIndex(ctx, client)
	if err != nil {
		return nil, err
	}

	base := filesBaseURL()
	var installed []string
	for _, skill := range index.Skills {
		if skill.Name == "" {
			continue
		}
		wrote := false
		for _, file := range skill.Files {
			if file == "" {
				continue
			}
			target := filepath.Join(destDir, skill.Name, filepath.FromSlash(file))
			if !isUnderDir(target, destDir) {
				continue
			}
			content, err := get(ctx, client, base+skill.Name+"/"+file)
			if err != nil {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				continue
			}
			if err := os.WriteFile(target, content, 0600); err != nil {
				continue
			}
			wrote = true
		}
		if wrote {
			installed = append(installed, skill.Name)
		}
	}

	return installed, nil
}

// isUnderDir reports whether target is strictly under dir after cleaning both
// paths. This rejects path traversal via "../" or absolute paths in skill names.
func isUnderDir(target, dir string) bool {
	cleanTarget := filepath.Clean(target)
	cleanDir := filepath.Clean(dir) + string(filepath.Separator)
	return strings.HasPrefix(cleanTarget, cleanDir)
}

func get(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	return body, nil
}
