package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	cfgpkg "github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// CustomTestConfig is a test config that allows overriding the config folder path.
type CustomTestConfig struct {
	TestConfig
	customConfigPath string
}

// GetConfigFolder overrides the TestConfig method to return a custom path.
func (c *CustomTestConfig) GetConfigFolder(xdgPath string) string {
	return c.customConfigPath
}

func TestListPluginsUsesAuthenticatedEndpoint(t *testing.T) {
	config := &TestConfig{}
	config.InitConfig()

	var authenticatedLookups int
	apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/list-plugins":
			authenticatedLookups++
			require.Equal(t, runtime.GOOS, req.URL.Query().Get("os"))
			require.Equal(t, runtime.GOARCH, req.URL.Query().Get("arch"))
			require.Equal(t, "Bearer "+config.Profile.APIKey, req.Header.Get("Authorization"))
			require.Equal(t, stripe.APIVersion, req.Header.Get("Stripe-Version"))
			_, _ = res.Write(testListEndpointResponseJSON())
		case "/ajax/stripecli/list-plugins":
			t.Fatalf("authenticated list should not hit the anonymous endpoint: %s", req.URL.String())
		default:
			t.Fatalf("unexpected request URL: %s", req.URL.String())
		}
	}))
	defer apiServer.Close()

	pluginList, err := ListPlugins(context.Background(), config, apiServer.URL, "")
	require.NoError(t, err)
	require.Equal(t, 1, authenticatedLookups)
	require.Len(t, pluginList.Plugins, 1)
	require.Equal(t, "apps", pluginList.Plugins[0].Shortname)
	require.Equal(t, "Build and manage Stripe Apps", pluginList.Plugins[0].Shortdesc)
	require.Len(t, pluginList.Plugins[0].Commands, 1)
	require.Equal(t, "create", pluginList.Plugins[0].Commands[0].Name)
	require.Len(t, pluginList.Plugins[0].Releases, 1)
	require.Equal(t, "1.12.0", pluginList.Plugins[0].Releases[0].Version)
}

func TestListPluginsUsesAnonymousEndpointWhenAPIKeyNotConfigured(t *testing.T) {
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""

	apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		t.Fatalf("anonymous list should not hit the API host: %s", req.URL.String())
	}))
	defer apiServer.Close()

	var anonymousLookups int
	dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/ajax/stripecli/list-plugins":
			anonymousLookups++
			require.Equal(t, runtime.GOOS, req.URL.Query().Get("os"))
			require.Equal(t, runtime.GOARCH, req.URL.Query().Get("arch"))
			require.Empty(t, req.Header.Get("Authorization"))
			require.Equal(t, stripe.APIVersion, req.Header.Get("Stripe-Version"))
			_, _ = res.Write(testListEndpointResponseJSON())
		case "/v1/stripecli/list-plugins":
			t.Fatalf("anonymous list should not hit the authenticated endpoint: %s", req.URL.String())
		default:
			t.Fatalf("unexpected request URL: %s", req.URL.String())
		}
	}))
	defer dashboardServer.Close()

	pluginList, err := ListPlugins(context.Background(), config, apiServer.URL, dashboardServer.URL)
	require.NoError(t, err)
	require.Equal(t, 1, anonymousLookups)
	require.Len(t, pluginList.Plugins, 1)
	require.Equal(t, "apps", pluginList.Plugins[0].Shortname)
}

func TestLookUpPluginUsesLocalMetadata(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, err := LookUpPlugin(context.Background(), config, fs, "appB")
	require.NoError(t, err)
	require.Equal(t, "appB", plugin.Shortname)
	require.Equal(t, "stripe-cli-app-b", plugin.Binary)
	require.Equal(t, "FDBE6FB9-A149-44BD-9639-4D33D8B594E8", plugin.MagicCookieValue)
	require.Len(t, plugin.Releases, 5)
}

func TestLookUpPluginFallsBackToCachedManifest(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	manifestContent, err := os.ReadFile("./test_artifacts/plugins.toml")
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(fs, getCachedPluginManifestPath(config), manifestContent, os.ModePerm))

	plugin, err := LookUpPlugin(context.Background(), config, fs, "appA")
	require.NoError(t, err)
	require.Equal(t, "appA", plugin.Shortname)
	require.Equal(t, "stripe-cli-app-a", plugin.Binary)
	require.NotEmpty(t, plugin.Releases)
}

func TestLookUpPluginReturnsErrPluginNotFoundWithoutLocalMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	_, err := LookUpPlugin(context.Background(), config, fs, "missing")
	require.Error(t, err)

	var pluginNotFound *ErrPluginNotFound
	require.ErrorAs(t, err, &pluginNotFound)
	require.Equal(t, "missing", pluginNotFound.Name)
}

func TestGetInstalledPluginNamesIncludesLocalMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InstalledPlugins = []string{"projects"}

	localPlugin := Plugin{
		Shortname:        "docs",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}

	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	pluginNames, err := GetInstalledPluginNames(config, fs)
	require.NoError(t, err)
	require.Equal(t, []string{"projects", "docs"}, pluginNames)
}

func TestRecordInstalledPlugin(t *testing.T) {
	config := &TestConfig{}

	require.NoError(t, RecordInstalledPlugin(config, "docs"))
	require.Equal(t, []string{"docs"}, config.GetInstalledPlugins())

	require.NoError(t, RecordInstalledPlugin(config, "docs"))
	require.Equal(t, []string{"docs"}, config.GetInstalledPlugins())
}

