package coopcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testStripeBestPracticesSkillFiles = map[string][]byte{
	"SKILL.md": []byte(`---
name: stripe-best-practices
description: >-
  Guides Stripe integration decisions across API selection.

---

# Stripe best practices
`),
	"references/payments.md": []byte("# Payments\n"),
	"references/tax.md":      []byte("# Tax\n"),
}

func TestInstallPinnedStripeBestPracticesSkillDownloadsCompleteGitHubSkill(t *testing.T) {
	repoRoot := t.TempDir()
	source := startStripeSkillSource(t, testStripeBestPracticesSkillFiles)

	installed, err := installPinnedStripeBestPracticesSkillFrom(context.Background(), repoRoot, source)

	require.NoError(t, err)
	assert.True(t, installed)
	assert.Contains(t, stripeBestPracticesSkillSource, stripeBestPracticesSkillCommit)
	for _, relativeTarget := range stripeBestPracticesSkillTargets {
		target := filepath.Join(repoRoot, relativeTarget)
		for relativePath, sourceContents := range testStripeBestPracticesSkillFiles {
			want, scopeErr := stripeBestPracticesInstallContents(relativePath, sourceContents)
			require.NoError(t, scopeErr)
			installedPath := filepath.Join(target, filepath.FromSlash(relativePath))
			got, readErr := os.ReadFile(installedPath)
			require.NoError(t, readErr)
			assert.Equal(t, want, got, relativeTarget+"/"+relativePath)
			if relativePath == "SKILL.md" {
				assert.Contains(t, string(got), "session after Co-op has selected a blueprint, integration, and API family")
				assert.Contains(t, string(got), "Do not use this skill to recommend, choose, or switch the integration or API")
				assert.NotContains(t, string(got), "Guides Stripe integration decisions across API selection")
			}

			info, statErr := os.Stat(installedPath)
			require.NoError(t, statErr)
			assert.True(t, info.Mode().IsRegular(), relativeTarget+"/"+relativePath)
			if runtime.GOOS != "windows" {
				assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), relativeTarget+"/"+relativePath)
			}
		}
	}

	installed, err = installPinnedStripeBestPracticesSkillFrom(context.Background(), repoRoot, stripeSkillGitHubSource{
		treeURL: "://must-not-be-fetched",
	})
	require.NoError(t, err)
	assert.False(t, installed)
}

func TestInstallPinnedStripeBestPracticesSkillPreservesExistingSkillAndInstallsOtherAgentTarget(t *testing.T) {
	repoRoot := t.TempDir()
	target := filepath.Join(repoRoot, ".agents", "skills", stripeBestPracticesSkillName)
	require.NoError(t, os.MkdirAll(target, 0o755))
	existing := []byte("user-managed skill\n")
	require.NoError(t, os.WriteFile(filepath.Join(target, "SKILL.md"), existing, 0o600))
	source := startStripeSkillSource(t, testStripeBestPracticesSkillFiles)

	installed, err := installPinnedStripeBestPracticesSkillFrom(context.Background(), repoRoot, source)

	require.NoError(t, err)
	assert.True(t, installed)
	got, readErr := os.ReadFile(filepath.Join(target, "SKILL.md"))
	require.NoError(t, readErr)
	assert.Equal(t, existing, got)
	assert.NoFileExists(t, filepath.Join(target, "references", "payments.md"))
	assert.FileExists(t, filepath.Join(repoRoot, stripeBestPracticesClaudeSkillTarget, "SKILL.md"))
}

func TestInstallPinnedStripeBestPracticesSkillDoesNothingWhenTargetsExist(t *testing.T) {
	tests := []struct {
		name   string
		create func(t *testing.T, target string)
	}{
		{
			name: "directories",
			create: func(t *testing.T, target string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(target, 0o755))
			},
		},
		{
			name: "files",
			create: func(t *testing.T, target string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
				require.NoError(t, os.WriteFile(target, []byte("reserved by repository\n"), 0o600))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			for _, relativeTarget := range stripeBestPracticesSkillTargets {
				tt.create(t, filepath.Join(repoRoot, relativeTarget))
			}

			installed, err := installPinnedStripeBestPracticesSkillFrom(context.Background(), repoRoot, stripeSkillGitHubSource{
				treeURL: "://must-not-be-fetched",
			})

			require.NoError(t, err)
			assert.False(t, installed)
		})
	}
}

