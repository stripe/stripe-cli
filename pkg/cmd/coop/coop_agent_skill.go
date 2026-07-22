package coopcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	stripeBestPracticesSkillName     = "stripe-best-practices"
	stripeBestPracticesSkillTreeRoot = "skills/stripe-best-practices"
	stripeSkillDownloadTimeout       = 15 * time.Second
	stripeCoopSkillDescription       = `description: >-
  Supplemental Stripe implementation guidance for an active Stripe Co-op
  session after Co-op has selected a blueprint, integration, and API family.
  Do not use this skill to recommend, choose, or switch the integration or API
  family. Use it only to implement or review the selected integration.`
)

var (
	stripeBestPracticesCodexSkillTarget  = filepath.Join(".agents", "skills", stripeBestPracticesSkillName)
	stripeBestPracticesClaudeSkillTarget = filepath.Join(".claude", "skills", stripeBestPracticesSkillName)
	stripeBestPracticesSkillTargets      = []string{
		stripeBestPracticesCodexSkillTarget,
		stripeBestPracticesClaudeSkillTarget,
	}
	stripeBestPracticesGitHubSource = stripeSkillGitHubSource{
		client:     &http.Client{Timeout: stripeSkillDownloadTimeout},
		treeURL:    stripeBestPracticesSkillTreeURL,
		rawBaseURL: stripeBestPracticesSkillRawBaseURL,
	}
)

type stripeSkillGitHubSource struct {
	client     *http.Client
	treeURL    string
	rawBaseURL string
}

type stripeSkillGitTree struct {
	Truncated bool `json:"truncated"`
	Tree      []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
}

// ensureRepoStripeBestPracticesSkill makes the pinned skill available to the
// active agent after Co-op selects an integration. It never replaces a skill
// path that the repository already owns.
func ensureRepoStripeBestPracticesSkill() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolving current directory: %w", err)
	}
	repoRoot, err := stripeSkillRepoRoot(cwd)
	if err != nil {
		return err
	}
	if isUserGlobalSkillScope(repoRoot) {
		return nil
	}
	_, err = installPinnedStripeBestPracticesSkill(repoRoot)
	return err
}

// ensureRepoClaudeSkillsDiscoveryRoot lets an already-running Claude Code
// discovery session notice the skill after Co-op selects a blueprint. Creating
// the empty root does not expose any Stripe routing guidance before selection.
func ensureRepoClaudeSkillsDiscoveryRoot() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolving current directory: %w", err)
	}
	repoRoot, err := stripeSkillRepoRoot(cwd)
	if err != nil {
		return err
	}
	if isUserGlobalSkillScope(repoRoot) {
		return nil
	}
	repo, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("opening repository root: %w", err)
	}
	defer func() { _ = repo.Close() }()
	if err := repo.MkdirAll(filepath.Join(".claude", "skills"), 0o755); err != nil {
		return fmt.Errorf("creating Claude Code repository skills directory: %w", err)
	}
	return nil
}

func warnRepoStripeBestPracticesSkill(cmd *cobra.Command, err error) {
	var out io.Writer = os.Stderr
	if cmd != nil {
		out = cmd.ErrOrStderr()
	}
	fmt.Fprintf(out, "Warning: unable to install the optional repo-scoped Stripe skill; continuing without it: %v\n", err)
}

func warnRepoClaudeSkillsDiscovery(cmd *cobra.Command, err error) {
	var out io.Writer = os.Stderr
	if cmd != nil {
		out = cmd.ErrOrStderr()
	}
	fmt.Fprintf(out, "Warning: unable to prepare optional repo-scoped Claude skill discovery; continuing without hot-loading: %v\n", err)
}

// stripeSkillRepoRoot returns the nearest ancestor with a .git marker. Co-op
// also supports new, not-yet-versioned projects, where the current directory
// is the repository scope.
func stripeSkillRepoRoot(start string) (string, error) {
	start, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolving repository root: %w", err)
	}

	for current := start; ; current = filepath.Dir(current) {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("checking repository root: %w", err)
		}

		if parent := filepath.Dir(current); parent == current {
			return start, nil
		}
	}
}

// isUserGlobalSkillScope prevents the no-Git fallback from turning a Co-op
// launch in $HOME (or a filesystem root) into a user-global skill install.
func isUserGlobalSkillScope(repoRoot string) bool {
	if filepath.Dir(repoRoot) == repoRoot {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	repoInfo, repoErr := os.Stat(repoRoot)
	homeInfo, homeErr := os.Stat(home)
	if repoErr == nil && homeErr == nil {
		return os.SameFile(repoInfo, homeInfo)
	}
	return filepath.Clean(repoRoot) == filepath.Clean(home)
}

// installPinnedStripeBestPracticesSkill installs the complete skill directly
// from its pinned stripe/ai GitHub commit. Existing targets are left untouched.
func installPinnedStripeBestPracticesSkill(repoRoot string) (bool, error) {
	return installPinnedStripeBestPracticesSkillFrom(
		context.Background(),
		repoRoot,
		stripeBestPracticesGitHubSource,
	)
}

func installPinnedStripeBestPracticesSkillFrom(ctx context.Context, repoRoot string, source stripeSkillGitHubSource) (bool, error) {
	repo, err := os.OpenRoot(repoRoot)
	if err != nil {
		return false, fmt.Errorf("opening repository root: %w", err)
	}
	defer func() { _ = repo.Close() }()

	missingTargets, err := missingStripeSkillTargets(repo)
	if err != nil {
		return false, err
	}
	if len(missingTargets) == 0 {
		return false, nil
	}

	files, err := fetchStripeBestPracticesSkill(ctx, source)
	if err != nil {
		return false, err
	}

	installed := false
	for _, target := range missingTargets {
		targetInstalled, targetErr := installStripeSkillTarget(repo, target, files)
		installed = installed || targetInstalled
		if targetErr != nil {
			err = errors.Join(err, fmt.Errorf("installing Stripe skill at %s: %w", target, targetErr))
		}
	}
	return installed, err
}

func missingStripeSkillTargets(repo *os.Root) ([]string, error) {
	missing := make([]string, 0, len(stripeBestPracticesSkillTargets))
	for _, target := range stripeBestPracticesSkillTargets {
		if _, err := repo.Lstat(target); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("checking existing Stripe skill at %s: %w", target, err)
		}
		missing = append(missing, target)
	}
	return missing, nil
}