func TestRemoveInstalledPlugin(t *testing.T) {
	config := &TestConfig{}
	config.InstalledPlugins = []string{"projects", "docs"}

	require.NoError(t, RemoveInstalledPlugin(config, "docs"))
	require.Equal(t, []string{"projects"}, config.GetInstalledPlugins())

	require.NoError(t, RemoveInstalledPlugin(config, "docs"))
	require.Equal(t, []string{"projects"}, config.GetInstalledPlugins())
}

func TestPersistInstalledPluginState(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	plugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "search",
				Desc: "Search docs",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}

	require.NoError(t, PersistInstalledPluginState(config, fs, plugin))
	require.Equal(t, []string{"docs"}, config.GetInstalledPlugins())

	cachedPlugin, err := readLocalPluginMetadata(config, fs, "docs")
	require.NoError(t, err)
	require.Equal(t, plugin, cachedPlugin)
}

func TestPersistInstalledPluginStateRollsBackOnConfigWriteFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &FailingWriteConfig{
		WriteErr:                 errors.New("boom"),
		MutateInstalledPluginsOn: true,
	}
	plugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}

	err := PersistInstalledPluginState(config, fs, plugin)
	require.ErrorIs(t, err, config.WriteErr)

	metadataPath, err := getLocalPluginMetadataPath(config, "docs")
	require.NoError(t, err)
	metadataExists, err := afero.Exists(fs, metadataPath)
	require.NoError(t, err)
	require.False(t, metadataExists)
	require.Empty(t, config.GetInstalledPlugins())
}

func TestPersistInstalledPluginStateRestoresPreviousMetadataOnConfigWriteFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &FailingWriteConfig{
		WriteErr:                 errors.New("boom"),
		MutateInstalledPluginsOn: true,
	}
	existingPlugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Existing docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}
	updatedPlugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Updated docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.1.0",
				Sum:     "def456",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, existingPlugin))

	err := PersistInstalledPluginState(config, fs, updatedPlugin)
	require.ErrorIs(t, err, config.WriteErr)

	cachedPlugin, err := readLocalPluginMetadata(config, fs, "docs")
	require.NoError(t, err)
	require.Equal(t, existingPlugin, cachedPlugin)
	require.Empty(t, config.GetInstalledPlugins())
}

func TestResolvePluginForInstallUsesLocalMetadataAsMetadataBase(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	localPlugin := Plugin{
		Shortname:        "generate",
		Binary:           "stripe-cli-generate",
		MagicCookieValue: "GENERATE-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "create",
				Desc: "Create generated artifacts",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	metadataManifest := fmt.Sprintf(`[[Plugin]]
  Shortname = "generate"
  Shortdesc = "Generate things"
  Binary = "stripe-cli-generate"
  MagicCookieValue = "GENERATE-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "1.0.0"
    Sum = "abc123"
`, runtime.GOARCH, runtime.GOOS)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "https://example.test/generate/1.0.0",
				PluginManifest: metadataManifest,
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "generate", "1.0.0", stripeServer.URL, stripeServer.URL)
	require.NoError(t, err)
	require.Equal(t, "1.0.0", resolvedPlugin.Version)
	require.Len(t, resolvedPlugin.Plugin.Commands, 1)
	require.Equal(t, "create", resolvedPlugin.Plugin.Commands[0].Name)

	release := resolvedPlugin.Plugin.getReleaseForVersion("1.0.0")
	require.NotNil(t, release)
}

func TestResolvePluginForInstallUsesAnonymousMetadataWithoutCachedManifest(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	var metadataLookups int
	apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		t.Fatalf("anonymous install resolution should not hit the API host: %s", req.URL.String())
	}))
	defer apiServer.Close()

	dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/ajax/stripecli/plugins_metadata":
			metadataLookups++
			require.False(t, req.URL.Query().Has("machine_uuid"))
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "https://example.test/appA/2.0.1",
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer dashboardServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", apiServer.URL, dashboardServer.URL)
	require.NoError(t, err)
	require.NotNil(t, resolvedPlugin.Plugin)
	require.Equal(t, "appA", resolvedPlugin.Plugin.Shortname)
	require.Equal(t, "2.0.1", resolvedPlugin.Version)
	require.Equal(t, "https://example.test/appA/2.0.1", resolvedPlugin.BinaryURL)
	require.Equal(t, 1, metadataLookups)
}

func TestResolvePluginForInstallFallsBackToCachedLocalMetadataWhenEndpointFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	localPlugin := Plugin{
		Shortname:        "appA",
		Binary:           "stripe-cli-app-a",
		MagicCookieValue: "APP-A-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "2.0.1",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	failingServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			res.WriteHeader(http.StatusInternalServerError)
			_, _ = res.Write([]byte(`{"error":{"message":"boom"}}`))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer failingServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", failingServer.URL, failingServer.URL)
	require.NoError(t, err)
	require.Equal(t, "appA", resolvedPlugin.Plugin.Shortname)
	require.Equal(t, "2.0.1", resolvedPlugin.Version)
	require.Empty(t, resolvedPlugin.BinaryURL)
	// Nothing answered the auto-install question, so the caller keeps prompting
	// rather than installing on an assumption.
	require.False(t, resolvedPlugin.AutoInstall)
}

