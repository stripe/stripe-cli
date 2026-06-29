package canary

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stripe/stripe-cli/canary/testutil"
)

func TestAPIProjectsPlugin(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	configDir, err := testutil.CreateTempConfigDir("plugin-projects")
	if err != nil {
		fatalf(t, "Failed to create temp config dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(configDir)
	})

	runner = runner.WithConfigDir(configDir).WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}).WithTimeout(2 * time.Minute)

	// Install the plugin first — all subtests depend on this
	installResult, err := runner.Run("plugin", "install", "projects")
	if err != nil {
		fatalf(t, "Failed to run 'stripe plugin install projects': %v", err)
	}
	if installResult.ExitCode != 0 {
		fatalf(t, "Plugin install failed with exit code %d. Stderr: %s", installResult.ExitCode, installResult.Stderr)
	}

	combined := installResult.Stdout + installResult.Stderr
	if !strings.Contains(combined, "installation") && !strings.Contains(combined, "already installed") {
		errorf(t, "Expected installation confirmation in output, got: %s", combined)
	}

	t.Run("Help", func(t *testing.T) {
		result, err := runner.Run("projects", "--help")
		if err != nil {
			fatalf(t, "Failed to run 'stripe projects --help': %v", err)
		}

		logSanitizedf(t, "stdout: %s", result.Stdout)

		if result.ExitCode != 0 {
			errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
		}

		if !strings.Contains(result.Stdout, "projects") && !strings.Contains(result.Stderr, "projects") {
			errorf(t, "Expected 'projects' in help output, got stdout: %s", result.Stdout)
		}
	})

	t.Run("Catalog", func(t *testing.T) {
		result, err := runner.Run("projects", "catalog")
		if err != nil {
			fatalf(t, "Failed to run 'stripe projects catalog': %v", err)
		}

		logSanitizedf(t, "stdout: %s", result.Stdout)

		if result.ExitCode > 1 {
			errorf(t, "Command crashed with exit code %d (possible panic). Stderr: %s", result.ExitCode, result.Stderr)
		}

		combined := result.Stdout + result.Stderr
		if strings.Contains(combined, "panic:") || strings.Contains(combined, "runtime error") {
			errorf(t, "Plugin panicked! Output: %s", combined)
		}
	})
}
