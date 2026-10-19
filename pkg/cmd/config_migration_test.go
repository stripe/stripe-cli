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
	stamped      bool
	stampedPath  string
}

func newMigrationHarness(t *testing.T) *migrationHarness {
	t.Helper()

	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte("[default]\n  display_name = 'Account'\n"), 0600))

	h := &migrationHarness{out: &bytes.Buffer{}}
	h.migration = configMigration{
		profilesFile:      profilesFile,
		needsMigration:    func() bool { return true },
		pluginsReady:      func() bool { return true },
		incompatibilities: func() ([]plugins.ConfigV2Incompatibility, error) { return nil, nil },
		upgradePlugin: func(incompatibility plugins.ConfigV2Incompatibility) (string, error) {
			h.upgrades = append(h.upgrades, incompatibility.Plugin)
			return "1.2.0", nil
		},
		migrate: func(path string) (bool, error) {
			h.migrated = true
			h.migratedPath = path

			return true, nil
		},
		stampNew: func(path string) error {
			h.stamped = true
			h.stampedPath = path

			return nil
		},
		reload: func() error {
			h.reloaded = true

			return nil
		},
		out: h.out,
	}

	return h
}

// The common case, and the whole point of the quiet contract: the file is
// reorganized and the user sees nothing about it. They asked to run a command,
// nothing about it changed, and there is nothing for them to decide.
func TestConfigMigrationRunsWhenNeeded(t *testing.T) {
	h := newMigrationHarness(t)

	h.migration.run()

	require.True(t, h.migrated)
	require.True(t, h.reloaded)
	require.Equal(t, h.migration.profilesFile, h.migratedPath)
	require.Empty(t, h.out.String())
}

func TestNewConfigMigrationUsesEffectiveConfigPath(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "custom.toml")
	cfg := &config.Config{ProfilesFile: customPath}

	migration := newConfigMigration(cfg, t.Context())

	require.Equal(t, customPath, migration.profilesFile)
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

	// The one thing worth saying out loud, and it is said before the download rather
	// than reported after it, because the point is to explain the wait.
	require.Equal(t, "Upgrading the projects plugin so it can read the updated config file.\n", h.out.String())
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
	require.Equal(t, "Upgrading the projects plugin so it can read the updated config file.\n", h.out.String())
}

func TestConfigMigrationSkipsUntilPluginVersionsAreKnown(t *testing.T) {
	h := newMigrationHarness(t)
	h.migration.pluginsReady = func() bool { return false }

	h.migration.run()

	require.False(t, h.migrated)
	require.Empty(t, h.out.String())
}

// A migration that fails has already restored the original file, and the command
// the user asked for still runs against it -- so there is nothing to report.
func TestConfigMigrationStaysQuietWhenTheMigrationFails(t *testing.T) {
	h := newMigrationHarness(t)
	h.migration.migrate = func(string) (bool, error) {
		return false, os.ErrPermission
	}

	h.migration.run()

	require.False(t, h.reloaded)
	require.Empty(t, h.out.String())
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

// A config file that does not exist yet has nothing to migrate, but it still has a
// layout to choose. Recording v2 now means the first write -- usually a login --
// lands in the new layout, instead of writing the flat layout and migrating it on
// the next command, which would leave a backup holding a copy of the credentials
// just written.
func TestRunStampsAConfigFileThatDoesNotExistYet(t *testing.T) {
	h := newMigrationHarness(t)
	h.migration.profilesFile = filepath.Join(t.TempDir(), "absent", "config.toml")

	h.migration.run()

	require.True(t, h.stamped, "a new config file should record the v2 layout")
	require.Equal(t, h.migration.profilesFile, h.stampedPath)
	require.True(t, h.reloaded, "viper has to see the stamped file")
	require.False(t, h.migrated, "there is nothing to migrate")

	// A brand-new user has not run anything yet; a status line here would be the
	// first thing they ever see from the CLI.
	require.Empty(t, h.out.String())
}

// Gated for the same reason the migration is: once a profile is written under the
// profiles table, a plugin too old to look there cannot find it. Nothing is at
// risk, so the file just keeps the flat layout.
func TestRunDoesNotStampWhenAPluginCannotReadV2(t *testing.T) {
	h := newMigrationHarness(t)
	h.migration.profilesFile = filepath.Join(t.TempDir(), "absent", "config.toml")
	h.migration.incompatibilities = func() ([]plugins.ConfigV2Incompatibility, error) {
		return []plugins.ConfigV2Incompatibility{{
			Plugin:           "apps",
			InstalledVersion: "1.19.0",
		}}, nil
	}

	h.migration.run()

	require.False(t, h.stamped)
	require.Empty(t, h.out.String(), "nothing to upgrade for, so say nothing")
}

func TestRunDoesNotStampBeforeAnyPluginReleaseIsKnown(t *testing.T) {
	h := newMigrationHarness(t)
	h.migration.profilesFile = filepath.Join(t.TempDir(), "absent", "config.toml")
	h.migration.pluginsReady = func() bool { return false }

	h.migration.run()

	require.False(t, h.stamped)
}

func TestRunWritesNothingInV2WhileTheMinimumVersionMapIsEmpty(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "absent", "config.toml")
	migration := newConfigMigration(&config.Config{ProfilesFile: profilesFile}, t.Context())

	require.False(t, migration.pluginsReady(), "configV2MinimumVersions is empty, so the gate has to be closed")

	// The only stub, and it opens a gate rather than closing one: there is no config
	// file here, so this is what gets run() as far as the stamp.
	migration.needsMigration = func() bool { return true }

	migration.run()

	require.NoFileExists(t, profilesFile)
}
