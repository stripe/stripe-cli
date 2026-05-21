package cmd

import (
	"bytes"
	"errors"
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
// re-binds the persistent root flags afterward so other tests in this
// package (which rely on the global viper state set up in init()) are
// unaffected by ordering.
func resetViperWithCleanup(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		// Re-bind what package init() set up, so tests like
		// TestReadProjectFromFlag continue to work regardless of
		// test execution order.
		for _, key := range keysToReBind {
			if flag := rootCmd.PersistentFlags().Lookup(key); flag != nil {
				viper.BindPFlag(key, flag)
			}
		}
		viper.BindEnv("project-name", "STRIPE_PROJECT_NAME")
		viper.BindPFlag("color", rootCmd.PersistentFlags().Lookup("color"))
	})
}

// newTempConfig returns a *config.Config pointing at a temp file so tests
// don't touch the developer's real ~/.config/stripe/config.toml.
func newTempConfig(t *testing.T) (*config.Config, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	require.NoError(t, os.WriteFile(path, []byte(""), 0600))

	// Reset viper so each test starts clean.
	resetViperWithCleanup(t)
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")
	require.NoError(t, viper.ReadInConfig())

	cfg := &config.Config{ProfilesFile: path}
	return cfg, path
}

func TestAgentGuidanceSnooze_HappyPath(t *testing.T) {
	cfg, path := newTempConfig(t)

	cmd := newAgentGuidanceCmd(cfg)
	snooze, _, err := cmd.Find([]string{"snooze"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	snooze.SetOut(&stdout)
	snooze.SetErr(&stdout)

	require.NoError(t, snooze.RunE(snooze, []string{}))

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
	cfg := &config.Config{ProfilesFile: "/nonexistent/path/that/cannot/be/written/config.toml"}
	resetViperWithCleanup(t)
	viper.SetConfigFile(cfg.ProfilesFile)
	viper.SetConfigType("toml")

	cmd := newAgentGuidanceCmd(cfg)
	snooze, _, err := cmd.Find([]string{"snooze"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	snooze.SetOut(&stdout)
	snooze.SetErr(&stdout)

	err = snooze.RunE(snooze, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to snooze")
	// Sanity-check it's an error wrap, not a nil pointer panic
	assert.False(t, errors.Is(err, nil))
}