func TestResolvePluginForInstallCarriesAutoInstallFromMetadata(t *testing.T) {
	optedIn, optedOut := true, false
	manifest := fmt.Sprintf(`[[Plugin]]
  Shortname = "appA"
  Binary = "stripe-cli-app-a"
  MagicCookieValue = "APP-A-COOKIE"
  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "2.0.1"
    Sum = "abc123"
`, runtime.GOARCH, runtime.GOOS)

	tests := []struct {
		name string
		// nil leaves the field out entirely, as an older server would.
		autoInstall     *bool
		wantAutoInstall bool
	}{
		{
			name:            "backend enabled auto-install",
			autoInstall:     &optedIn,
			wantAutoInstall: true,
		},
		{
			name:        "backend disabled auto-install",
			autoInstall: &optedOut,
		},
		{
			name: "server did not answer",
		},
	}

	for _, authenticated := range []bool{true, false} {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("authenticated=%t/%s", authenticated, tt.name), func(t *testing.T) {
				fs := afero.NewMemMapFs()
				config := &TestConfig{MachineUUID: "machine-abc"}
				config.InitConfig()
				path := "/v1/stripecli/get-plugin-metadata"
				if !authenticated {
					config.Profile.APIKey = ""
					path = "/ajax/stripecli/plugins_metadata"
				}

				response := map[string]interface{}{
					"binary_url":      "https://example.test/appA/2.0.1",
					"plugin_manifest": manifest,
				}
				if tt.autoInstall != nil {
					response["auto_install"] = *tt.autoInstall
				}

				stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
					require.Equal(t, path, req.URL.Path)
					require.False(t, req.URL.Query().Has("machine_uuid"))
					body, err := json.Marshal(response)
					require.NoError(t, err)
					_, _ = res.Write(body)
				}))
				defer stripeServer.Close()

				resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", stripeServer.URL, stripeServer.URL)
				require.NoError(t, err)
				require.Equal(t, "2.0.1", resolvedPlugin.Version)
				require.Equal(t, tt.wantAutoInstall, resolvedPlugin.AutoInstall)
			})
		}
	}
}

func TestResolvePluginForInstallPrefersFresherCachedManifestWhenEndpointFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	localPlugin := Plugin{
		Shortname:        "generate",
		Shortdesc:        "Generate things",
		Binary:           "stripe-cli-generate",
		MagicCookieValue: "GENERATE-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "create",
				Desc: "Create generated artifacts",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	manifestContent := []byte(fmt.Sprintf(`[[Plugin]]
  Shortname = "generate"
  Shortdesc = "Generate things"
  Binary = "stripe-cli-generate"
  MagicCookieValue = "GENERATE-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "1.0.0"
    Sum = "abc123"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "1.1.0"
    Sum = "def456"
`, runtime.GOARCH, runtime.GOOS, runtime.GOARCH, runtime.GOOS))
	require.NoError(t, afero.WriteFile(fs, getCachedPluginManifestPath(config), manifestContent, os.ModePerm))

	failingServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusInternalServerError)
		_, _ = res.Write([]byte(`{"error":{"message":"boom"}}`))
	}))
	defer failingServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "generate", "1.0.0", failingServer.URL, failingServer.URL)
	require.NoError(t, err)
	require.Equal(t, "1.0.0", resolvedPlugin.Version)
	require.Equal(t, "1.1.0", resolvedPlugin.Plugin.LookUpLatestVersion())
	require.Len(t, resolvedPlugin.Plugin.Commands, 1)
	require.Equal(t, "create", resolvedPlugin.Plugin.Commands[0].Name)

	release := resolvedPlugin.Plugin.getReleaseForVersion("1.0.0")
	require.NotNil(t, release)
}

func TestResolvePluginForUpgradeUsesMetadataEndpointWhenAvailable(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	localPlugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "search",
				Desc: "Search docs",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "0.1.25",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	var metadataLookups int
	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			metadataLookups++
			require.Equal(t, "", req.URL.Query().Get("version"))
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL: "https://example.test/docs/latest",
				PluginManifest: fmt.Sprintf(`[[Plugin]]
  Shortname = "docs"
  Shortdesc = "Docs plugin"
  Binary = "stripe-cli-docs"
  MagicCookieValue = "DOCS-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "0.1.26"
    Sum = "def456"
`, runtime.GOARCH, runtime.GOOS),
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	resolvedPlugin, err := ResolvePluginForUpgrade(context.Background(), config, fs, "docs", stripeServer.URL, stripeServer.URL)
	require.NoError(t, err)
	require.Equal(t, "0.1.26", resolvedPlugin.Plugin.LookUpLatestVersion())
	require.Equal(t, "0.1.26", resolvedPlugin.Version)
	require.Equal(t, "https://example.test/docs/latest", resolvedPlugin.BinaryURL)
	require.Len(t, resolvedPlugin.Plugin.Commands, 1)
	require.Equal(t, "search", resolvedPlugin.Plugin.Commands[0].Name)
	require.Equal(t, 1, metadataLookups)
}

func TestResolvePluginForUpgradeUsesAnonymousMetadataEndpointWhenAPIKeyUnavailable(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""

	localPlugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "search",
				Desc: "Search docs",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "0.1.25",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	var metadataLookups int
	apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		t.Fatalf("anonymous upgrade resolution should not hit the API host: %s", req.URL.String())
	}))
	defer apiServer.Close()

	dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/ajax/stripecli/plugins_metadata":
			metadataLookups++
			require.Equal(t, "", req.URL.Query().Get("version"))
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL: "https://example.test/docs/latest",
				PluginManifest: fmt.Sprintf(`[[Plugin]]
  Shortname = "docs"
  Shortdesc = "Docs plugin"
  Binary = "stripe-cli-docs"
  MagicCookieValue = "DOCS-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "0.1.26"
    Sum = "def456"
`, runtime.GOARCH, runtime.GOOS),
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer dashboardServer.Close()

	resolvedPlugin, err := ResolvePluginForUpgrade(context.Background(), config, fs, "docs", apiServer.URL, dashboardServer.URL)
	require.NoError(t, err)
	require.Equal(t, "0.1.26", resolvedPlugin.Plugin.LookUpLatestVersion())
	require.Equal(t, "0.1.26", resolvedPlugin.Version)
	require.Equal(t, "https://example.test/docs/latest", resolvedPlugin.BinaryURL)
	require.Len(t, resolvedPlugin.Plugin.Commands, 1)
	require.Equal(t, "search", resolvedPlugin.Plugin.Commands[0].Name)
	require.Equal(t, 1, metadataLookups)
}

