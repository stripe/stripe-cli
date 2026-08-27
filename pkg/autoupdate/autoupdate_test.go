package autoupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/version"
)

// newTestConfig returns a config whose state directory is a fresh temp dir, so
// that stamp and lock files never touch the developer's real config folder.
func newTestConfig(t *testing.T) config.IConfig {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	return &config.Config{}
}

// freezeClock pins now() so interval arithmetic does not depend on wall time.
func freezeClock(t *testing.T, at time.Time) {
	original := now
	now = func() time.Time { return at }

	t.Cleanup(func() { now = original })
}

// pretendRelease makes the tests look like a released build rather than a build
// from source, which is otherwise excluded from updating at all.
func pretendRelease(t *testing.T) {
	original := version.Version
	version.Version = "1.0.0"

	t.Cleanup(func() { version.Version = original })
}

func TestSelfManaged(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()

	exe, ok := selfManaged(filepath.Join(dir, "stripe"), dir)
	assert.True(t, ok)
	assert.Equal(t, "stripe", filepath.Base(exe))

	_, ok = selfManaged(filepath.Join(other, "stripe"), dir)
	assert.False(t, ok, "a binary outside the install directory is owned by something else")

	// A subdirectory is not the install directory.
	_, ok = selfManaged(filepath.Join(dir, "nested", "stripe"), dir)
	assert.False(t, ok)

	// No resolvable home and no override means there is nothing to compare against.
	_, ok = selfManaged(filepath.Join(dir, "stripe"), "")
	assert.False(t, ok)
}

func TestSelfManagedFollowsASymlinkedInstallDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating a symlink on Windows needs elevation")
	}

	root := t.TempDir()
	real := filepath.Join(root, "real")
	link := filepath.Join(root, "link")

	require.NoError(t, os.MkdirAll(real, 0755))
	require.NoError(t, os.Symlink(real, link))

	// The binary is reached through the symlink, but lives in the real directory.
	_, ok := selfManaged(filepath.Join(link, "stripe"), real)
	assert.True(t, ok)

	// And the other way round: STRIPE_INSTALL_DIR points at the symlink.
	_, ok = selfManaged(filepath.Join(real, "stripe"), link)
	assert.True(t, ok)
}

func TestSameDirIsCaseInsensitiveOnlyOnWindows(t *testing.T) {
	assert.True(t, sameDir(filepath.Join("a", "b"), filepath.Join("a", "b")))
	assert.False(t, sameDir(filepath.Join("a", "b"), filepath.Join("a", "c")))
	assert.Equal(t, runtime.GOOS == "windows", sameDir(filepath.Join("a", "b"), filepath.Join("A", "B")))
}

func TestInstallDirHonorsTheOverride(t *testing.T) {
	t.Setenv(EnvInstallDir, filepath.Join("custom", "bin"))
	assert.Equal(t, filepath.Join("custom", "bin"), InstallDir())
}

func TestEnvDisable(t *testing.T) {
	for value, disabled := range map[string]bool{
		"":      false,
		"0":     false,
		"false": false,
		"FALSE": false,
		"1":     true,
		"true":  true,
		"yes":   true,
	} {
		t.Setenv(EnvDisable, value)
		assert.Equal(t, disabled, envDisabled(), "value %q", value)
	}
}

func TestEnabled(t *testing.T) {
	pretendRelease(t)
	viper.Reset()

	t.Cleanup(viper.Reset)

	assert.True(t, Enabled(), "on by default")

	viper.Set(config.AutoUpdateSettingKey(), false)
	assert.False(t, Enabled(), "settings.auto_update = false opts out")

	viper.Set(config.AutoUpdateSettingKey(), true)
	assert.True(t, Enabled())

	t.Setenv(EnvDisable, "1")
	assert.False(t, Enabled(), "the environment overrides an opted-in config")
}

func TestEnabledIsFalseForADevelopmentBuild(t *testing.T) {
	original := version.Version
	version.Version = "master"

	t.Cleanup(func() { version.Version = original })

	assert.True(t, IsDevBuild())
	assert.False(t, Enabled())
}

func TestDueForCheck(t *testing.T) {
	cfg := newTestConfig(t)
	start := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

	freezeClock(t, start)

	assert.True(t, dueForCheck(cfg), "no stamp yet")

	require.NoError(t, writeStamp(cfg, start))
	assert.False(t, dueForCheck(cfg), "just checked")

	freezeClock(t, start.Add(checkInterval-time.Minute))
	assert.False(t, dueForCheck(cfg))

	freezeClock(t, start.Add(checkInterval))
	assert.True(t, dueForCheck(cfg))

	// A stamp from the future means the clock moved, not that a check is overdue.
	freezeClock(t, start.Add(-48*time.Hour))
	assert.False(t, dueForCheck(cfg))
}

func TestDueForCheckWithAnUnreadableStamp(t *testing.T) {
	cfg := newTestConfig(t)
	require.NoError(t, writeStamp(cfg, time.Now()))

	stamp := filepath.Join(stateDir(cfg), stampFileName)
	require.NoError(t, os.WriteFile(stamp, []byte("not a timestamp"), 0600))

	assert.True(t, dueForCheck(cfg), "a stamp that cannot be read must not wedge updates off")
}

func TestAcquireLockIsExclusive(t *testing.T) {
	cfg := newTestConfig(t)

	freezeClock(t, time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC))

	release, err := acquireLock(cfg)
	require.NoError(t, err)

	_, err = acquireLock(cfg)
	assert.ErrorContains(t, err, "already running")

	release()

	release, err = acquireLock(cfg)
	require.NoError(t, err)
	release()
}

func TestAcquireLockBreaksAnAbandonedLock(t *testing.T) {
	cfg := newTestConfig(t)
	start := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

	freezeClock(t, start)

	release, err := acquireLock(cfg)
	require.NoError(t, err)

	defer release()

	// A worker killed mid-download never releases the lock. Pretend enough time has
	// passed that it cannot still be running.
	freezeClock(t, start.Add(2*checkInterval))

	stolen, err := acquireLock(cfg)
	require.NoError(t, err)
	stolen()
}

func TestMaybeRunDoesNothingForAnUnmanagedBinary(t *testing.T) {
	pretendRelease(t)

	cfg := newTestConfig(t)
	// The test binary is not in here, which is the situation of every
	// package-manager install.
	t.Setenv(EnvInstallDir, t.TempDir())

	MaybeRun(cfg)

	assert.NoFileExists(t, filepath.Join(stateDir(cfg), stampFileName),
		"an install the CLI does not own must not even record a check")
}

func TestRunRefusesAnUnmanagedBinary(t *testing.T) {
	pretendRelease(t)

	cfg := newTestConfig(t)
	t.Setenv(EnvInstallDir, t.TempDir())

	err := Run(t.Context(), cfg, os.Stdout)
	assert.ErrorContains(t, err, "install script")
}
