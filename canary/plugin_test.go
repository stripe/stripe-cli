package canary

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	goversion "github.com/hashicorp/go-version"

	"github.com/stripe/stripe-cli/canary/testutil"
)

const (
	pluginManifestURL    = "https://stripe.jfrog.io/artifactory/stripe-cli-plugins-local/plugins.toml"
	recentVersionsToTest = 10
)

func TestAPIProjectsPlugin(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	versions, err := fetchRecentPluginVersions("projects", recentVersionsToTest)
	if err != nil {
		fatalf(t, "Failed to fetch plugin versions: %v", err)
	}
	if len(versions) == 0 {
		fatalf(t, "No versions found for projects plugin")
	}

	t.Logf("Testing %d recent versions: %v", len(versions), versions)

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			configDir, err := testutil.CreateTempConfigDir(fmt.Sprintf("plugin-projects-%s", version))
			if err != nil {
				fatalf(t, "Failed to create temp config dir: %v", err)
			}
			t.Cleanup(func() {
				os.RemoveAll(configDir)
			})

			versionRunner := runner.WithConfigDir(configDir).WithEnv(map[string]string{
				"STRIPE_API_KEY": testutil.GetAPIKey(),
			}).WithTimeout(2 * time.Minute)

			// Install specific version
			installResult, err := versionRunner.Run("plugin", "install", fmt.Sprintf("projects@%s", version))
			if err != nil {
				fatalf(t, "Failed to install projects@%s: %v", version, err)
			}
			if installResult.ExitCode != 0 {
				fatalf(t, "Install projects@%s failed with exit code %d. Stderr: %s", version, installResult.ExitCode, installResult.Stderr)
			}

			t.Run("Help", func(t *testing.T) {
				result, err := versionRunner.Run("projects", "--help")
				if err != nil {
					fatalf(t, "Failed to run 'stripe projects --help': %v", err)
				}

				if result.ExitCode > 1 {
					errorf(t, "projects@%s --help crashed with exit code %d. Stderr: %s", version, result.ExitCode, result.Stderr)
				}

				combined := result.Stdout + result.Stderr
				if strings.Contains(combined, "panic:") || strings.Contains(combined, "runtime error") {
					errorf(t, "projects@%s --help panicked! Output: %s", version, combined)
				}
			})

			t.Run("Catalog", func(t *testing.T) {
				result, err := versionRunner.Run("projects", "catalog")
				if err != nil {
					fatalf(t, "Failed to run 'stripe projects catalog': %v", err)
				}

				if result.ExitCode > 1 {
					errorf(t, "projects@%s catalog crashed with exit code %d. Stderr: %s", version, result.ExitCode, result.Stderr)
				}

				combined := result.Stdout + result.Stderr
				if strings.Contains(combined, "panic:") || strings.Contains(combined, "runtime error") {
					errorf(t, "projects@%s catalog panicked! Output: %s", version, combined)
				}
			})
		})
	}
}

// fetchRecentPluginVersions fetches the plugin manifest and returns the most
// recent N versions for the given plugin, sorted oldest to newest.
func fetchRecentPluginVersions(pluginName string, count int) ([]string, error) {
	resp, err := http.Get(pluginManifestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	// Extract the section for our plugin
	content := string(body)
	sections := strings.Split(content, "[[Plugin]]")

	var pluginSection string
	for _, section := range sections {
		if strings.Contains(section, fmt.Sprintf(`Shortname = "%s"`, pluginName)) {
			pluginSection = section
			break
		}
	}

	if pluginSection == "" {
		return nil, fmt.Errorf("plugin %q not found in manifest", pluginName)
	}

	// Extract unique versions
	versionRe := regexp.MustCompile(`Version = "([^"]+)"`)
	matches := versionRe.FindAllStringSubmatch(pluginSection, -1)

	seen := make(map[string]bool)
	var parsed []*goversion.Version
	for _, match := range matches {
		v := match[1]
		if seen[v] {
			continue
		}
		seen[v] = true
		gv, err := goversion.NewVersion(v)
		if err != nil {
			continue
		}
		parsed = append(parsed, gv)
	}

	sort.Sort(goversion.Collection(parsed))

	// Take the last N
	start := 0
	if len(parsed) > count {
		start = len(parsed) - count
	}
	recent := parsed[start:]

	result := make([]string, len(recent))
	for i, v := range recent {
		result[i] = v.Original()
	}
	return result, nil
}
