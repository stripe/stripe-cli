package coopcmd

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var stripeBestPracticesPinnedSHA256 = map[string]string{
	"SKILL.md":               "d666c9aa384ac20407c1178097aee4e772c9fd087c0e3d7114c1e7ef9742dd1c",
	"references/billing.md":  "c04de14ab1882c96350c32729055014747ac11b017d0d012523043a8ad72a45e",
	"references/connect.md":  "5da76cb728ab738f52b0838d5e5d4143a3aa91d73d842f69153e2a1f0ca9471f",
	"references/payments.md": "c56f39bbd590225c269e28cd01903f7d50d0041b56f821f330e72f6f843756f8",
	"references/security.md": "e4835b44d553d865cde6d7c289820edd20aaf493b27f42ca6eb664bdddb773ad",
	"references/tax.md":      "25525bf4f16f5c75295c9f7808b30fdb9ffcf2f47ec1560f7b8b54495896973b",
	"references/treasury.md": "ccd38ef29a654b2c59bde4d56e5537af6a89f10a7297ed0b87c3b9e9f2bb8ffa",
}

func TestInstallPinnedStripeBestPracticesSkillWritesCompleteExactSnapshot(t *testing.T) {
	repoRoot := t.TempDir()
	var embeddedFiles []string
	require.NoError(t, fs.WalkDir(stripeBestPracticesSkillFS, stripeBestPracticesSkillEmbedRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		embeddedFiles = append(embeddedFiles, strings.TrimPrefix(path, stripeBestPracticesSkillEmbedRoot+"/"))
		return nil
	}))
	assert.ElementsMatch(t, stripeBestPracticesSkillFiles, embeddedFiles)
	assert.ElementsMatch(t, stripeBestPracticesSkillFiles, mapKeys(stripeBestPracticesPinnedSHA256))
	assert.Equal(t, "c29cd23cfd27830bf10961d58646a9fd127fa6df", stripeBestPracticesSkillCommit)
	assert.Contains(t, stripeBestPracticesSkillSource, stripeBestPracticesSkillCommit)

	installed, err := installPinnedStripeBestPracticesSkill(repoRoot)

	require.NoError(t, err)
	assert.True(t, installed)
	for _, relativeTarget := range stripeBestPracticesSkillTargets {
		target := filepath.Join(repoRoot, relativeTarget)
		for _, relativePath := range stripeBestPracticesSkillFiles {
			want, readErr := fs.ReadFile(stripeBestPracticesSkillFS, stripeBestPracticesSkillEmbedRoot+"/"+relativePath)
			require.NoError(t, readErr)
			digest := sha256.Sum256(want)
			assert.Equal(t, stripeBestPracticesPinnedSHA256[relativePath], hex.EncodeToString(digest[:]), relativePath)
			if relativePath == "SKILL.md" {
				assert.Contains(t, string(want), "Latest Stripe API version: **2026-06-24.dahlia**")
			}
			installedWant, installErr := stripeBestPracticesInstallContents(relativePath, want)
			require.NoError(t, installErr)
			got, readErr := os.ReadFile(filepath.Join(target, filepath.FromSlash(relativePath)))
			require.NoError(t, readErr)
			assert.Equal(t, installedWant, got, relativeTarget+"/"+relativePath)
			if relativePath == "SKILL.md" {
				assert.Contains(t, string(got), "session after Co-op has selected a blueprint, integration, and API family")
				assert.Contains(t, string(got), "Do not use this skill to recommend, choose, or switch the integration or API")
				assert.NotContains(t, string(got), "Guides Stripe integration decisions across API selection")
			}

			info, statErr := os.Stat(filepath.Join(target, filepath.FromSlash(relativePath)))
			require.NoError(t, statErr)
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), relativeTarget+"/"+relativePath)
		}
	}

	installed, err = installPinnedStripeBestPracticesSkill(repoRoot)
	require.NoError(t, err)
	assert.False(t, installed)
}