func TestResolvePluginForUpgradeFallsBackToCachedMetadataWhenEndpointFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	localPlugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "search",
				Desc: "Search docs",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "0.1.25",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	failingServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusInternalServerError)
		_, _ = res.Write([]byte(`{"error":{"message":"boom"}}`))
	}))
	defer failingServer.Close()

	resolvedPlugin, err := ResolvePluginForUpgrade(context.Background(), config, fs, "docs", failingServer.URL, failingServer.URL)
	require.NoError(t, err)
	require.Equal(t, localPlugin, *resolvedPlugin.Plugin)
	require.Equal(t, "0.1.25", resolvedPlugin.Version)
	require.Empty(t, resolvedPlugin.BinaryURL)
}

func TestResolvePluginForUpgradePrefersFresherCachedManifestWhenEndpointFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	localPlugin := Plugin{
		Shortname:        "appA",
		Binary:           "stripe-cli-app-a",
		MagicCookieValue: "0337A75A-C3C4-4DCF-A9EF-E7A144E5A291",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.1",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	manifestContent, err := os.ReadFile("./test_artifacts/plugins.toml")
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(fs, getCachedPluginManifestPath(config), manifestContent, os.ModePerm))

	failingServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusInternalServerError)
		_, _ = res.Write([]byte(`{"error":{"message":"boom"}}`))
	}))
	defer failingServer.Close()

	resolvedPlugin, err := ResolvePluginForUpgrade(context.Background(), config, fs, "appA", failingServer.URL, failingServer.URL)
	require.NoError(t, err)
	require.Equal(t, "2.0.1", resolvedPlugin.Version)
	require.Equal(t, "2.0.1", resolvedPlugin.Plugin.LookUpLatestVersion())
}

func TestResolveCachedPluginForUpgradeUsesLocalMetadataWhenPresent(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "search",
				Desc: "Search docs",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "0.1.25",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	plugin, err := resolveCachedPluginForUpgrade(config, fs, "docs")
	require.NoError(t, err)
	require.Equal(t, localPlugin, *plugin)
}

func TestBackfillMissingInstalledPluginMetadataWritesLocalMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.InstalledPlugins = []string{"appA"}

	pluginBinaryPath := filepath.Join(getPluginsDir(config), "appA", "2.0.1", "stripe-cli-app-a"+GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("installed"), 0755))

	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	var requestedVersion string
	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			requestedVersion = req.URL.Query().Get("version")
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "https://example.test/appA/2.0.1",
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		default:
			t.Fatalf("unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	require.NoError(t, BackfillMissingInstalledPluginMetadata(context.Background(), config, fs, stripeServer.URL, stripeServer.URL))
	require.Equal(t, "2.0.1", requestedVersion)

	plugin, err := readLocalPluginMetadata(config, fs, "appA")
	require.NoError(t, err)
	require.Equal(t, "appA", plugin.Shortname)
	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())
}

func TestBackfillMissingInstalledPluginMetadataUsesCachedManifestBeforeNetwork(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.InstalledPlugins = []string{"appA"}

	pluginBinaryPath := filepath.Join(getPluginsDir(config), "appA", "2.0.1", "stripe-cli-app-a"+GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("installed"), 0755))

	manifestContent, err := os.ReadFile("./test_artifacts/plugins.toml")
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(fs, getCachedPluginManifestPath(config), manifestContent, os.ModePerm))

	var requestCount int
	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		requestCount++
		t.Fatalf("unexpected network request during cached-manifest backfill: %s", req.URL.String())
	}))
	defer stripeServer.Close()

	require.NoError(t, BackfillMissingInstalledPluginMetadata(context.Background(), config, fs, stripeServer.URL, stripeServer.URL))
	require.Equal(t, 0, requestCount)

	plugin, err := readLocalPluginMetadata(config, fs, "appA")
	require.NoError(t, err)
	require.Equal(t, "appA", plugin.Shortname)
	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())
}

func TestBackfillMissingInstalledPluginMetadataSkipsStaleCachedManifest(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.InstalledPlugins = []string{"appA"}

	pluginBinaryPath := filepath.Join(getPluginsDir(config), "appA", "2.0.1", "stripe-cli-app-a"+GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("installed"), 0755))

	staleManifest := []byte(fmt.Sprintf(`[[Plugin]]
  Shortname = "appA"
  Binary = "stripe-cli-app-a"
  MagicCookieValue = "APP-A-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "9.9.9"
    Sum = "abc123"
`, runtime.GOARCH, runtime.GOOS))
	require.NoError(t, afero.WriteFile(fs, getCachedPluginManifestPath(config), staleManifest, os.ModePerm))

	manifestContent, err := os.ReadFile("./test_artifacts/plugins.toml")
	require.NoError(t, err)

	var requestCount int
	var requestedVersion string
	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			requestCount++
			requestedVersion = req.URL.Query().Get("version")
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "https://example.test/appA/2.0.1",
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		default:
			t.Fatalf("unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	require.NoError(t, BackfillMissingInstalledPluginMetadata(context.Background(), config, fs, stripeServer.URL, stripeServer.URL))
	require.Equal(t, 1, requestCount)
	require.Equal(t, "2.0.1", requestedVersion)

	plugin, err := readLocalPluginMetadata(config, fs, "appA")
	require.NoError(t, err)
	require.NotNil(t, plugin.getReleaseForVersion("2.0.1"))
	require.Nil(t, plugin.getReleaseForVersion("9.9.9"))
}

