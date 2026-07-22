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

func TestInstallStripeBestPracticesSkillDownloadsCompleteGitHubSkill(t *testing.T) {
	projectDirectory := t.TempDir()
	source := startStripeSkillSource(t, testStripeBestPracticesSkillFiles)

	installed, err := installStripeBestPracticesSkillFrom(context.Background(), projectDirectory, source)

	require.NoError(t, err)
	assert.True(t, installed)
	for _, relativeTarget := range stripeBestPracticesSkillTargets {
		target := filepath.Join(projectDirectory, relativeTarget)
		for relativePath, sourceContents := range testStripeBestPracticesSkillFiles {
			installedPath := filepath.Join(target, filepath.FromSlash(relativePath))
			got, readErr := os.ReadFile(installedPath)
			require.NoError(t, readErr)
			assert.Equal(t, sourceContents, got, relativeTarget+"/"+relativePath)

			info, statErr := os.Stat(installedPath)
			require.NoError(t, statErr)
			assert.True(t, info.Mode().IsRegular(), relativeTarget+"/"+relativePath)
			if runtime.GOOS != "windows" {
				assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), relativeTarget+"/"+relativePath)
			}
		}
	}

	installed, err = installStripeBestPracticesSkillFrom(context.Background(), projectDirectory, stripeSkillGitHubSource{
		treeURL: "://must-not-be-fetched",
	})
	require.NoError(t, err)
	assert.False(t, installed)
}

func TestInstallStripeBestPracticesSkillPreservesExistingSkillAndInstallsOtherAgentTarget(t *testing.T) {
	projectDirectory := t.TempDir()
	target := filepath.Join(projectDirectory, projectSkillPath(codexProjectDirectory, stripeBestPracticesSkillName))
	require.NoError(t, os.MkdirAll(target, 0o755))
	existing := []byte("user-managed skill\n")
	require.NoError(t, os.WriteFile(filepath.Join(target, "SKILL.md"), existing, 0o600))
	source := startStripeSkillSource(t, testStripeBestPracticesSkillFiles)

	installed, err := installStripeBestPracticesSkillFrom(context.Background(), projectDirectory, source)

	require.NoError(t, err)
	assert.True(t, installed)
	got, readErr := os.ReadFile(filepath.Join(target, "SKILL.md"))
	require.NoError(t, readErr)
	assert.Equal(t, existing, got)
	assert.NoFileExists(t, filepath.Join(target, "references", "payments.md"))
	assert.FileExists(t, filepath.Join(projectDirectory, projectSkillPath(claudeProjectDirectory, stripeBestPracticesSkillName), "SKILL.md"))
}

func TestInstallStripeBestPracticesSkillDoesNothingWhenTargetsExist(t *testing.T) {
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
			projectDirectory := t.TempDir()
			for _, relativeTarget := range stripeBestPracticesSkillTargets {
				tt.create(t, filepath.Join(projectDirectory, relativeTarget))
			}

			installed, err := installStripeBestPracticesSkillFrom(context.Background(), projectDirectory, stripeSkillGitHubSource{
				treeURL: "://must-not-be-fetched",
			})

			require.NoError(t, err)
			assert.False(t, installed)
		})
	}
}

func TestInstallStripeBestPracticesSkillFetchFailureDoesNotCreateTargets(t *testing.T) {
	projectDirectory := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	installed, err := installStripeBestPracticesSkillFrom(context.Background(), projectDirectory, stripeSkillGitHubSource{
		client:     server.Client(),
		treeURL:    server.URL + "/tree",
		rawBaseURL: server.URL + "/raw",
	})

	require.Error(t, err)
	assert.False(t, installed)
	assert.NoDirExists(t, filepath.Join(projectDirectory, codexProjectDirectory))
	assert.NoDirExists(t, filepath.Join(projectDirectory, claudeProjectDirectory))
}

func TestInstallStripeBestPracticesSkillRejectsUnsafeGitHubPath(t *testing.T) {
	projectDirectory := t.TempDir()
	source := startStripeSkillSource(t, map[string][]byte{
		"SKILL.md":  testStripeBestPracticesSkillFiles["SKILL.md"],
		"../escape": []byte("do not write me\n"),
	})

	installed, err := installStripeBestPracticesSkillFrom(context.Background(), projectDirectory, source)

	require.ErrorContains(t, err, "unsafe path")
	assert.False(t, installed)
	assert.NoDirExists(t, filepath.Join(projectDirectory, codexProjectDirectory))
	assert.NoFileExists(t, filepath.Join(projectDirectory, "escape"))
}

func TestInstallStripeBestPracticesSkillRejectsSymlinkedParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}

	projectDirectory := t.TempDir()
	external := t.TempDir()
	require.NoError(t, os.Symlink(external, filepath.Join(projectDirectory, codexProjectDirectory)))

	installed, err := installStripeBestPracticesSkillFrom(
		context.Background(),
		projectDirectory,
		startStripeSkillSource(t, testStripeBestPracticesSkillFiles),
	)

	require.Error(t, err)
	assert.False(t, installed)
	assert.NoDirExists(t, filepath.Join(external, "skills", stripeBestPracticesSkillName))
}

func TestEnsureRepoStripeBestPracticesSkillInstallsInCurrentDirectory(t *testing.T) {
	useStripeSkillTestSource(t, testStripeBestPracticesSkillFiles)
	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, ".git"), []byte("gitdir: elsewhere\n"), 0o600))
	projectDirectory := filepath.Join(repoRoot, "app", "server")
	require.NoError(t, os.MkdirAll(projectDirectory, 0o755))
	t.Chdir(projectDirectory)

	require.NoError(t, ensureRepoStripeBestPracticesSkill())

	for _, target := range stripeBestPracticesSkillTargets {
		for relativePath := range testStripeBestPracticesSkillFiles {
			assert.FileExists(t, filepath.Join(projectDirectory, target, filepath.FromSlash(relativePath)))
		}
	}
	assert.NoDirExists(t, filepath.Join(repoRoot, codexProjectDirectory))
	assert.NoDirExists(t, filepath.Join(repoRoot, claudeProjectDirectory))
}

func TestEnsureProjectSkillsDiscoveryRootCreatesEmptyRootInCurrentDirectory(t *testing.T) {
	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, ".git"), []byte("gitdir: elsewhere\n"), 0o600))
	projectDirectory := filepath.Join(repoRoot, "app", "server")
	require.NoError(t, os.MkdirAll(projectDirectory, 0o755))
	t.Chdir(projectDirectory)

	require.NoError(t, ensureProjectSkillsDiscoveryRoot(claudeProjectDirectory))

	root := filepath.Join(projectDirectory, projectSkillsPath(claudeProjectDirectory))
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.NoDirExists(t, filepath.Join(repoRoot, claudeProjectDirectory))
}

func TestStripeBestPracticesSourceTracksMain(t *testing.T) {
	assert.Equal(t, "main", stripeBestPracticesSkillGitRef)
	assert.Contains(t, stripeBestPracticesSkillTreeURL, "/main?recursive=1")
	assert.Contains(t, stripeBestPracticesSkillRawBaseURL, "/main/skills/stripe-best-practices")
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