func TestInstallPinnedStripeBestPracticesSkillPreservesExistingSkillAndInstallsOtherAgentTarget(t *testing.T) {
	repoRoot := t.TempDir()
	target := filepath.Join(repoRoot, ".agents", "skills", stripeBestPracticesSkillName)
	require.NoError(t, os.MkdirAll(target, 0o755))
	existing := []byte("user-managed skill\n")
	require.NoError(t, os.WriteFile(filepath.Join(target, "SKILL.md"), existing, 0o600))

	installed, err := installPinnedStripeBestPracticesSkill(repoRoot)

	require.NoError(t, err)
	assert.True(t, installed)
	got, readErr := os.ReadFile(filepath.Join(target, "SKILL.md"))
	require.NoError(t, readErr)
	assert.Equal(t, existing, got)
	assert.NoFileExists(t, filepath.Join(target, "references", "payments.md"))
	assert.FileExists(t, filepath.Join(repoRoot, stripeBestPracticesClaudeSkillTarget, "SKILL.md"))
}

func TestInstallPinnedStripeBestPracticesSkillPreservesExistingTargetFile(t *testing.T) {
	repoRoot := t.TempDir()
	target := filepath.Join(repoRoot, ".agents", "skills", stripeBestPracticesSkillName)
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	existing := []byte("reserved by repository\n")
	require.NoError(t, os.WriteFile(target, existing, 0o600))

	installed, err := installPinnedStripeBestPracticesSkillTargetAt(repoRoot, stripeBestPracticesCodexSkillTarget)

	require.NoError(t, err)
	assert.False(t, installed)
	got, readErr := os.ReadFile(target)
	require.NoError(t, readErr)
	assert.Equal(t, existing, got)
}

func TestInstallPinnedStripeBestPracticesSkillPreservesExistingEmptyTarget(t *testing.T) {
	repoRoot := t.TempDir()
	target := filepath.Join(repoRoot, stripeBestPracticesCodexSkillTarget)
	require.NoError(t, os.MkdirAll(target, 0o755))

	installed, err := installPinnedStripeBestPracticesSkillTargetAt(repoRoot, stripeBestPracticesCodexSkillTarget)

	require.NoError(t, err)
	assert.False(t, installed)
	assert.DirExists(t, target)
	assert.NoFileExists(t, filepath.Join(target, "SKILL.md"))
}

func TestInstallPinnedStripeBestPracticesSkillRecoversStaleMarkedPartialInstall(t *testing.T) {
	repoRoot := t.TempDir()
	target := filepath.Join(repoRoot, ".agents", "skills", stripeBestPracticesSkillName)
	require.NoError(t, os.MkdirAll(filepath.Join(target, "references"), 0o755))
	marker := filepath.Join(target, stripeSkillInstallMarkerName)
	require.NoError(t, os.WriteFile(marker, []byte(stripeSkillInstallMarkerContents()), 0o600))
	embeddedPayment, err := fs.ReadFile(stripeBestPracticesSkillFS, stripeBestPracticesSkillEmbedRoot+"/references/payments.md")
	require.NoError(t, err)
	partialPayment, err := stripeBestPracticesInstallContents("references/payments.md", embeddedPayment)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(target, "references", "payments.md"), partialPayment, 0o600))
	stale := time.Now().Add(-stripeSkillInstallMarkerStaleAge - time.Second)
	require.NoError(t, os.Chtimes(marker, stale, stale))

	installed, err := installPinnedStripeBestPracticesSkillTargetAt(repoRoot, stripeBestPracticesCodexSkillTarget)

	require.NoError(t, err)
	assert.True(t, installed)
	assert.NoFileExists(t, marker)
	for _, relativePath := range stripeBestPracticesSkillFiles {
		assert.FileExists(t, filepath.Join(target, filepath.FromSlash(relativePath)))
	}
}

