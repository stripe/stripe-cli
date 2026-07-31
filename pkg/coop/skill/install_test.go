package skill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skillTarget(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "skills", Name)
}

func readInstalledManifest(t *testing.T, target string) Manifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(target, ManifestFileName))
	require.NoError(t, err)
	var manifest Manifest
	require.NoError(t, json.Unmarshal(data, &manifest))
	return manifest
}

func TestInstallWritesSkillAndManifest(t *testing.T) {
	target := skillTarget(t)

	result, err := Install(target)
	require.NoError(t, err)
	assert.Equal(t, ResultInstalled, result)

	files, err := Files()
	require.NoError(t, err)
	for path, contents := range files {
		installed, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(path)))
		require.NoError(t, err)
		assert.Equal(t, contents, installed)
	}

	manifest := readInstalledManifest(t, target)
	assert.Equal(t, "stripe-cli", manifest.ManagedBy)
	assert.Equal(t, Name, manifest.Skill)
	assert.NotEmpty(t, manifest.CLIVersion)
	hash, err := ContentHash()
	require.NoError(t, err)
	assert.Equal(t, hash, manifest.ContentHash)
}

func TestInstallIsIdempotent(t *testing.T) {
	target := skillTarget(t)

	_, err := Install(target)
	require.NoError(t, err)
	info, err := os.Stat(filepath.Join(target, "SKILL.md"))
	require.NoError(t, err)

	result, err := Install(target)
	require.NoError(t, err)
	assert.Equal(t, ResultCurrent, result)

	unchanged, err := os.Stat(filepath.Join(target, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, info.ModTime(), unchanged.ModTime(), "a current install must not rewrite files")
}

func TestInstallUpgradesStaleManagedCopy(t *testing.T) {
	target := skillTarget(t)
	_, err := Install(target)
	require.NoError(t, err)

	// Simulate a copy installed by an older CLI: same manifest shape, stale
	// hash, stale content, plus a file the old version shipped.
	manifest := readInstalledManifest(t, target)
	manifest.ContentHash = "stale"
	manifest.CLIVersion = "0.0.1"
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(target, ManifestFileName), data, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("old"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(target, "obsolete.md"), []byte("old"), 0o644))

	result, err := Install(target)
	require.NoError(t, err)
	assert.Equal(t, ResultUpgraded, result)

	files, err := Files()
	require.NoError(t, err)
	installed, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, files["SKILL.md"], installed)
	_, err = os.Stat(filepath.Join(target, "obsolete.md"))
	assert.True(t, os.IsNotExist(err), "upgrade must remove files the new skill no longer ships")

	hash, err := ContentHash()
	require.NoError(t, err)
	assert.Equal(t, hash, readInstalledManifest(t, target).ContentHash)
}

func TestInstallRefusesUnmanagedSameNameSkill(t *testing.T) {
	target := skillTarget(t)
	require.NoError(t, os.MkdirAll(target, 0o755))
	userSkill := []byte("---\nname: stripe-coop\n---\nuser-authored\n")
	require.NoError(t, os.WriteFile(filepath.Join(target, "SKILL.md"), userSkill, 0o644))

	_, err := Install(target)
	require.ErrorIs(t, err, ErrUnmanagedSkill)

	preserved, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, userSkill, preserved, "user-authored skill must be left untouched")
}

func TestInstallRefusesForeignManagedSkill(t *testing.T) {
	target := skillTarget(t)
	require.NoError(t, os.MkdirAll(target, 0o755))
	foreign, err := json.Marshal(Manifest{ManagedBy: "someone-else", Skill: Name})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(target, ManifestFileName), foreign, 0o644))

	_, err = Install(target)
	require.ErrorIs(t, err, ErrUnmanagedSkill)
}

func TestInstallRefusesCorruptManifest(t *testing.T) {
	target := skillTarget(t)
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, ManifestFileName), []byte("{not json"), 0o644))

	_, err := Install(target)
	require.ErrorIs(t, err, ErrUnmanagedSkill)
}

func TestInstallSweepsAbandonedDebrisButKeepsFreshStaging(t *testing.T) {
	target := skillTarget(t)
	parent := filepath.Dir(target)
	require.NoError(t, os.MkdirAll(parent, 0o755))

	stale := filepath.Join(parent, "."+Name+".staging-dead1234")
	retired := filepath.Join(parent, "."+Name+".retired-dead5678")
	fresh := filepath.Join(parent, "."+Name+".staging-live1234")
	for _, dir := range []string{stale, retired, fresh} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("debris"), 0o644))
	}
	old := time.Now().Add(-2 * installDebrisMaxAge)
	require.NoError(t, os.Chtimes(stale, old, old))
	require.NoError(t, os.Chtimes(retired, old, old))

	_, err := Install(target)
	require.NoError(t, err)

	for _, gone := range []string{stale, retired} {
		_, statErr := os.Lstat(gone)
		assert.True(t, os.IsNotExist(statErr), "%s must be swept", gone)
	}
	_, statErr := os.Lstat(fresh)
	assert.NoError(t, statErr, "a fresh staging dir (possible concurrent install) must be kept")
}

func TestInstallLeavesNoStagingDebris(t *testing.T) {
	target := skillTarget(t)
	_, err := Install(target)
	require.NoError(t, err)

	entries, err := os.ReadDir(filepath.Dir(target))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, Name, entries[0].Name())
}