func fetchStripeBestPracticesSkill(ctx context.Context, source stripeSkillGitHubSource) (map[string][]byte, error) {
	client := source.client
	if client == nil {
		client = &http.Client{Timeout: stripeSkillDownloadTimeout}
	}

	treeBody, err := fetchStripeSkillURL(ctx, client, source.treeURL)
	if err != nil {
		return nil, fmt.Errorf("fetching pinned Stripe skill tree: %w", err)
	}
	var tree stripeSkillGitTree
	if err := json.Unmarshal(treeBody, &tree); err != nil {
		return nil, fmt.Errorf("parsing pinned Stripe skill tree: %w", err)
	}
	if tree.Truncated {
		return nil, errors.New("pinned Stripe skill tree response was truncated")
	}

	prefix := stripeBestPracticesSkillTreeRoot + "/"
	relativePaths := make([]string, 0)
	for _, entry := range tree.Tree {
		if entry.Type != "blob" || !strings.HasPrefix(entry.Path, prefix) {
			continue
		}
		relativePath := strings.TrimPrefix(entry.Path, prefix)
		if !safeStripeSkillRelativePath(relativePath) {
			return nil, fmt.Errorf("pinned Stripe skill contains unsafe path %q", relativePath)
		}
		relativePaths = append(relativePaths, relativePath)
	}
	sort.Strings(relativePaths)
	if !containsString(relativePaths, "SKILL.md") {
		return nil, errors.New("pinned Stripe skill does not contain SKILL.md")
	}

	files := make(map[string][]byte, len(relativePaths))
	for _, relativePath := range relativePaths {
		rawURL, err := url.JoinPath(source.rawBaseURL, relativePath)
		if err != nil {
			return nil, fmt.Errorf("building pinned Stripe skill URL for %s: %w", relativePath, err)
		}
		contents, err := fetchStripeSkillURL(ctx, client, rawURL)
		if err != nil {
			return nil, fmt.Errorf("fetching pinned Stripe skill file %s: %w", relativePath, err)
		}
		contents, err = stripeBestPracticesInstallContents(relativePath, contents)
		if err != nil {
			return nil, err
		}
		files[relativePath] = contents
	}
	return files, nil
}

func fetchStripeSkillURL(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "stripe-cli")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d", rawURL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func safeStripeSkillRelativePath(relativePath string) bool {
	clean := path.Clean(relativePath)
	return clean != "." && clean == relativePath && !path.IsAbs(clean) && !strings.HasPrefix(clean, "../")
}

func installStripeSkillTarget(repo *os.Root, target string, files map[string][]byte) (bool, error) {
	if _, err := repo.Lstat(target); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("checking existing Stripe skill: %w", err)
	}

	if err := repo.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, fmt.Errorf("creating repository skills directory: %w", err)
	}
	if err := repo.Mkdir(target, 0o755); err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("creating Stripe skill directory: %w", err)
	}

	for _, relativePath := range orderedStripeSkillPaths(files) {
		destination := filepath.Join(target, filepath.FromSlash(relativePath))
		if err := repo.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return false, fmt.Errorf("creating Stripe skill subdirectory for %s: %w", relativePath, err)
		}
		if err := repo.WriteFile(destination, files[relativePath], 0o600); err != nil {
			return false, fmt.Errorf("writing Stripe skill file %s: %w", relativePath, err)
		}
	}
	return true, nil
}

func orderedStripeSkillPaths(files map[string][]byte) []string {
	paths := make([]string, 0, len(files))
	for relativePath := range files {
		if relativePath != "SKILL.md" {
			paths = append(paths, relativePath)
		}
	}
	sort.Strings(paths)
	return append(paths, "SKILL.md")
}

func stripeBestPracticesInstallContents(relativePath string, contents []byte) ([]byte, error) {
	if relativePath != "SKILL.md" {
		return contents, nil
	}
	descriptionStart := bytes.Index(contents, []byte("description: >-\n"))
	if descriptionStart < 0 {
		return nil, errors.New("scoping pinned Stripe skill: description frontmatter not found")
	}
	descriptionEndOffset := bytes.Index(contents[descriptionStart:], []byte("\n\n---\n"))
	if descriptionEndOffset < 0 {
		return nil, errors.New("scoping pinned Stripe skill: frontmatter terminator not found")
	}
	descriptionEnd := descriptionStart + descriptionEndOffset

	scoped := make([]byte, 0, len(contents)+len(stripeCoopSkillDescription))
	scoped = append(scoped, contents[:descriptionStart]...)
	scoped = append(scoped, stripeCoopSkillDescription...)
	scoped = append(scoped, contents[descriptionEnd:]...)
	return scoped, nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