func TestInstallPinnedStripeBestPracticesSkillPreservesUnknownFilesInStaleMarkedTarget(t *testing.T) {
	repoRoot := t.TempDir()
	target := filepath.Join(repoRoot, ".agents", "skills", stripeBestPracticesSkillName)
	require.NoError(t, os.MkdirAll(target, 0o755))
	marker := filepath.Join(target, stripeSkillInstallMarkerName)
	require.NoError(t, os.WriteFile(marker, []byte(stripeSkillInstallMarkerContents()), 0o600))
	unknown := filepath.Join(target, "user-notes.md")
	require.NoError(t, os.WriteFile(unknown, []byte("keep me\n"), 0o600))
	stale := time.Now().Add(-stripeSkillInstallMarkerStaleAge - time.Second)
	require.NoError(t, os.Chtimes(marker, stale, stale))

	installed, err := installPinnedStripeBestPracticesSkillTargetAt(repoRoot, stripeBestPracticesCodexSkillTarget)

	require.NoError(t, err)
	assert.False(t, installed)
	assert.FileExists(t, marker)
	got, readErr := os.ReadFile(unknown)
	require.NoError(t, readErr)
	assert.Equal(t, "keep me\n", string(got))
}

func TestInstallPinnedStripeBestPracticesSkillPreservesFreshMarkedInstall(t *testing.T) {
	repoRoot := t.TempDir()
	target := filepath.Join(repoRoot, ".agents", "skills", stripeBestPracticesSkillName)
	require.NoError(t, os.MkdirAll(target, 0o755))
	marker := filepath.Join(target, stripeSkillInstallMarkerName)
	require.NoError(t, os.WriteFile(marker, []byte(stripeSkillInstallMarkerContents()), 0o600))

	installed, err := installPinnedStripeBestPracticesSkillTargetAt(repoRoot, stripeBestPracticesCodexSkillTarget)

	require.NoError(t, err)
	assert.False(t, installed)
	assert.FileExists(t, marker)
	assert.NoFileExists(t, filepath.Join(target, "SKILL.md"))
}

func TestInstallPinnedStripeBestPracticesSkillPublishesAtomicallyAcrossConcurrentCalls(t *testing.T) {
	repoRoot := t.TempDir()
	const callers = 8

	type result struct {
		installed bool
		err       error
	}
	results := make(chan result, callers)
	var ready sync.WaitGroup
	ready.Add(callers)
	start := make(chan struct{})
	for range callers {
		go func() {
			ready.Done()
			<-start
			installed, err := installPinnedStripeBestPracticesSkillTargetAt(repoRoot, stripeBestPracticesCodexSkillTarget)
			results <- result{installed: installed, err: err}
		}()
	}
	ready.Wait()
	close(start)

	installedCount := 0
	for range callers {
		got := <-results
		require.NoError(t, got.err)
		if got.installed {
			installedCount++
		}
	}
	assert.Equal(t, 1, installedCount)

	target := filepath.Join(repoRoot, ".agents", "skills", stripeBestPracticesSkillName)
	for _, relativePath := range stripeBestPracticesSkillFiles {
		assert.FileExists(t, filepath.Join(target, filepath.FromSlash(relativePath)))
	}
	entries, err := os.ReadDir(filepath.Join(repoRoot, ".agents"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "skills", entries[0].Name())
}

func TestPublishStagedStripeSkillDoesNotReplaceTargetCreatedBeforePublish(t *testing.T) {
	repoRoot := t.TempDir()
	repo, err := os.OpenRoot(repoRoot)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })
	require.NoError(t, repo.MkdirAll(filepath.Join(".agents", "skills"), 0o755))
	staging, err := makeStripeSkillStagingDir(repo, ".agents")
	require.NoError(t, err)
	target := filepath.Join(".agents", "skills", stripeBestPracticesSkillName)
	require.NoError(t, repo.Mkdir(target, 0o755))

	installed, err := publishStagedStripeSkill(repo, staging, target)

	require.NoError(t, err)
	assert.False(t, installed)
	assert.DirExists(t, filepath.Join(repoRoot, target))
	assert.DirExists(t, filepath.Join(repoRoot, staging))
}