func TestResolvePluginForInstallReturnsErrPluginNotFoundWhenLoggedIn(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	failingServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			res.WriteHeader(http.StatusNotFound)
			_, _ = res.Write([]byte(`{"error":{"message":"not found"}}`))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer failingServer.Close()

	_, err := ResolvePluginForInstall(context.Background(), config, fs, "nonexistent", "1.0.0", failingServer.URL, failingServer.URL)
	require.Error(t, err)

	var pluginNotFound *ErrPluginNotFound
	require.ErrorAs(t, err, &pluginNotFound)
	require.Equal(t, "nonexistent", pluginNotFound.Name)
}

func TestResolvePluginForInstallReturnsErrPluginNotFoundWhenNotLoggedIn(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""

	failingServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/ajax/stripecli/plugins_metadata":
			res.WriteHeader(http.StatusNotFound)
			_, _ = res.Write([]byte(`{"error":{"message":"not found"}}`))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer failingServer.Close()

	_, err := ResolvePluginForInstall(context.Background(), config, fs, "nonexistent", "1.0.0", failingServer.URL, failingServer.URL)
	require.Error(t, err)

	var pluginNotFound *ErrPluginNotFound
	require.ErrorAs(t, err, &pluginNotFound)
	require.Equal(t, "nonexistent", pluginNotFound.Name)
}

func TestResolvePluginForInstallSucceedsForGAPluginWhenNotLoggedIn(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/ajax/stripecli/plugins_metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "https://example.test/appA/2.0.1",
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer dashboardServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		t.Fatalf("anonymous resolution should not hit the API host: %s", req.URL.String())
	}))
	defer apiServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", apiServer.URL, dashboardServer.URL)
	require.NoError(t, err)
	require.NotNil(t, resolvedPlugin.Plugin)
	require.Equal(t, "appA", resolvedPlugin.Plugin.Shortname)
	require.Equal(t, "2.0.1", resolvedPlugin.Version)
}

func TestCheckLatestPluginVersionPrintsWhenUpgradeAvailable(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	plugin := Plugin{
		Shortname:        "myplugin",
		Binary:           "stripe-cli-myplugin",
		MagicCookieValue: "MY-COOKIE",
		Releases: []Release{
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.0.0", Sum: "abc123"},
		},
	}

	pluginBinaryPath := fmt.Sprintf("/plugins/myplugin/1.0.0/stripe-cli-myplugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("binary"), 0755))

	origResolver := checkLatestPluginVersionResolver
	checkLatestPluginVersionResolver = func(ctx context.Context, cfg cfgpkg.IConfig, fs afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
		return &ResolvedPluginVersion{
			Plugin: &Plugin{
				Shortname: "myplugin",
				Releases: []Release{
					{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.1.0", Sum: "abc123"},
				},
			},
			Version: "1.1.0",
		}, nil
	}
	defer func() { checkLatestPluginVersionResolver = origResolver }()

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(context.Background(), config, fs, plugin, stripe.DefaultAPIBaseURL, "")
	})

	require.Contains(t, output, "A newer version of the myplugin plugin is available")
	require.Contains(t, output, "v1.0.0")
	require.Contains(t, output, "v1.1.0")
	require.Contains(t, output, "stripe plugin upgrade myplugin")
}

func TestCheckLatestPluginVersionSilentWhenUpToDate(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	plugin := Plugin{
		Shortname:        "myplugin",
		Binary:           "stripe-cli-myplugin",
		MagicCookieValue: "MY-COOKIE",
		Releases: []Release{
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.1.0", Sum: "abc123"},
		},
	}

	pluginBinaryPath := fmt.Sprintf("/plugins/myplugin/1.1.0/stripe-cli-myplugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("binary"), 0755))

	origResolver := checkLatestPluginVersionResolver
	checkLatestPluginVersionResolver = func(ctx context.Context, cfg cfgpkg.IConfig, fs afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
		return &ResolvedPluginVersion{
			Plugin:  &plugin,
			Version: "1.1.0",
		}, nil
	}
	defer func() { checkLatestPluginVersionResolver = origResolver }()

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(context.Background(), config, fs, plugin, stripe.DefaultAPIBaseURL, "")
	})

	require.Empty(t, output)
}

func TestCheckLatestPluginVersionSilentWhenNoInstalledVersion(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	plugin := Plugin{
		Shortname:        "myplugin",
		Binary:           "stripe-cli-myplugin",
		MagicCookieValue: "MY-COOKIE",
	}

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(context.Background(), config, fs, plugin, stripe.DefaultAPIBaseURL, "")
	})

	require.Empty(t, output)
}