func TestInstallPinnedStripeBestPracticesSkillFetchFailureDoesNotCreateTargets(t *testing.T) {
	repoRoot := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	installed, err := installPinnedStripeBestPracticesSkillFrom(context.Background(), repoRoot, stripeSkillGitHubSource{
		client:     server.Client(),
		treeURL:    server.URL + "/tree",
		rawBaseURL: server.URL + "/raw",
	})

	require.Error(t, err)
	assert.False(t, installed)
	assert.NoDirExists(t, filepath.Join(repoRoot, ".agents"))
	assert.NoDirExists(t, filepath.Join(repoRoot, ".claude"))
}

func TestInstallPinnedStripeBestPracticesSkillRejectsUnsafeGitHubPath(t *testing.T) {
	repoRoot := t.TempDir()
	source := startStripeSkillSource(t, map[string][]byte{
		"SKILL.md":  testStripeBestPracticesSkillFiles["SKILL.md"],
		"../escape": []byte("do not write me\n"),
	})

	installed, err := installPinnedStripeBestPracticesSkillFrom(context.Background(), repoRoot, source)

	require.ErrorContains(t, err, "unsafe path")
	assert.False(t, installed)
	assert.NoDirExists(t, filepath.Join(repoRoot, ".agents"))
	assert.NoFileExists(t, filepath.Join(repoRoot, "escape"))
}

func TestInstallPinnedStripeBestPracticesSkillRejectsSymlinkedParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}

	repoRoot := t.TempDir()
	external := t.TempDir()
	require.NoError(t, os.Symlink(external, filepath.Join(repoRoot, ".agents")))

	installed, err := installPinnedStripeBestPracticesSkillFrom(
		context.Background(),
		repoRoot,
		startStripeSkillSource(t, testStripeBestPracticesSkillFiles),
	)

	require.Error(t, err)
	assert.False(t, installed)
	assert.NoDirExists(t, filepath.Join(external, "skills", stripeBestPracticesSkillName))
}

func TestEnsureRepoStripeBestPracticesSkillDoesNotInstallAtUserScope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(home)
	original := stripeBestPracticesGitHubSource
	stripeBestPracticesGitHubSource = stripeSkillGitHubSource{treeURL: "://must-not-be-fetched"}
	t.Cleanup(func() { stripeBestPracticesGitHubSource = original })

	require.NoError(t, ensureRepoStripeBestPracticesSkill())

	assert.True(t, isUserGlobalSkillScope(home))
	for _, target := range stripeBestPracticesSkillTargets {
		assert.NoDirExists(t, filepath.Join(home, target))
	}
}

func TestEnsureRepoStripeBestPracticesSkillInstallsAtNearestRepoRoot(t *testing.T) {
	useStripeSkillTestSource(t, testStripeBestPracticesSkillFiles)
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, ".git"), []byte("gitdir: elsewhere\n"), 0o600))
	nested := filepath.Join(repoRoot, "app", "server")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	t.Chdir(nested)

	require.NoError(t, ensureRepoStripeBestPracticesSkill())

	for _, target := range stripeBestPracticesSkillTargets {
		for relativePath := range testStripeBestPracticesSkillFiles {
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

func startStripeSkillSource(t *testing.T, files map[string][]byte) stripeSkillGitHubSource {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tree":
			entries := make([]map[string]string, 0, len(files)+2)
			entries = append(entries,
				map[string]string{"path": stripeBestPracticesSkillTreeRoot, "type": "tree"},
				map[string]string{"path": "skills/unrelated/SKILL.md", "type": "blob"},
			)
			for relativePath := range files {
				entries = append(entries, map[string]string{
					"path": stripeBestPracticesSkillTreeRoot + "/" + relativePath,
					"type": "blob",
				})
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"truncated": false,
				"tree":      entries,
			}))
		case strings.HasPrefix(r.URL.Path, "/raw/"):
			relativePath := strings.TrimPrefix(r.URL.Path, "/raw/")
			contents, ok := files[relativePath]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(contents)
		default:
			t.Errorf("unexpected skill source request: %s", r.URL.String())
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return stripeSkillGitHubSource{
		client:     server.Client(),
		treeURL:    server.URL + "/tree",
		rawBaseURL: server.URL + "/raw",
	}
}

func useStripeSkillTestSource(t *testing.T, files map[string][]byte) {
	t.Helper()
	original := stripeBestPracticesGitHubSource
	stripeBestPracticesGitHubSource = startStripeSkillSource(t, files)
	t.Cleanup(func() { stripeBestPracticesGitHubSource = original })
}

func TestStripeSkillGitHubSourceErrorIncludesURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	_, err := fetchStripeBestPracticesSkill(context.Background(), stripeSkillGitHubSource{
		client:     server.Client(),
		treeURL:    server.URL + "/tree",
		rawBaseURL: server.URL + "/raw",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("%s/tree returned %d", server.URL, http.StatusNotFound))
}
