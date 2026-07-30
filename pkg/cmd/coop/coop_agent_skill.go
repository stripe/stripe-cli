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

	// stripeSkillCompletionMarker is the file that makes an installed skill
	// discoverable by an agent. It doubles as the marker for a finished
	// install: it is written last, so a target directory without it is a
	// partial install that a later run repairs instead of skipping.
	stripeSkillCompletionMarker = "SKILL.md"
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
// current project after Co-op selects an integration. It never replaces an
// installed skill or any other path that the project already owns.
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
// stripe/ai's main branch. Targets that already hold an installed skill are
// left untouched.
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
		needsInstall, err := stripeSkillTargetNeedsInstall(project, target)
		if err != nil {
			return nil, err
		}
		if needsInstall {
			missing = append(missing, target)
		}
	}
	return missing, nil
}

// stripeSkillTargetNeedsInstall reports whether target still needs the skill
// written to it. A directory without the completion marker is a partial
// install — left by an interrupted run, or by a run that failed partway
// through its writes — and is repaired rather than skipped. Anything else that
// already occupies the path belongs to the project and is left alone.
func stripeSkillTargetNeedsInstall(project *os.Root, target string) (bool, error) {
	info, err := project.Lstat(target)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking existing Stripe skill at %s: %w", target, err)
	}
	if !info.IsDir() {
		return false, nil
	}

	marker := filepath.Join(target, stripeSkillCompletionMarker)
	if _, err := project.Lstat(marker); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("checking existing Stripe skill at %s: %w", marker, err)
	}
	return true, nil
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
	if !containsString(relativePaths, stripeSkillCompletionMarker) {
		return nil, errors.New("stripe skill does not contain " + stripeSkillCompletionMarker)
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
	needsInstall, err := stripeSkillTargetNeedsInstall(project, target)
	if err != nil || !needsInstall {
		return false, err
	}

	// Writing into target covers both a fresh install and finishing a partial
	// one. Files the project may still own are never removed.
	if err := project.MkdirAll(target, 0o755); err != nil {
		return false, fmt.Errorf("creating Stripe skill directory: %w", err)
	}
	return true, writeStripeSkillFiles(project, target, files)
}

// writeStripeSkillFiles writes every skill file under directory, saving the
// completion marker for last so an interrupted write leaves a directory that
// stripeSkillTargetNeedsInstall still reports as needing an install.
func writeStripeSkillFiles(project *os.Root, directory string, files map[string][]byte) error {
	marker, ok := files[stripeSkillCompletionMarker]
	if !ok {
		return fmt.Errorf("stripe skill does not contain %s", stripeSkillCompletionMarker)
	}

	for _, relativePath := range sortedStripeSkillPaths(files) {
		if relativePath == stripeSkillCompletionMarker {
			continue
		}
		if err := writeStripeSkillFile(project, directory, relativePath, files[relativePath]); err != nil {
			return err
		}
	}
	return writeStripeSkillFile(project, directory, stripeSkillCompletionMarker, marker)
}

func writeStripeSkillFile(project *os.Root, directory, relativePath string, contents []byte) error {
	destination := filepath.Join(directory, filepath.FromSlash(relativePath))
	if err := project.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("creating Stripe skill subdirectory for %s: %w", relativePath, err)
	}
	if err := project.WriteFile(destination, contents, 0o600); err != nil {
		return fmt.Errorf("writing Stripe skill file %s: %w", relativePath, err)
	}
	return nil
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
