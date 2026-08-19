package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
)

// configMigration is the policy around config.MigrateConfigFile: whether the
// file should be rewritten in the v2 layout right now, and what the user is told
// about it. Every dependency is a field so the policy can be tested without a
// terminal or an installed plugin.
type configMigration struct {
	profilesFile      string
	needsMigration    func() bool
	pluginsReady      func() bool
	incompatibilities func() ([]plugins.ConfigV2Incompatibility, error)
	migrate           func(path string) (bool, error)
	reload            func() error
	getEnv            func(string) string
	interactive       bool
	in                io.Reader
	out               io.Writer
}

func newConfigMigration(cfg *config.Config) configMigration {
	return configMigration{
		profilesFile:   cfg.ProfilesFile,
		needsMigration: config.NeedsMigration,
		pluginsReady:   plugins.ConfigV2Ready,
		incompatibilities: func() ([]plugins.ConfigV2Incompatibility, error) {
			return plugins.ConfigV2Incompatibilities(cfg, fs)
		},
		migrate:     config.MigrateConfigFile,
		reload:      config.ReloadConfigFile,
		getEnv:      os.Getenv,
		interactive: interactiveHuman(os.Getenv, term.IsTerminal(int(os.Stdin.Fd()))),
		in:          os.Stdin,
		out:         os.Stderr,
	}
}

// migrateConfigIfNeeded offers the one-time move to the v2 config layout, before
// the command the user asked for runs.
func migrateConfigIfNeeded(cmd *cobra.Command) {
	if !migrationSafeCommand(cmd) {
		return
	}

	newConfigMigration(&Config).run()
}

// migrationSafeCommand reports whether it is acceptable to write to the terminal
// and to the config file while running this command. Help and shell completion
// output gets read by other programs, and a prompt in the middle of it would be
// worse than a config file left in the old layout.
func migrationSafeCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "help", "completion":
			return false
		}
	}

	return true
}

// run migrates the config file when doing so is both needed and safe.
//
// Nothing here fails the command the user actually asked for. Every reason to
// stop is a reason the old layout stays, and the old layout keeps working: this
// version of the CLI reads both.
func (m configMigration) run() {
	logger := log.WithFields(log.Fields{
		"prefix": "cmd.configMigration.run",
	})

	if !m.needsMigration() {
		return
	}

	if m.getEnv(config.SkipMigrationEnvVar) != "" {
		logger.Debugf("Skipping the config migration because %s is set", config.SkipMigrationEnvVar)
		return
	}

	if _, err := os.Stat(m.profilesFile); err != nil {
		// No config file yet, so there is nothing to move. The first write picks
		// the layout.
		logger.Debugf("Skipping the config migration: %s", err)
		return
	}

	if !m.pluginsReady() {
		logger.Debug("Skipping the config migration: no plugin release is known to read the new config format yet")
		return
	}

	// A prompt nobody answers would hang, and migrating unasked in a script is
	// worse than not migrating at all.
	if !m.interactive {
		logger.Debug("Skipping the config migration: not an interactive session")
		return
	}

	incompatibilities, err := m.incompatibilities()
	if err != nil {
		logger.Debugf("Skipping the config migration: could not check installed plugins: %s", err)
		return
	}

	if len(incompatibilities) > 0 {
		m.reportIncompatiblePlugins(incompatibilities)
		return
	}

	if !m.confirm() {
		return
	}

	changed, err := m.migrate(m.profilesFile)
	if err != nil {
		fmt.Fprintf(m.out, "Could not update %s to the new config format: %s\n", m.profilesFile, err)
		fmt.Fprintln(m.out, "The file was left as it was, and the CLI still reads it.")

		return
	}

	if !changed {
		return
	}

	if err := m.reload(); err != nil {
		fmt.Fprintf(m.out, "Updated %s to the new config format, but could not re-read it: %s\n", m.profilesFile, err)
		return
	}

	fmt.Fprintf(m.out, "Updated %s to the new config format. The previous version is saved as %s.\n", m.profilesFile, m.profilesFile+config.ConfigBackupSuffix)
}

func (m configMigration) reportIncompatiblePlugins(incompatibilities []plugins.ConfigV2Incompatibility) {
	fmt.Fprintf(m.out, "%s uses an older config format. Updating it needs newer versions of these plugins:\n", m.profilesFile)

	for _, incompatibility := range incompatibilities {
		fmt.Fprintf(m.out, "  %s %s — run `%s`\n", incompatibility.Plugin, incompatibility.InstalledVersion, incompatibility.UpgradeCommand())
	}

	fmt.Fprintln(m.out, "Until then the file is left alone, and everything keeps working: this CLI reads both formats.")
}

// confirm asks whether to rewrite the config file. It defaults to yes, since the
// migration keeps a backup and the CLI reads either layout afterwards.
func (m configMigration) confirm() bool {
	fmt.Fprintf(m.out, "%s uses an older config format, in which a profile name can collide with a CLI setting.\n", m.profilesFile)
	fmt.Fprintf(m.out, "Update it now? The previous version is saved as %s. [Y/n]: ", m.profilesFile+config.ConfigBackupSuffix)

	answer, err := bufio.NewReader(m.in).ReadString('\n')
	if err != nil && answer == "" {
		fmt.Fprintln(m.out)
		return false
	}

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "", "y", "yes":
		return true
	default:
		fmt.Fprintf(m.out, "Leaving %s as it is. This CLI reads both formats.\n", m.profilesFile)
		return false
	}
}