func TestCheckLatestPluginVersionSilentWhenLookupTimesOut(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	plugin := Plugin{
		Shortname:        "myplugin",
		Binary:           "stripe-cli-myplugin",
		MagicCookieValue: "MY-COOKIE",
		Releases: []Release{
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.0.0", Sum: "abc123"},
		},
	}

	pluginBinaryPath := fmt.Sprintf("/plugins/myplugin/1.0.0/stripe-cli-myplugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("binary"), 0755))

	origResolver := checkLatestPluginVersionResolver
	origTimeout := checkLatestPluginVersionTimeout
	checkLatestPluginVersionResolver = func(ctx context.Context, cfg cfgpkg.IConfig, fs afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	checkLatestPluginVersionTimeout = 10 * time.Millisecond
	defer func() {
		checkLatestPluginVersionResolver = origResolver
		checkLatestPluginVersionTimeout = origTimeout
	}()

	done := make(chan string, 1)
	go func() {
		done <- captureStderr(t, func() {
			CheckLatestPluginVersion(context.Background(), config, fs, plugin, stripe.DefaultAPIBaseURL, "")
		})
	}()

	select {
	case output := <-done:
		require.Empty(t, output)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("CheckLatestPluginVersion did not return before the timeout guard")
	}
}

// TestCheckLatestPluginVersionStillHintsUnderAnEnvironmentPluginsPath pins the one place
// the two plugins-path guards deliberately disagree. maybeAutoUpgrade refuses to install
// into a directory the user pointed the CLI at, whichever way they pointed it; the hint
// only goes quiet for a localdev build, which has no published release to be behind.
// Someone who relocated ordinary installs with the environment variable still wants to
// hear that an upgrade exists -- all the more so now that they will not get it silently.
func TestCheckLatestPluginVersionStillHintsUnderAnEnvironmentPluginsPath(t *testing.T) {
	origPluginsPath := PluginsPath
	origResolver := checkLatestPluginVersionResolver
	PluginsPath = ""
	t.Setenv("STRIPE_PLUGINS_PATH", "/somewhere/else")
	checkLatestPluginVersionResolver = func(ctx context.Context, cfg cfgpkg.IConfig, fs afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
		return &ResolvedPluginVersion{
			Plugin: &Plugin{
				Shortname: "myplugin",
				Releases: []Release{
					{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.1.0", Sum: "abc123"},
				},
			},
			Version: "1.1.0",
		}, nil
	}
	defer func() {
		PluginsPath = origPluginsPath
		checkLatestPluginVersionResolver = origResolver
	}()

	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	plugin := Plugin{
		Shortname:        "myplugin",
		Binary:           "stripe-cli-myplugin",
		MagicCookieValue: "MY-COOKIE",
	}

	pluginBinaryPath := fmt.Sprintf("/somewhere/else/myplugin/1.0.0/stripe-cli-myplugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("binary"), 0755))

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(context.Background(), config, fs, plugin, stripe.DefaultAPIBaseURL, "")
	})

	require.Contains(t, output, "A newer version of the myplugin plugin is available")
}

func TestGetPluginsDirOverrides(t *testing.T) {
	// TestConfig's config folder is "/", which is why every other test in this package
	// finds plugins at /plugins without arranging anything. Joined rather than written
	// out because this is the one case getPluginsDir builds a path for, and Windows
	// builds it with the other separator. The overrides below are handed back verbatim,
	// so they are the same string everywhere.
	defaultPluginsDir := filepath.Join("/", "plugins")

	tests := []struct {
		name           string
		pluginsPathEnv string
		pluginsPath    string
		want           string
	}{
		{
			name: "neither, so the CLI's own config folder",
			want: defaultPluginsDir,
		},
		{
			name:           "the environment variable",
			pluginsPathEnv: "/from/the/environment",
			want:           "/from/the/environment",
		},
		{
			name:        "a path compiled into a localdev build",
			pluginsPath: "/compiled/in",
			want:        "/compiled/in",
		},
		{
			// The order these have always resolved in, kept because a variable set for
			// one invocation is a narrower statement than one baked into a binary.
			name:           "both, so the environment variable",
			pluginsPathEnv: "/from/the/environment",
			pluginsPath:    "/compiled/in",
			want:           "/from/the/environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origPluginsPath := PluginsPath
			PluginsPath = tt.pluginsPath
			t.Setenv("STRIPE_PLUGINS_PATH", tt.pluginsPathEnv)
			defer func() { PluginsPath = origPluginsPath }()

			require.Equal(t, tt.want, getPluginsDir(&TestConfig{}))

			// What the auto-upgrade guard reads. Anything but the config folder is a
			// directory the CLI was pointed at and must not install over.
			require.Equal(t, tt.want != defaultPluginsDir, pluginsDirOverride() != "")
		})
	}
}

func TestCheckLatestPluginVersionSilentInDevMode(t *testing.T) {
	origPluginsPath := PluginsPath
	origResolver := checkLatestPluginVersionResolver
	PluginsPath = "/some/local/dev/path"
	checkLatestPluginVersionResolver = func(ctx context.Context, cfg cfgpkg.IConfig, fs afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
		return &ResolvedPluginVersion{
			Version: "1.1.0",
		}, nil
	}
	defer func() {
		PluginsPath = origPluginsPath
		checkLatestPluginVersionResolver = origResolver
	}()

	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	plugin := Plugin{
		Shortname:        "myplugin",
		Binary:           "stripe-cli-myplugin",
		MagicCookieValue: "MY-COOKIE",
	}

	pluginBinaryPath := fmt.Sprintf("/plugins/myplugin/1.0.0/stripe-cli-myplugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("binary"), 0755))

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(context.Background(), config, fs, plugin, stripe.DefaultAPIBaseURL, "")
	})

	require.Empty(t, output)
}

