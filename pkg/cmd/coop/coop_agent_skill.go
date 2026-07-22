package coopcmd

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

const (
	stripeBestPracticesSkillEmbedRoot = "guidance/stripe-best-practices"
	stripeBestPracticesSkillName      = "stripe-best-practices"
	stripeSkillInstallMarkerName      = ".coop-installing"
	stripeSkillInstallMarkerStaleAge  = 30 * time.Second
	stripeCoopSkillDescription        = `description: >-
  Supplemental Stripe implementation guidance for an active Stripe Co-op
  session after Co-op has selected a blueprint, integration, and API family.
  Do not use this skill to recommend, choose, or switch the integration or API
  family. Use it only to implement or review the selected integration.`
)

var stripeBestPracticesSkillFiles = []string{
	"SKILL.md",
	"references/billing.md",
	"references/connect.md",
	"references/payments.md",
	"references/security.md",
	"references/tax.md",
	"references/treasury.md",
}

var (
	stripeBestPracticesCodexSkillTarget  = filepath.Join(".agents", "skills", stripeBestPracticesSkillName)
	stripeBestPracticesClaudeSkillTarget = filepath.Join(".claude", "skills", stripeBestPracticesSkillName)
	stripeBestPracticesSkillTargets      = []string{
		stripeBestPracticesCodexSkillTarget,
		stripeBestPracticesClaudeSkillTarget,
	}
)

// ensureRepoStripeBestPracticesSkill makes the pinned skill available to the
// active agent after Co-op selects an integration. It never fetches at runtime
// and never replaces a skill directory that the repository already owns.
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

// installPinnedStripeBestPracticesSkill installs the complete embedded skill
// beneath repoRoot. The boolean reports whether this call created it. Existing
// skills and user-owned targets are preserved; interrupted Co-op installs are
// recognized by their marker and can be recovered safely.
func installPinnedStripeBestPracticesSkill(repoRoot string) (installed bool, err error) {
	repo, err := os.OpenRoot(repoRoot)
	if err != nil {
		return false, fmt.Errorf("opening repository root: %w", err)
	}
	defer func() { _ = repo.Close() }()

	for _, target := range stripeBestPracticesSkillTargets {
		targetInstalled, targetErr := installPinnedStripeBestPracticesSkillTarget(repo, target)
		installed = installed || targetInstalled
		if targetErr != nil {
			err = errors.Join(err, fmt.Errorf("installing Stripe skill at %s: %w", target, targetErr))
		}
	}
	return installed, err
}

func installPinnedStripeBestPracticesSkillTarget(repo *os.Root, target string) (installed bool, err error) {
	skillsRoot := filepath.Dir(target)
	preserve, err := preserveExistingStripeSkillTarget(repo, target, time.Now())
	if err != nil {
		return false, err
	}
	if preserve {
		return false, nil
	}

	if err := repo.MkdirAll(skillsRoot, 0o755); err != nil {
		return false, fmt.Errorf("creating repository skills directory: %w", err)
	}
	staging, err := makeStripeSkillStagingDir(repo, filepath.Dir(skillsRoot))
	if err != nil {
		return false, err
	}

	// Build the complete skill outside the agent's discovery directory.
	// Publication claims the final directory without replacing anything and
	// writes SKILL.md last, so the agent cannot discover a half-installed skill.
	defer func() {
		if staging == "" {
			return
		}
		if cleanupErr := repo.RemoveAll(staging); cleanupErr != nil {
			installed = false
			err = errors.Join(err, fmt.Errorf("cleaning partial Stripe skill install: %w", cleanupErr))
		}
	}()

	for _, relativePath := range stripeBestPracticesSkillFiles {
		embeddedPath := stripeBestPracticesSkillEmbedRoot + "/" + relativePath
		contents, readErr := fs.ReadFile(stripeBestPracticesSkillFS, embeddedPath)
		if readErr != nil {
			return false, fmt.Errorf("reading embedded Stripe skill file %s: %w", relativePath, readErr)
		}
		contents, readErr = stripeBestPracticesInstallContents(relativePath, contents)
		if readErr != nil {
			return false, readErr
		}

		destination := filepath.Join(staging, filepath.FromSlash(relativePath))
		if mkdirErr := repo.MkdirAll(filepath.Dir(destination), 0o755); mkdirErr != nil {
			return false, fmt.Errorf("creating Stripe skill subdirectory for %s: %w", relativePath, mkdirErr)
		}
		if writeErr := repo.WriteFile(destination, contents, 0o600); writeErr != nil {
			return false, fmt.Errorf("writing Stripe skill file %s: %w", relativePath, writeErr)
		}
	}

	return publishStagedStripeSkill(repo, staging, target)
}

