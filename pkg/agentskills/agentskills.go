// Package agentskills installs Stripe agent skills natively, reproducing what
// `npx skills add https://docs.stripe.com` does without any Node/npx dependency.
// It fetches the skills index from docs.stripe.com and writes each skill's files
// to a destination directory, preserving the skill/relative-path layout.
package agentskills

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

// IndexURL is the canonical Stripe skills index. It is a var (not a const) so
// tests can point it at an httptest server; individual file URLs are derived
// from it, so overriding it redirects file fetches too.
var IndexURL = "https://docs.stripe.com/.well-known/skills/index.json"

const (
	requestTimeout         = 5 * time.Second
	skillCheckConcurrency  = 8
	remoteFetchConcurrency = 8
)

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

const (
	StatusNotInstalled = "not_installed"
	StatusCurrent      = "current"
	StatusOutOfDate    = "out_of_date"
	StatusError        = "error"
)

// SkillCheck is the per-skill result of comparing a local install to the index.
type SkillCheck struct {
	Name         string   `json:"name"`
	Status       string   `json:"status"`
	MissingFiles []string `json:"missing_files,omitempty"`
	ChangedFiles []string `json:"changed_files,omitempty"`
}

// DirStatus summarizes installed skills under a single destination directory.
type DirStatus struct {
	Dir            string       `json:"dir"`
	Status         string       `json:"status"`
	Skills         []SkillCheck `json:"skills,omitempty"`
	InstalledCount int          `json:"installed_count"`
	OutOfDateCount int          `json:"out_of_date_count"`
	Error          string       `json:"error,omitempty"`
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

// Check compares the skills installed under destDir against the remote index.
// It fetches remote content for each indexed file and compares SHA256 hashes
// with the local copies.
func Check(ctx context.Context, httpClient *http.Client, destDir string) (*DirStatus, error) {
	client := clientOrDefault(httpClient)

	index, err := FetchIndex(ctx, client)
	if err != nil {
		return &DirStatus{Dir: destDir, Status: StatusError, Error: err.Error()}, err
	}

	result := &DirStatus{
		Dir:    destDir,
		Status: StatusNotInstalled,
	}

	base := filesBaseURL()
	remote := newLimitedGetter(client, remoteFetchConcurrency)
	type skillJob struct {
		skill Skill
	}
	jobs := make([]skillJob, 0, len(index.Skills))
	for _, skill := range index.Skills {
		if skill.Name == "" {
			continue
		}
		jobs = append(jobs, skillJob{skill: skill})
	}

	checks := make([]SkillCheck, len(jobs))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(skillCheckConcurrency)
	for i, job := range jobs {
		i, job := i, job
		g.Go(func() error {
			checks[i] = checkSkill(ctx, remote, destDir, base, job.skill)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return &DirStatus{Dir: destDir, Status: StatusError, Error: err.Error()}, err
	}

	for _, check := range checks {
		if check.Status == StatusCurrent || check.Status == StatusOutOfDate {
			result.InstalledCount++
		}
		if check.Status == StatusOutOfDate {
			result.OutOfDateCount++
		}
		result.Skills = append(result.Skills, check)
	}

	result.Status = aggregateDirStatus(result)
	return result, nil
}

type limitedGetter struct {
	client *http.Client
	sem    chan struct{}
}

func newLimitedGetter(client *http.Client, concurrency int) *limitedGetter {
	return &limitedGetter{
		client: client,
		sem:    make(chan struct{}, concurrency),
	}
}

func (g *limitedGetter) get(ctx context.Context, rawURL string) ([]byte, error) {
	select {
	case g.sem <- struct{}{}:
		defer func() { <-g.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return get(ctx, g.client, rawURL)
}

func checkSkill(ctx context.Context, remote *limitedGetter, destDir, base string, skill Skill) SkillCheck {
	check := SkillCheck{Name: skill.Name}

	skillDir := filepath.Join(destDir, skill.Name)
	if info, err := os.Stat(skillDir); err != nil || !info.IsDir() {
		check.Status = StatusNotInstalled
		return check
	}

	type fileJob struct {
		file string
	}
	jobs := make([]fileJob, 0, len(skill.Files))
	for _, file := range skill.Files {
		if file == "" {
			continue
		}
		target := filepath.Join(destDir, skill.Name, filepath.FromSlash(file))
		if !isUnderDir(target, destDir) {
			continue
		}
		jobs = append(jobs, fileJob{file: file})
	}

	type fileOutcome struct {
		file     string
		missing  bool
		changed  bool
		hasLocal bool
	}
	outcomes := make([]fileOutcome, len(jobs))

	g, ctx := errgroup.WithContext(ctx)
	for i, job := range jobs {
		i, job := i, job
		g.Go(func() error {
			target := filepath.Join(destDir, skill.Name, filepath.FromSlash(job.file))
			local, err := os.ReadFile(target)
			if err != nil {
				outcomes[i] = fileOutcome{file: job.file, missing: true}
				return nil
			}

			outcome := fileOutcome{file: job.file, hasLocal: true}
			remoteContent, err := remote.get(ctx, base+skill.Name+"/"+job.file)
			if err != nil {
				// We have a local copy but couldn't fetch the remote to
				// compare. Bias toward "out of date" rather than "current":
				// updating is idempotent and never discards local work, so a
				// false positive just re-syncs, whereas a false "current" could
				// hide real drift. (A fully unreachable index is already caught
				// earlier and reported as StatusError.)
				outcome.changed = true
				outcomes[i] = outcome
				return nil
			}
			if !bytes.Equal(hashBytes(local), hashBytes(remoteContent)) {
				outcome.changed = true
			}
			outcomes[i] = outcome
			return nil
		})
	}
	// Each goroutine records its result in outcomes[i] and always returns nil,
	// so g.Wait never reports an error; we intentionally ignore its return and
	// derive the skill status from the collected outcomes below.
	_ = g.Wait()

	hasFiles := false
	for _, outcome := range outcomes {
		if outcome.missing {
			check.MissingFiles = append(check.MissingFiles, outcome.file)
			continue
		}
		if outcome.hasLocal {
			hasFiles = true
		}
		if outcome.changed {
			check.ChangedFiles = append(check.ChangedFiles, outcome.file)
		}
	}

	switch {
	case !hasFiles && len(check.MissingFiles) > 0:
		check.Status = StatusNotInstalled
	case len(check.MissingFiles) > 0 || len(check.ChangedFiles) > 0:
		check.Status = StatusOutOfDate
	default:
		check.Status = StatusCurrent
	}
	return check
}

func aggregateDirStatus(result *DirStatus) string {
	if result.InstalledCount == 0 {
		return StatusNotInstalled
	}

	total := len(result.Skills)
	if result.OutOfDateCount > 0 || result.InstalledCount < total {
		return StatusOutOfDate
	}
	return StatusCurrent
}

func hashBytes(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
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
