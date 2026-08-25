package cmd

import (
	"bytes"
	"os"
	"path/filepath"
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
	upgrades     []string
}

func newMigrationHarness(t *testing.T) *migrationHarness {
	t.Helper()

	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte("[default]\n  display_name = 'Account'\n"), 0600))

	h := &migrationHarness{out: &bytes.Buffer{}}
	h.migration = configMigration{
		profilesFile:         profilesFile,
		needsMigration:       func() bool { return true },
		pluginsReady:         func() bool { return true },
		incompatibilities:    func() ([]plugins.ConfigV2Incompatibility, error) { return nil, nil },
		installedPluginCount: func() int { return 3 },
		upgradePlugin: func(incompatibility plugins.ConfigV2Incompatibility) (string, error) {
			h.upgrades = append(h.upgrades, incompatibility.Plugin)
			return "1.2.0", nil
		},
		migrate: func(path string) (bool, error) {
			h.migrated = true
			h.migratedPath = path

			return true, nil
		},
		reload: func() error {
			h.reloaded = true

			return nil
		},
		getEnv: func(string) string { return "" },
		out:    h.out,
	}

	return h
}

func TestConfigMigrationRunsWhenNeeded(t *testing.T) {
	h := newMigrationHarness(t)

	h.migration.run()

	require.True(t, h.migrated)
	require.True(t, h.reloaded)
	require.Equal(t, h.migration.profilesFile, h.migratedPath)
	require.Contains(t, h.out.String(), "checking installed plugins... all 3 are compatible.")
	require.Contains(t, h.out.String(), "✔ updated "+h.migration.profilesFile)
	require.Contains(t, h.out.String(), "backup saved to config.toml"+config.ConfigBackupSuffix)
}

// There is no confirm prompt, so CI and AI agents migrate the same way a person
// at a terminal does.
func TestConfigMigrationDoesNotAsk(t *testing.T) {
	h := newMigrationHarness(t)

	h.migration.run()

	require.True(t, h.migrated)
	require.NotContains(t, h.out.String(), "Update it now")
	require.NotContains(t, h.out.String(), "[Y/n]")
}

func TestConfigMigrationHonorsKillSwitch(t *testing.T) {
	h := newMigrationHarness(t)
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
	h := newMigrationHarness(t)
	h.migration.needsMigration = func() bool { return false }

	h.migration.run()

	require.False(t, h.migrated)
	require.Empty(t, h.out.String())
}

func TestConfigMigrationSkipsMissingConfigFile(t *testing.T) {
	h := newMigrationHarness(t)
	h.migration.profilesFile = filepath.Join(t.TempDir(), "config.toml")

	h.migration.run()

	require.False(t, h.migrated)
	require.Empty(t, h.out.String())
}

func TestConfigMigrationUpgradesAPluginThatIsTooOld(t *testing.T) {
	h := newMigrationHarness(t)
	h.migration.incompatibilities = func() ([]plugins.ConfigV2Incompatibility, error) {
		return []plugins.ConfigV2Incompatibility{{
			Plugin:           "projects",
			InstalledVersion: "0.8.2",
			MinimumVersion:   "1.2.0",
		}}, nil
	}

	h.migration.run()

	require.Equal(t, []string{"projects"}, h.upgrades)
	require.True(t, h.migrated)
	require.Contains(t, h.out.String(), "checking installed plugins...")
	require.Contains(t, h.out.String(), "✔ upgraded projects from v0.8.2 to v1.2.0.")
	require.Contains(t, h.out.String(), "✔ updated "+h.migration.profilesFile)
}

func TestConfigMigrationDoesNotMigrateWhenPluginUpgradeFails(t *testing.T) {
	h := newMigrationHarness(t)
	h.migration.incompatibilities = func() ([]plugins.ConfigV2Incompatibility, error) {
		return []plugins.ConfigV2Incompatibility{{
			Plugin:           "projects",
			InstalledVersion: "0.8.2",
			MinimumVersion:   "1.2.0",
		}}, nil
	}
	h.migration.upgradePlugin = func(plugins.ConfigV2Incompatibility) (string, error) {
		return "", os.ErrPermission
	}

	h.migration.run()

	require.False(t, h.migrated)
	require.Contains(t, h.out.String(), "! could not upgrade projects to the minimum required version (1.2.0).")
	require.Contains(t, h.out.String(), "run `stripe plugin upgrade projects`, then try again.")
	require.Contains(t, h.out.String(), "your config file was not changed.")
}

func TestConfigMigrationSkipsUntilPluginVersionsAreKnown(t *testing.T) {
	h := newMigrationHarness(t)
	h.migration.pluginsReady = func() bool { return false }

	h.migration.run()

	require.False(t, h.migrated)
	require.Empty(t, h.out.String())
}

// A migration that fails has already restored the original file, and the command
// the user asked for still runs.
func TestConfigMigrationReportsFailureWithoutFailingTheCommand(t *testing.T) {
	h := newMigrationHarness(t)
	h.migration.migrate = func(string) (bool, error) {
		return false, os.ErrPermission
	}

	h.migration.run()

	require.False(t, h.reloaded)
	require.Contains(t, h.out.String(), "Could not update")
	require.Contains(t, h.out.String(), "still reads it")
}

// Help and completion output is read by other programs, so a status line in the
// middle of it is worse than a config file left in the old layout.
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