func stripeBestPracticesInstallContents(relativePath string, contents []byte) ([]byte, error) {
	if relativePath != "SKILL.md" {
		return contents, nil
	}
	descriptionStart := bytes.Index(contents, []byte("description: >-\n"))
	if descriptionStart < 0 {
		return nil, fmt.Errorf("scoping pinned Stripe skill: description frontmatter not found")
	}
	descriptionEndOffset := bytes.Index(contents[descriptionStart:], []byte("\n\n---\n"))
	if descriptionEndOffset < 0 {
		return nil, fmt.Errorf("scoping pinned Stripe skill: frontmatter terminator not found")
	}
	descriptionEnd := descriptionStart + descriptionEndOffset

	scoped := make([]byte, 0, len(contents)+len(stripeCoopSkillDescription))
	scoped = append(scoped, contents[:descriptionStart]...)
	scoped = append(scoped, stripeCoopSkillDescription...)
	scoped = append(scoped, contents[descriptionEnd:]...)
	return scoped, nil
}

func preserveExistingStripeSkillTarget(repo *os.Root, target string, now time.Time) (bool, error) {
	info, err := repo.Lstat(target)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking existing Stripe skill: %w", err)
	}
	if !info.IsDir() {
		return true, nil
	}
	if _, skillErr := repo.Lstat(filepath.Join(target, "SKILL.md")); skillErr == nil {
		return true, nil
	} else if !os.IsNotExist(skillErr) {
		return false, fmt.Errorf("checking existing Stripe skill discovery file: %w", skillErr)
	}

	marker := filepath.Join(target, stripeSkillInstallMarkerName)
	markerContents, markerErr := repo.ReadFile(marker)
	if markerErr == nil && string(markerContents) == stripeSkillInstallMarkerContents() {
		markerInfo, statErr := repo.Stat(marker)
		if os.IsNotExist(statErr) {
			// A concurrent publisher can remove the marker after SKILL.md is
			// published. Preserve the completed target instead of reporting a race.
			return true, nil
		}
		if statErr != nil {
			return false, fmt.Errorf("checking interrupted Stripe skill install: %w", statErr)
		}
		if now.Sub(markerInfo.ModTime()) < stripeSkillInstallMarkerStaleAge {
			return true, nil
		}
		recovered, recoverErr := recoverInterruptedStripeSkillTarget(repo, target)
		if recoverErr != nil {
			return false, recoverErr
		}
		return !recovered, nil
	}
	if markerErr != nil && !os.IsNotExist(markerErr) {
		return false, fmt.Errorf("checking Stripe skill install marker: %w", markerErr)
	}

	// Preserve an unmarked directory even when it appears empty. It can be
	// user-owned, or another installer can have claimed it immediately before
	// publishing the marker. Removing it here would race with that publisher.
	return true, nil
}

func stripeSkillInstallMarkerContents() string {
	return "stripe-coop-skill-install:" + stripeBestPracticesSkillCommit + "\n"
}