func TestCheckLatestPluginVersionSilentWhenPluginAutoUpdates(t *testing.T) {
	origPluginsPath := PluginsPath
	origUpdatesEnabled := pluginUpdatesEnabled
	origResolver := checkLatestPluginVersionResolver
	PluginsPath = ""
	// A plugins directory the CLI has not been pointed at, which is what makes deferring
	// to the pre-run check the right thing to do here. Pinned rather than assumed: the
	// suppression this asserts is now conditional on it, so a stray variable in the
	// environment running the tests would turn the whole test into its own opposite.
	t.Setenv("STRIPE_PLUGINS_PATH", "")

	var settingReads []string
	pluginUpdatesEnabled = func(pluginName string) bool {
		settingReads = append(settingReads, pluginName)
		return true
	}
	resolveCalls := 0
	checkLatestPluginVersionResolver = func(ctx context.Context, cfg cfgpkg.IConfig, fs afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
		resolveCalls++
		return &ResolvedPluginVersion{
			Plugin: &Plugin{
				Shortname: "myplugin",
				Releases: []Release{
					{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.1.0", Sum: "abc123"},
				},
			},
			Version: "1.1.0",
		}, nil
	}
	defer func() {
		PluginsPath = origPluginsPath
		pluginUpdatesEnabled = origUpdatesEnabled
		checkLatestPluginVersionResolver = origResolver
	}()

	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	plugin := Plugin{
		Shortname:        "myplugin",
		Binary:           "stripe-cli-myplugin",
		MagicCookieValue: "MY-COOKIE",
		Releases: []Release{
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.0.0", Sum: "abc123"},
		},
	}

	pluginBinaryPath := fmt.Sprintf("/plugins/myplugin/1.0.0/stripe-cli-myplugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("binary"), 0755))

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(context.Background(), config, fs, plugin, stripe.DefaultAPIBaseURL, "")
	})

	// This setup is exactly TestCheckLatestPluginVersionPrintsWhenUpgradeAvailable --
	// 1.0.0 installed, 1.1.0 offered -- so the setting is the only thing keeping it
	// quiet, and the hint text is not what is being suppressed here anyway.
	require.Equal(t, []string{"myplugin"}, settingReads)
	require.Empty(t, output)

	// The point is the request, not just the message. A hint here would put a lookup on
	// every command of an auto-updating plugin, which is the cost
	// autoUpgradeCheckInterval exists to keep maybeAutoUpgrade from imposing.
	require.Zero(t, resolveCalls)
}

// TestCheckLatestPluginVersionHintsWhenAutoUpgradeWillNotRun covers the one state where
// both halves of the feature could go quiet at once: auto-update is on for the plugin, so
// the hint would hand the job to the pre-run check, while the plugins directory is
// overridden, so that check refuses it outright. Deferring to something that never runs
// leaves the plugin silently out of date -- the single outcome neither guard is willing to
// own, and the reason the suppression above asks whether the upgrade can happen at all.
//
// Distinct from the throttle, which is also a decline: that one is for this invocation and
// the next one may well upgrade, so staying quiet costs nothing but a few hours. An
// overridden directory is refused on every invocation, forever.
func TestCheckLatestPluginVersionHintsWhenAutoUpgradeWillNotRun(t *testing.T) {
	origPluginsPath := PluginsPath
	origUpdatesEnabled := pluginUpdatesEnabled
	origResolver := checkLatestPluginVersionResolver
	PluginsPath = ""
	t.Setenv("STRIPE_PLUGINS_PATH", "/somewhere/else")

	var settingReads []string
	pluginUpdatesEnabled = func(pluginName string) bool {
		settingReads = append(settingReads, pluginName)
		return true
	}
	checkLatestPluginVersionResolver = func(ctx context.Context, cfg cfgpkg.IConfig, fs afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
		return &ResolvedPluginVersion{
			Plugin: &Plugin{
				Shortname: "myplugin",
				Releases: []Release{
					{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.1.0", Sum: "abc123"},
				},
			},
			Version: "1.1.0",
		}, nil
	}
	defer func() {
		PluginsPath = origPluginsPath
		pluginUpdatesEnabled = origUpdatesEnabled
		checkLatestPluginVersionResolver = origResolver
	}()

	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	plugin := Plugin{
		Shortname:        "myplugin",
		Binary:           "stripe-cli-myplugin",
		MagicCookieValue: "MY-COOKIE",
		Releases: []Release{
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.0.0", Sum: "abc123"},
		},
	}

	pluginBinaryPath := fmt.Sprintf("/somewhere/else/myplugin/1.0.0/stripe-cli-myplugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("binary"), 0755))

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(context.Background(), config, fs, plugin, stripe.DefaultAPIBaseURL, "")
	})

	require.Contains(t, output, "A newer version of the myplugin plugin is available")

	// The setting is not read at all. Under an overridden directory it has nothing left to
	// decide, and asserting that rules out passing for the neighboring reason -- a hint
	// printed because the setting happened to be off rather than because the override
	// took precedence over it.
	require.Empty(t, settingReads)
}

func TestIsPluginCommand(t *testing.T) {
	pluginCmd := &cobra.Command{
		Annotations: map[string]string{"scope": "plugin"},
	}

	notPluginCmd := &cobra.Command{}

	require.True(t, IsPluginCommand(pluginCmd))
	require.False(t, IsPluginCommand(notPluginCmd))
}

