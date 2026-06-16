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
	"testing"

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
	require.Equal(t, "20", pluginList.Plugins[0].Releases[0].Runtime["node"])
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
	require.Len(t, plugin.Releases, 4)
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
				Runtime: map[string]string{"node": "20"},
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
	require.Equal(t, "20", release.Runtime["node"])
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
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "https://example.test/appA/2.0.1",
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		case "/v1/stripecli/get-plugin-url":
			t.Fatalf("install resolution should not fall back to /v1/stripecli/get-plugin-url when anonymous metadata is available")
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
		case "/v1/stripecli/get-plugin-url":
			t.Fatalf("install resolution should not hit /v1/stripecli/get-plugin-url when cached metadata is available")
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
		case "/v1/stripecli/get-plugin-url":
			t.Fatalf("upgrade resolution should not fall back to /v1/stripecli/get-plugin-url when plugin metadata is available")
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
		case "/v1/stripecli/get-plugin-url":
			t.Fatalf("upgrade resolution should not fall back to /v1/stripecli/get-plugin-url when anonymous plugin metadata is available")
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

func TestIsPluginCommand(t *testing.T) {
	pluginCmd := &cobra.Command{
		Annotations: map[string]string{"scope": "plugin"},
	}

	notPluginCmd := &cobra.Command{}

	require.True(t, IsPluginCommand(pluginCmd))
	require.False(t, IsPluginCommand(notPluginCmd))
}

func TestIsValidNodeLTSVersion(t *testing.T) {
	validVersions := []string{"18", "20", "22", "24", "26"}
	for _, version := range validVersions {
		require.True(t, isValidNodeLTSVersion(version), "Expected %s to be valid LTS version", version)
	}

	invalidVersions := []string{"10", "11", "12", "13", "14", "15", "16", "17", "19", "21", "23", "25"}
	for _, version := range invalidVersions {
		require.False(t, isValidNodeLTSVersion(version), "Expected %s to be invalid LTS version", version)
	}

	invalidFormats := []string{"", "abc", "20.0", "v20", "20.0.0", "node20"}
	for _, version := range invalidFormats {
		require.False(t, isValidNodeLTSVersion(version), "Expected %s to be invalid format", version)
	}
}

func TestValidateRuntimeVersionsValid(t *testing.T) {
	pluginList := &PluginList{
		Plugins: []Plugin{
			{
				Shortname: "test-plugin",
				Releases: []Release{
					{
						Version: "1.0.0",
						Runtime: map[string]string{"node": "18"},
					},
					{
						Version: "1.1.0",
						Runtime: map[string]string{"node": "20"},
					},
					{
						Version: "2.0.0",
						Runtime: map[string]string{"node": "24"},
					},
				},
			},
		},
	}

	require.NoError(t, validateRuntimeVersions(pluginList))
}

func TestValidateRuntimeVersionsInvalidNonLTS(t *testing.T) {
	pluginList := &PluginList{
		Plugins: []Plugin{
			{
				Shortname: "test-plugin",
				Releases: []Release{
					{
						Version: "1.0.0",
						Runtime: map[string]string{"node": "19"},
					},
				},
			},
		},
	}

	err := validateRuntimeVersions(pluginList)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid Node.js version '19'")
	require.ErrorContains(t, err, "test-plugin")
	require.ErrorContains(t, err, "Only LTS major versions are allowed")
}

func TestValidateRuntimeVersionsInvalidOldVersion(t *testing.T) {
	pluginList := &PluginList{
		Plugins: []Plugin{
			{
				Shortname: "test-plugin",
				Releases: []Release{
					{
						Version: "1.0.0",
						Runtime: map[string]string{"node": "10"},
					},
				},
			},
		},
	}

	err := validateRuntimeVersions(pluginList)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid Node.js version '10'")
}

func TestValidateRuntimeVersionsNoRuntime(t *testing.T) {
	pluginList := &PluginList{
		Plugins: []Plugin{
			{
				Shortname: "test-plugin",
				Releases: []Release{
					{
						Version: "1.0.0",
					},
				},
			},
		},
	}

	require.NoError(t, validateRuntimeVersions(pluginList))
}

func TestValidatePluginManifestWithInvalidRuntime(t *testing.T) {
	invalidManifest := `
[[Plugin]]
  Shortname = "test-app"
  Binary = "stripe-cli-test-app"
  MagicCookieValue = "TEST-COOKIE"

  [[Plugin.Release]]
    Arch = "amd64"
    OS = "darwin"
    Version = "1.0.0"
    Sum = "abcdef1234567890"
    Runtime = {node = "17"}
`

	_, err := validatePluginManifest([]byte(invalidManifest))
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid Node.js version '17'")
}

func TestValidatePluginManifestWithValidRuntime(t *testing.T) {
	validManifest := `
[[Plugin]]
  Shortname = "test-app"
  Binary = "stripe-cli-test-app"
  MagicCookieValue = "TEST-COOKIE"

  [[Plugin.Release]]
    Arch = "amd64"
    OS = "darwin"
    Version = "1.0.0"
    Sum = "abcdef1234567890"
    Runtime = {node = "24"}
`

	pluginList, err := validatePluginManifest([]byte(validManifest))
	require.NoError(t, err)
	require.NotNil(t, pluginList)
	require.Len(t, pluginList.Plugins, 1)
	require.Equal(t, "24", pluginList.Plugins[0].Releases[0].Runtime["node"])
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
          "version": "1.12.0",
          "runtime": {
            "node": "20"
          }
        }
      ],
      "binary_url": null
    }
  ]
}`, runtime.GOOS, runtime.GOARCH))
}

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
