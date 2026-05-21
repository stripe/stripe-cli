package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

// resetViperWithCleanup wipes viper for the duration of the test and
// restores the bindings root.go::init() set up afterward, so other tests
// in this package (which rely on that global viper state) are unaffected
// by ordering.
func resetViperWithCleanup(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		rebindViperFromRoot()
	})
}

// newTempConfigForPath builds a *config.Config pointed at the supplied
// path and prepares viper to write to it. Pass an existing temp path
// (use t.TempDir()) for the happy-path case, or a deliberately
// unwritable path for the failure case.
//
// Cobra's OnInitialize hook (registered in root.go) re-points viper at
// the package-global Config.ProfilesFile on every Execute(), so this
// helper also redirects the global Config to the test path for the
// duration of the test and restores it on cleanup.
//
// Tests should not touch viper directly; this helper is the one obvious
// way to set up a temp-config-backed test.
func newTempConfigForPath(t *testing.T, path string) *config.Config {
	t.Helper()
	resetViperWithCleanup(t)
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")

	// Only attempt to read in config if the file exists; the failure-path
	// test uses a path that intentionally does not exist.
	if _, err := os.Stat(path); err == nil {
		require.NoError(t, viper.ReadInConfig())
	}

	// Redirect the package-global Config to the test path so that
	// cobra.OnInitialize -> Config.InitConfig (which fires inside
	// cmd.Execute()) does not clobber viper back to the developer's
	// real config. Restore on cleanup.
	prevProfilesFile := Config.ProfilesFile
	Config.ProfilesFile = path
	t.Cleanup(func() { Config.ProfilesFile = prevProfilesFile })

	return &config.Config{ProfilesFile: path}
}

// newTempConfig returns a *config.Config pointing at a freshly-created
// temp file so tests don't touch the developer's real
// ~/.config/stripe/config.toml.
func newTempConfig(t *testing.T) (*config.Config, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(""), 0600))
	return newTempConfigForPath(t, path), path
}

func TestAgentGuidanceSnooze_HappyPath(t *testing.T) {
	cfg, path := newTempConfig(t)

	cmd := newAgentGuidanceCmd(cfg)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"snooze"})

	require.NoError(t, cmd.Execute())

	assert.Contains(t, stdout.String(), "Agent guidance snoozed")

	// Read the file directly to assert what was actually written.
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	today := time.Now().Format("2006-01-02")
	// Viper's TOML writer may emit either basic strings (double-quoted) or
	// literal strings (single-quoted); accept either.
	assert.True(t,
		strings.Contains(string(contents), fmt.Sprintf(`snoozed_until = "%s"`, today)) ||
			strings.Contains(string(contents), fmt.Sprintf(`snoozed_until = '%s'`, today)),
		"expected snoozed_until = %q somewhere in config, got: %s", today, string(contents),
	)
	assert.True(t, strings.Contains(string(contents), "[agent_guidance]") ||
		strings.Contains(string(contents), "agent_guidance.snoozed_until"))
}

func TestAgentGuidanceSnooze_WriteFailure(t *testing.T) {
	cfg := newTempConfigForPath(t, "/nonexistent/path/that/cannot/be/written/config.toml")

	cmd := newAgentGuidanceCmd(cfg)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"snooze"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to snooze")
}