func TestAddPluginToListSortsBySemver(t *testing.T) {
	pluginList := &PluginList{
		Plugins: []Plugin{
			{
				Shortname:        "test-plugin",
				MagicCookieValue: "TEST-COOKIE-123",
				Releases: []Release{
					{Version: "1.0.0", OS: "darwin", Arch: "amd64"},
					{Version: "1.2.0", OS: "darwin", Arch: "amd64"},
				},
			},
		},
	}

	newPlugin := Plugin{
		Shortname:        "test-plugin",
		MagicCookieValue: "TEST-COOKIE-123",
		Releases: []Release{
			{Version: "1.10.0", OS: "darwin", Arch: "amd64"},
			{Version: "1.9.0", OS: "darwin", Arch: "amd64"},
			{Version: "2.0.0", OS: "darwin", Arch: "amd64"},
			{Version: "1.0.1", OS: "darwin", Arch: "amd64"},
		},
	}

	addPluginToList(pluginList, newPlugin)

	require.Len(t, pluginList.Plugins, 1)
	require.Len(t, pluginList.Plugins[0].Releases, 6)

	expectedOrder := []string{"1.0.0", "1.0.1", "1.2.0", "1.9.0", "1.10.0", "2.0.0"}
	for i, release := range pluginList.Plugins[0].Releases {
		require.Equal(t, expectedOrder[i], release.Version)
	}
}

func testListEndpointResponseJSON() []byte {
	return []byte(fmt.Sprintf(`{
  "plugins": [
    {
      "shortname": "apps",
      "shortdesc": "Build and manage Stripe Apps",
      "binary": "stripe-cli-apps",
      "commands": [
        {
          "name": "create",
          "desc": "Create an app"
        }
      ],
      "releases": [
        {
          "os": "%s",
          "arch": "%s",
          "version": "1.12.0"
        }
      ],
      "binary_url": null
    }
  ]
}`, runtime.GOOS, runtime.GOARCH))
}

func TestResolveInstallBaseURLs(t *testing.T) {
	tests := []struct {
		name             string
		apiBaseURL       string
		dashboardBaseURL string
		wantAPI          string
		wantDashboard    string
	}{
		{
			// What Run is handed when the user passed no base URL flags at all, which is
			// the usual case. An empty pair has to become a real host, not stay empty.
			name:          "both empty fall back to this CLI's defaults",
			wantAPI:       "https://api.stripe.com",
			wantDashboard: "https://dashboard.stripe.com",
		},
		{
			// The dashboard has to follow the API base URL. If it didn't, an --api-base
			// pointed at QA would pair with production's dashboard.
			name:          "dashboard follows an overridden api base",
			apiBaseURL:    "https://qa-api.stripe.com",
			wantAPI:       "https://qa-api.stripe.com",
			wantDashboard: "https://qa-dashboard.stripe.com",
		},
		{
			name:             "dashboard override stands on its own",
			dashboardBaseURL: "https://qa-dashboard.stripe.com",
			wantAPI:          "https://api.stripe.com",
			wantDashboard:    "https://qa-dashboard.stripe.com",
		},
		{
			name:             "both overrides pass through untouched",
			apiBaseURL:       "https://qa-api.stripe.com",
			dashboardBaseURL: "https://custom-dashboard.stripe.com",
			wantAPI:          "https://qa-api.stripe.com",
			wantDashboard:    "https://custom-dashboard.stripe.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAPI, gotDashboard := resolveInstallBaseURLs(tt.apiBaseURL, tt.dashboardBaseURL)

			require.Equal(t, tt.wantAPI, gotAPI)
			require.Equal(t, tt.wantDashboard, gotDashboard)
		})
	}
}

// The cheap half of cancellation: a context that is already done should not reach
// the network at all.
func TestFetchRemoteResourceDoesNotRequestWithCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
	}))
	defer server.Close()

	_, err := FetchRemoteResource(ctx, server.URL)

	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, requestCount.Load())
}

// The half that matters for Ctrl+C. A plugin binary is large enough that a wait
// worth abandoning is a wait that has already gotten past the response headers, so
// this cancels mid-body and asserts the transfer is actually torn down rather than
// running to completion behind an error return.
func TestFetchRemoteResourceCancelsDownloadInFlight(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tornDown := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Headers and a first chunk, so the client is inside the body read rather than
		// still waiting to hear back.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("the first bytes of a plugin binary"))
		w.(http.Flusher).Flush()

		// Stands in for the rest of the download never arriving. Bounded so a
		// regression cannot wedge httptest's Close, which waits on its handlers.
		cancel()
		select {
		case <-r.Context().Done():
			close(tornDown)
		case <-time.After(cancellationTestTimeout):
		}
	}))
	defer server.Close()

	// Off the test goroutine, and bounded, because the whole point of the assertion
	// is that this call returns at all. Waiting on it directly would turn a
	// regression into a hung package instead of a failed test.
	fetched := make(chan error, 1)
	go func() {
		_, err := FetchRemoteResource(ctx, server.URL)
		fetched <- err
	}()

	select {
	case err := <-fetched:
		require.Error(t, err)
	case <-time.After(cancellationTestTimeout):
		t.Fatal("canceling the context did not stop the download")
	}

	select {
	case <-tornDown:
	case <-time.After(cancellationTestTimeout):
		t.Fatal("canceling the context left the connection open")
	}
}

// Long enough that a loaded CI machine will not trip it, short enough that a
// regression reports itself rather than running out the package's test timeout.
const cancellationTestTimeout = 15 * time.Second

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	fn()

	require.NoError(t, w.Close())
	os.Stderr = orig

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}
