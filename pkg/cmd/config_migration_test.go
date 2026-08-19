package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
)

// migrationHarness is a configMigration with every gate open and nothing real
// behind it, so each test can close exactly one gate.
type migrationHarness struct {
	migration    configMigration
	out          *bytes.Buffer
	migrated     bool
	reloaded     bool
	migratedPath string
}

func newMigrationHarness(t *testing.T, answer string) *migrationHarness {
	t.Helper()

	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte("[default]\n  display_name = 'Account'\n"), 0600))

	h := &migrationHarness{out: &bytes.Buffer{}}
	h.migration = configMigration{
		profilesFile:      profilesFile,
		needsMigration:    func() bool { return true },
		pluginsReady:      func() bool { return true },
		incompatibilities: func() ([]plugins.ConfigV2Incompatibility, error) { return nil, nil },
		migrate: func(path string) (bool, error) {
			h.migrated = true
			h.migratedPath = path

			return true, nil
		},
		reload: func() error {
			h.reloaded = true

			return nil
		},
		getEnv:      func(string) string { return "" },
		interactive: true,
		in:          strings.NewReader(answer),
		out:         h.out,
	}

	return h
}

func TestConfigMigrationRunsWhenNeeded(t *testing.T) {
	h := newMigrationHarness(t, "y\n")

	h.migration.run()

	require.True(t, h.migrated)
	require.True(t, h.reloaded)
	require.Equal(t, h.migration.profilesFile, h.migratedPath)
	require.Contains(t, h.out.String(), "Updated "+h.migration.profilesFile)
	require.Contains(t, h.out.String(), config.ConfigBackupSuffix)
}

// The migration keeps a backup and the CLI reads both layouts afterwards, so a
// bare Enter accepts.
func TestConfigMigrationDefaultsToYes(t *testing.T) {
	h := newMigrationHarness(t, "\n")

	h.migration.run()

	require.True(t, h.migrated)
}

func TestConfigMigrationRespectsDeclining(t *testing.T) {
	h := newMigrationHarness(t, "n\n")

	h.migration.run()

	require.False(t, h.migrated)
	require.Contains(t, h.out.String(), "Leaving")
}

// Nothing is read from a terminal that isn't there, and a script should not have
// its config file rewritten unasked.
func TestConfigMigrationSkipsNonInteractiveSession(t *testing.T) {
	h := newMigrationHarness(t, "y\n")
	h.migration.interactive = false

	h.migration.run()

	require.False(t, h.migrated)
	require.Empty(t, h.out.String())
}

func TestConfigMigrationHonorsKillSwitch(t *testing.T) {
	h := newMigrationHarness(t, "y\n")
	h.migration.getEnv = func(name string) string {
		if name == config.SkipMigrationEnvVar {
			return "1"
		}

		return ""
	}

	h.migration.run()

	require.False(t, h.migrated)
	require.Empty(t, h.out.String())
}

func TestConfigMigrationSkipsAlreadyMigratedConfig(t *testing.T) {
	h := newMigrationHarness(t, "y\n")
	h.migration.needsMigration = func() bool { return false }

	h.migration.run()

	require.False(t, h.migrated)
	require.Empty(t, h.out.String())
}

func TestConfigMigrationSkipsMissingConfigFile(t *testing.T) {
	h := newMigrationHarness(t, "y\n")
	h.migration.profilesFile = filepath.Join(t.TempDir(), "config.toml")

	h.migration.run()

	require.False(t, h.migrated)
	require.Empty(t, h.out.String())
}

// Plugins parse the config file themselves, so migrating past one that only knows
// the flat layout would leave it unable to find any profile.
func TestConfigMigrationSkipsWhenAPluginIsTooOld(t *testing.T) {
	h := newMigrationHarness(t, "y\n")
	h.migration.incompatibilities = func() ([]plugins.ConfigV2Incompatibility, error) {
		return []plugins.ConfigV2Incompatibility{{
			Plugin:           "apps",
			InstalledVersion: "1.4.0",
			MinimumVersion:   "1.5.0",
		}}, nil
	}

	h.migration.run()

	require.False(t, h.migrated)
	require.Contains(t, h.out.String(), "apps 1.4.0")
	require.Contains(t, h.out.String(), "stripe plugin upgrade apps")
}

func TestConfigMigrationSkipsUntilPluginVersionsAreKnown(t *testing.T) {
	h := newMigrationHarness(t, "y\n")
	h.migration.pluginsReady = func() bool { return false }

	h.migration.run()

	require.False(t, h.migrated)
	require.Empty(t, h.out.String())
}

// A migration that fails has already restored the original file, and the command
// the user asked for still runs.
func TestConfigMigrationReportsFailureWithoutFailingTheCommand(t *testing.T) {
	h := newMigrationHarness(t, "y\n")
	h.migration.migrate = func(string) (bool, error) {
		return false, os.ErrPermission
	}

	h.migration.run()

	require.False(t, h.reloaded)
	require.Contains(t, h.out.String(), "Could not update")
	require.Contains(t, h.out.String(), "still reads it")
}

// Help and completion output is read by other programs, so a prompt in the middle
// of it is worse than a config file left in the old layout.
func TestMigrationSafeCommand(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	listen := &cobra.Command{Use: "listen"}
	completion := &cobra.Command{Use: "completion"}
	completionZsh := &cobra.Command{Use: "zsh"}
	help := &cobra.Command{Use: "help"}

	completion.AddCommand(completionZsh)
	root.AddCommand(listen, completion, help)

	require.True(t, migrationSafeCommand(root))
	require.True(t, migrationSafeCommand(listen))
	require.False(t, migrationSafeCommand(completion))
	require.False(t, migrationSafeCommand(completionZsh))
	require.False(t, migrationSafeCommand(help))
}
