package coopcmd

import (
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
	stripeBestPracticesSkillName       = "stripe-best-practices"
	stripeBestPracticesSkillTreeRoot   = "skills/stripe-best-practices"
	stripeBestPracticesSkillGitRef     = "main"
	stripeBestPracticesSkillTreeURL    = "https://api.github.com/repos/stripe/ai/git/trees/" + stripeBestPracticesSkillGitRef + "?recursive=1"
	stripeBestPracticesSkillRawBaseURL = "https://raw.githubusercontent.com/stripe/ai/" + stripeBestPracticesSkillGitRef + "/" + stripeBestPracticesSkillTreeRoot
	stripeSkillDownloadTimeout         = 15 * time.Second
	codexProjectDirectory              = ".agents"
	claudeProjectDirectory             = ".claude"
)

var (
	stripeBestPracticesSkillTargets = []string{
		projectSkillPath(codexProjectDirectory, stripeBestPracticesSkillName),
		projectSkillPath(claudeProjectDirectory, stripeBestPracticesSkillName),
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

// ensureRepoStripeBestPracticesSkill makes the latest skill available in the
// current project after Co-op selects an integration. It never replaces a
// skill path that the project already owns.
func ensureRepoStripeBestPracticesSkill() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolving current directory: %w", err)
	}
	_, err = installStripeBestPracticesSkill(cwd)
	return err
}

// ensureProjectSkillsDiscoveryRoot creates an agent's project-local skills
// root in the current directory so the agent can watch it for later additions.
func ensureProjectSkillsDiscoveryRoot(projectDirectory string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolving current directory: %w", err)
	}
	project, err := os.OpenRoot(cwd)
	if err != nil {
		return fmt.Errorf("opening project directory: %w", err)
	}
	defer func() { _ = project.Close() }()
	if err := project.MkdirAll(projectSkillsPath(projectDirectory), 0o755); err != nil {
		return fmt.Errorf("creating project skills directory: %w", err)
	}
	return nil
}

func projectSkillsPath(projectDirectory string) string {
	return filepath.Join(projectDirectory, "skills")
}

func projectSkillPath(projectDirectory, skillName string) string {
	return filepath.Join(projectSkillsPath(projectDirectory), skillName)
}

func warnRepoStripeBestPracticesSkill(cmd *cobra.Command, err error) {
	var out io.Writer = os.Stderr
	if cmd != nil {
		out = cmd.ErrOrStderr()
	}
	fmt.Fprintf(out, "Warning: unable to install the optional project-scoped Stripe skill; continuing without it: %v\n", err)
}

func warnRepoClaudeSkillsDiscovery(cmd *cobra.Command, err error) {
	var out io.Writer = os.Stderr
	if cmd != nil {
		out = cmd.ErrOrStderr()
	}
	fmt.Fprintf(out, "Warning: unable to prepare optional project-scoped Claude skill discovery; continuing without hot-loading: %v\n", err)
}

// installStripeBestPracticesSkill installs the complete skill directly from
// stripe/ai's main branch. Existing targets are left untouched.
func installStripeBestPracticesSkill(projectDirectory string) (bool, error) {
	return installStripeBestPracticesSkillFrom(
		context.Background(),
		projectDirectory,
		stripeBestPracticesGitHubSource,
	)
}

func installStripeBestPracticesSkillFrom(ctx context.Context, projectDirectory string, source stripeSkillGitHubSource) (bool, error) {
	project, err := os.OpenRoot(projectDirectory)
	if err != nil {
		return false, fmt.Errorf("opening project directory: %w", err)
	}
	defer func() { _ = project.Close() }()

	missingTargets, err := missingStripeSkillTargets(project)
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
		targetInstalled, targetErr := installStripeSkillTarget(project, target, files)
		installed = installed || targetInstalled
		if targetErr != nil {
			err = errors.Join(err, fmt.Errorf("installing Stripe skill at %s: %w", target, targetErr))
		}
	}
	return installed, err
}

func missingStripeSkillTargets(project *os.Root) ([]string, error) {
	missing := make([]string, 0, len(stripeBestPracticesSkillTargets))
	for _, target := range stripeBestPracticesSkillTargets {
		if _, err := project.Lstat(target); err == nil {
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
		return nil, fmt.Errorf("fetching Stripe skill tree: %w", err)
	}
	var tree stripeSkillGitTree
	if err := json.Unmarshal(treeBody, &tree); err != nil {
		return nil, fmt.Errorf("parsing Stripe skill tree: %w", err)
	}
	if tree.Truncated {
		return nil, errors.New("stripe skill tree response was truncated")
	}

	prefix := stripeBestPracticesSkillTreeRoot + "/"
	relativePaths := make([]string, 0)
	for _, entry := range tree.Tree {
		if entry.Type != "blob" || !strings.HasPrefix(entry.Path, prefix) {
			continue
		}
		relativePath := strings.TrimPrefix(entry.Path, prefix)
		if !safeStripeSkillRelativePath(relativePath) {
			return nil, fmt.Errorf("stripe skill contains unsafe path %q", relativePath)
		}
		relativePaths = append(relativePaths, relativePath)
	}
	sort.Strings(relativePaths)
	if !containsString(relativePaths, "SKILL.md") {
		return nil, errors.New("stripe skill does not contain SKILL.md")
	}

	files := make(map[string][]byte, len(relativePaths))
	for _, relativePath := range relativePaths {
		rawURL, err := url.JoinPath(source.rawBaseURL, relativePath)
		if err != nil {
			return nil, fmt.Errorf("building Stripe skill URL for %s: %w", relativePath, err)
		}
		contents, err := fetchStripeSkillURL(ctx, client, rawURL)
		if err != nil {
			return nil, fmt.Errorf("fetching Stripe skill file %s: %w", relativePath, err)
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

func installStripeSkillTarget(project *os.Root, target string, files map[string][]byte) (bool, error) {
	if _, err := project.Lstat(target); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("checking existing Stripe skill: %w", err)
	}

	if err := project.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, fmt.Errorf("creating project skills directory: %w", err)
	}
	if err := project.Mkdir(target, 0o755); err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("creating Stripe skill directory: %w", err)
	}

	for _, relativePath := range sortedStripeSkillPaths(files) {
		destination := filepath.Join(target, filepath.FromSlash(relativePath))
		if err := project.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return false, fmt.Errorf("creating Stripe skill subdirectory for %s: %w", relativePath, err)
		}
		if err := project.WriteFile(destination, files[relativePath], 0o600); err != nil {
			return false, fmt.Errorf("writing Stripe skill file %s: %w", relativePath, err)
		}
	}
	return true, nil
}

func sortedStripeSkillPaths(files map[string][]byte) []string {
	paths := make([]string, 0, len(files))
	for relativePath := range files {
		paths = append(paths, relativePath)
	}
	sort.Strings(paths)
	return paths
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