func recoverInterruptedStripeSkillTarget(repo *os.Root, target string) (bool, error) {
	allowedFiles, allowedDirs, err := stripeSkillRecoveryState()
	if err != nil {
		return false, err
	}
	recognized, err := interruptedStripeSkillTargetMatches(repo, target, allowedFiles, allowedDirs)
	if err != nil {
		return false, err
	}
	if !recognized {
		return false, nil
	}

	// Recheck and remove only known Co-op bytes. Unknown or concurrently added
	// state keeps its containing directory nonempty and is never removed.
	return removeInterruptedStripeSkillTarget(repo, target, allowedFiles, allowedDirs)
}

func stripeSkillRecoveryState() (map[string][]byte, map[string]bool, error) {
	allowedFiles := map[string][]byte{
		stripeSkillInstallMarkerName: []byte(stripeSkillInstallMarkerContents()),
	}
	allowedDirs := map[string]bool{".": true}
	for _, relativePath := range stripeBestPracticesSkillFiles {
		embedded, err := fs.ReadFile(stripeBestPracticesSkillFS, stripeBestPracticesSkillEmbedRoot+"/"+relativePath)
		if err != nil {
			return nil, nil, fmt.Errorf("reading embedded Stripe skill file %s during recovery: %w", relativePath, err)
		}
		want, err := stripeBestPracticesInstallContents(relativePath, embedded)
		if err != nil {
			return nil, nil, err
		}
		allowedFiles[relativePath] = want
		for dir := path.Dir(relativePath); dir != "."; dir = path.Dir(dir) {
			allowedDirs[dir] = true
		}
	}
	return allowedFiles, allowedDirs, nil
}

func interruptedStripeSkillTargetMatches(repo *os.Root, target string, allowedFiles map[string][]byte, allowedDirs map[string]bool) (bool, error) {
	targetRoot, err := repo.OpenRoot(target)
	if err != nil {
		return false, fmt.Errorf("opening interrupted Stripe skill install: %w", err)
	}
	recognized := true
	walkErr := fs.WalkDir(targetRoot.FS(), ".", func(walkPath string, entry fs.DirEntry, entryErr error) error {
		if entryErr != nil {
			return entryErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			recognized = false
			return nil
		}
		if entry.IsDir() {
			if !allowedDirs[walkPath] {
				recognized = false
				return fs.SkipDir
			}
			return nil
		}
		want, ok := allowedFiles[walkPath]
		if !ok {
			recognized = false
			return nil
		}
		got, readErr := targetRoot.ReadFile(walkPath)
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(got, want) {
			recognized = false
		}
		return nil
	})
	closeErr := targetRoot.Close()
	if walkErr != nil {
		return false, fmt.Errorf("inspecting interrupted Stripe skill install: %w", walkErr)
	}
	if closeErr != nil {
		return false, fmt.Errorf("closing interrupted Stripe skill install: %w", closeErr)
	}
	return recognized, nil
}

func removeInterruptedStripeSkillTarget(repo *os.Root, target string, allowedFiles map[string][]byte, allowedDirs map[string]bool) (bool, error) {
	removeFiles := append([]string(nil), stripeBestPracticesSkillFiles...)
	removeFiles = append(removeFiles, stripeSkillInstallMarkerName)
	for _, relativePath := range removeFiles {
		destination := filepath.Join(target, filepath.FromSlash(relativePath))
		got, readErr := repo.ReadFile(destination)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return false, fmt.Errorf("rechecking interrupted Stripe skill file %s: %w", relativePath, readErr)
		}
		if !bytes.Equal(got, allowedFiles[relativePath]) {
			return false, nil
		}
		if removeErr := repo.Remove(destination); removeErr != nil && !os.IsNotExist(removeErr) {
			return false, fmt.Errorf("removing interrupted Stripe skill file %s: %w", relativePath, removeErr)
		}
	}

	directories := make([]string, 0, len(allowedDirs)-1)
	for dir := range allowedDirs {
		if dir != "." {
			directories = append(directories, dir)
		}
	}
	sort.Slice(directories, func(i, j int) bool { return len(directories[i]) > len(directories[j]) })
	for _, dir := range directories {
		destination := filepath.Join(target, filepath.FromSlash(dir))
		if removeErr := repo.Remove(destination); removeErr != nil && !os.IsNotExist(removeErr) {
			hasEntries, readErr := rootDirectoryHasEntries(repo, destination)
			if readErr == nil && hasEntries {
				return false, nil
			}
			return false, fmt.Errorf("removing interrupted Stripe skill directory %s: %w", dir, removeErr)
		}
	}
	if removeErr := repo.Remove(target); removeErr != nil && !os.IsNotExist(removeErr) {
		hasEntries, readErr := rootDirectoryHasEntries(repo, target)
		if readErr == nil && hasEntries {
			return false, nil
		}
		return false, fmt.Errorf("removing interrupted Stripe skill directory: %w", removeErr)
	}
	return true, nil
}