func TestInstallPinnedStripeBestPracticesSkillRejectsSymlinkedParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}

	repoRoot := t.TempDir()
	external := t.TempDir()
	require.NoError(t, os.Symlink(external, filepath.Join(repoRoot, ".agents")))

	installed, err := installPinnedStripeBestPracticesSkillTargetAt(repoRoot, stripeBestPracticesCodexSkillTarget)

	require.Error(t, err)
	assert.False(t, installed)
	assert.NoDirExists(t, filepath.Join(external, "skills", stripeBestPracticesSkillName))
}

func TestEnsureRepoStripeBestPracticesSkillDoesNotInstallAtUserScope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(home)

	err := ensureRepoStripeBestPracticesSkill()

	require.NoError(t, err)
	assert.True(t, isUserGlobalSkillScope(home))
	for _, target := range stripeBestPracticesSkillTargets {
		assert.NoDirExists(t, filepath.Join(home, target))
	}
}

func TestEnsureRepoStripeBestPracticesSkillInstallsAtNearestRepoRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, ".git"), []byte("gitdir: elsewhere\n"), 0o600))
	nested := filepath.Join(repoRoot, "app", "server")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	t.Chdir(nested)

	require.NoError(t, ensureRepoStripeBestPracticesSkill())

	for _, target := range stripeBestPracticesSkillTargets {
		for _, relativePath := range stripeBestPracticesSkillFiles {
			assert.FileExists(t, filepath.Join(repoRoot, target, filepath.FromSlash(relativePath)))
		}
	}
	assert.NoDirExists(t, filepath.Join(nested, ".agents"))
	assert.NoDirExists(t, filepath.Join(nested, ".claude"))
	assert.NoDirExists(t, filepath.Join(home, ".agents"))
	assert.NoDirExists(t, filepath.Join(home, ".claude"))
}

func TestEnsureRepoClaudeSkillsDiscoveryRootCreatesEmptyRootAtNearestRepo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, ".git"), []byte("gitdir: elsewhere\n"), 0o600))
	nested := filepath.Join(repoRoot, "app", "server")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	t.Chdir(nested)

	require.NoError(t, ensureRepoClaudeSkillsDiscoveryRoot())

	root := filepath.Join(repoRoot, ".claude", "skills")
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.NoDirExists(t, filepath.Join(nested, ".claude"))
	assert.NoDirExists(t, filepath.Join(home, ".claude"))
}

func TestEnsureRepoClaudeSkillsDiscoveryRootDoesNotCreateUserScope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(home)

	require.NoError(t, ensureRepoClaudeSkillsDiscoveryRoot())

	assert.NoDirExists(t, filepath.Join(home, ".claude"))
}

func TestStripeSkillRepoRootUsesNearestGitAncestor(t *testing.T) {
	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, ".git"), []byte("gitdir: elsewhere\n"), 0o600))
	nested := filepath.Join(repoRoot, "app", "server")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	got, err := stripeSkillRepoRoot(nested)

	require.NoError(t, err)
	assert.Equal(t, repoRoot, got)
}

func TestStripeSkillRepoRootFallsBackToCurrentScopeWithoutGit(t *testing.T) {
	start := t.TempDir()

	got, err := stripeSkillRepoRoot(start)

	require.NoError(t, err)
	assert.Equal(t, start, got)
}

func mapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func installPinnedStripeBestPracticesSkillTargetAt(repoRoot, target string) (bool, error) {
	repo, err := os.OpenRoot(repoRoot)
	if err != nil {
		return false, err
	}
	defer func() { _ = repo.Close() }()
	return installPinnedStripeBestPracticesSkillTarget(repo, target)
}