func rootDirectoryHasEntries(repo *os.Root, name string) (bool, error) {
	dir, err := repo.Open(name)
	if err != nil {
		return false, err
	}
	defer func() { _ = dir.Close() }()
	entries, err := dir.Readdirnames(1)
	if errors.Is(err, io.EOF) {
		return false, nil
	}
	return len(entries) > 0, err
}

func publishStagedStripeSkill(repo *os.Root, staging, target string) (installed bool, err error) {
	if err := repo.Mkdir(target, 0o755); err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("claiming Stripe skill directory: %w", err)
	}
	marker := filepath.Join(target, stripeSkillInstallMarkerName)
	if markerErr := repo.WriteFile(marker, []byte(stripeSkillInstallMarkerContents()), 0o600); markerErr != nil {
		_ = repo.Remove(target)
		return false, fmt.Errorf("marking Stripe skill installation: %w", markerErr)
	}
	defer func() {
		if err == nil {
			return
		}
		installed = false
		_, cleanupErr := recoverInterruptedStripeSkillTarget(repo, target)
		if cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("cleaning unpublished Stripe skill: %w", cleanupErr))
		}
	}()

	// Move every supporting file first. SKILL.md is the discovery marker and is
	// moved last, after the complete referenced content is already in place.
	for _, relativePath := range stripeBestPracticesSkillFiles {
		if relativePath == "SKILL.md" {
			continue
		}
		source := filepath.Join(staging, filepath.FromSlash(relativePath))
		destination := filepath.Join(target, filepath.FromSlash(relativePath))
		if mkdirErr := repo.MkdirAll(filepath.Dir(destination), 0o755); mkdirErr != nil {
			return false, fmt.Errorf("publishing Stripe skill subdirectory for %s: %w", relativePath, mkdirErr)
		}
		if renameErr := repo.Rename(source, destination); renameErr != nil {
			return false, fmt.Errorf("publishing Stripe skill file %s: %w", relativePath, renameErr)
		}
	}
	if renameErr := repo.Rename(filepath.Join(staging, "SKILL.md"), filepath.Join(target, "SKILL.md")); renameErr != nil {
		return false, fmt.Errorf("publishing Stripe skill discovery file: %w", renameErr)
	}
	if removeErr := repo.Remove(marker); removeErr != nil {
		return false, fmt.Errorf("finalizing Stripe skill installation: %w", removeErr)
	}
	return true, nil
}

func makeStripeSkillStagingDir(repo *os.Root, scopeRoot string) (string, error) {
	for range 10 {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return "", fmt.Errorf("generating Stripe skill staging directory: %w", err)
		}
		staging := filepath.Join(scopeRoot, "."+stripeBestPracticesSkillName+"-"+hex.EncodeToString(suffix[:]))
		if err := repo.Mkdir(staging, 0o755); err == nil {
			return staging, nil
		} else if !os.IsExist(err) {
			return "", fmt.Errorf("creating Stripe skill staging directory: %w", err)
		}
	}
	return "", fmt.Errorf("creating unique Stripe skill staging directory")
}
