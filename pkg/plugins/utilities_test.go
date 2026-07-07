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

	"github.com/BurntSushi/toml"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// CustomTestConfig is a test config that allows overriding the config folder path
type CustomTestConfig struct {
	TestConfig
	customConfigPath string
}

// GetConfigFolder overrides the TestConfig method to return a custom path
func (c *CustomTestConfig) GetConfigFolder(xdgPath string) string {
	return c.customConfigPath
}

func TestGetPluginList(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()

	pluginList, err := GetPluginList(context.Background(), config, fs)

	require.Nil(t, err)
	require.Equal(t, 3, len(pluginList.Plugins))
	plugin := pluginList.Plugins[0]
	require.Equal(t, "appA", plugin.Shortname)
	require.Equal(t, "stripe-cli-app-a", plugin.Binary)
	require.Equal(t, "0337A75A-C3C4-4DCF-A9EF-E7A144E5A291", plugin.MagicCookieValue)

	require.Equal(t, 12, len(plugin.Releases))
	release := plugin.Releases[0]
	require.Equal(t, "amd64", release.Arch)
	require.Equal(t, "darwin", release.OS)
	require.Equal(t, "0.0.1", release.Version)
	require.Equal(t, "125653c37803a51a048f6687f7f66d511be614f675f199cd6c71928b74875238", release.Sum)
}

func TestGetPluginListIgnoresLocalPluginMetadataOutsideManifest(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "sample-plugin",
		Shortdesc:        "Sample plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
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

	pluginList, err := GetPluginList(context.Background(), config, fs)
	require.NoError(t, err)
	require.Len(t, pluginList.Plugins, 3)

	_, err = findPlugin(pluginList, "docs")
	require.Error(t, err)
	_, err = findPlugin(pluginList, "appA")
	require.NoError(t, err)
	_, err = findPlugin(pluginList, "appB")
	require.NoError(t, err)
	_, err = findPlugin(pluginList, "appC")
	require.NoError(t, err)
}

func TestGetPluginListUsesManifestMetadataForOverlappingPlugin(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "appA",
		Shortdesc:        "Locally cached App A",
		Binary:           "stripe-cli-local-app-a",
		MagicCookieValue: "0337A75A-C3C4-4DCF-A9EF-E7A144E5A291",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "2.0.1",
				Sum:     "abc123",
				Runtime: map[string]string{"node": "20"},
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	pluginList, err := GetPluginList(context.Background(), config, fs)
	require.NoError(t, err)
	require.Len(t, pluginList.Plugins, 3)

	plugin, err := findPlugin(pluginList, "appA")
	require.NoError(t, err)
	require.Equal(t, "stripe-cli-app-a", plugin.Binary)
	require.Empty(t, plugin.Shortdesc)
	require.Len(t, plugin.Releases, 12)

	release := plugin.getReleaseForVersion("2.0.1")
	require.NotNil(t, release)
	require.Empty(t, release.Runtime)
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

func TestLookUpPlugin(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, err := LookUpPlugin(context.Background(), config, fs, "appB")
	require.Nil(t, err)
	require.Equal(t, "appB", plugin.Shortname)
	require.Equal(t, "stripe-cli-app-b", plugin.Binary)
	require.Equal(t, "FDBE6FB9-A149-44BD-9639-4D33D8B594E8", plugin.MagicCookieValue)
	require.Equal(t, 4, len(plugin.Releases))
}

func TestLookUpPluginUsesLocalMetadataWhenManifestMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "local-plugin",
		Binary:           "stripe-cli-local-plugin",
		MagicCookieValue: "LOCAL-PLUGIN-COOKIE",
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

	plugin, err := LookUpPlugin(context.Background(), config, fs, "local-plugin")
	require.NoError(t, err)
	require.Equal(t, localPlugin, plugin)
}

func TestGetInstalledPluginNamesIncludesLocalMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InstalledPlugins = []string{"projects"}

	localPlugin := Plugin{
		Shortname:        "sample-plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
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
	require.Equal(t, []string{"projects", "sample-plugin"}, pluginNames)
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
		Shortname:        "sample-plugin",
		Shortdesc:        "Sample plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "hello",
				Desc: "Say hello",
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
	require.Equal(t, []string{"sample-plugin"}, config.GetInstalledPlugins())

	cachedPlugin, err := readLocalPluginMetadata(config, fs, "sample-plugin")
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
		Shortname:        "sample-plugin",
		Shortdesc:        "Sample plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
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

	metadataExists, err := afero.Exists(fs, getLocalPluginMetadataPath(config, "docs"))
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
		Shortname:        "sample-plugin",
		Shortdesc:        "Existing docs plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
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
		Shortname:        "sample-plugin",
		Shortdesc:        "Updated docs plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
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

	cachedPlugin, err := readLocalPluginMetadata(config, fs, "sample-plugin")
	require.NoError(t, err)
	require.Equal(t, existingPlugin, cachedPlugin)
	require.Empty(t, config.GetInstalledPlugins())
}

func TestLookUpPluginInManifestIgnoresLocalMetadata(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "appA",
		Binary:           "stripe-cli-local-app-a",
		MagicCookieValue: "LOCAL-APP-A-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "9.9.9",
				Sum:     "abc123",
			},
		},
	}

	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	plugin, err := LookUpPluginInManifest(context.Background(), config, fs, "appA")
	require.NoError(t, err)
	require.Equal(t, "stripe-cli-app-a", plugin.Binary)
	require.Equal(t, "0337A75A-C3C4-4DCF-A9EF-E7A144E5A291", plugin.MagicCookieValue)
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
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "generate", "1.0.0", stripeServer.URL, stripeServer.URL)
	require.NoError(t, err)
	plugin := resolvedPlugin.Plugin
	version := resolvedPlugin.Version
	require.Equal(t, "1.0.0", version)
	require.Len(t, plugin.Commands, 1)
	require.Equal(t, "create", plugin.Commands[0].Name)
	release := plugin.getReleaseForVersion("1.0.0")
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
			res.Write(body)
		case "/v1/stripecli/get-plugin-url":
			t.Fatalf("install resolution should not fall back to /v1/stripecli/get-plugin-url when anonymous metadata is available")
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer dashboardServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", apiServer.URL, dashboardServer.URL)
	require.NoError(t, err)
	plugin := resolvedPlugin.Plugin
	version := resolvedPlugin.Version
	require.NotNil(t, plugin)
	require.Equal(t, "appA", plugin.Shortname)
	require.Equal(t, "2.0.1", version)
	require.Equal(t, "https://example.test/appA/2.0.1", resolvedPlugin.BinaryURL)
	require.Equal(t, 1, metadataLookups)

	_, err = fs.Stat("/plugins.toml")
	require.True(t, os.IsNotExist(err))
}

func TestResolvePluginForInstallFallsBackToManifestLookupWhenAnonymousMetadataFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	originalPluginData := requests.DefaultPluginData
	requests.DefaultPluginData = requests.PluginData{
		PluginBaseURL:       testServers.ArtifactoryServer.URL,
		AdditionalManifests: nil,
	}
	defer func() {
		requests.DefaultPluginData = originalPluginData
	}()

	failingServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/ajax/stripecli/plugins_metadata":
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(`{"error":{"message":"boom"}}`))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer failingServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", failingServer.URL, failingServer.URL)
	require.NoError(t, err)
	plugin := resolvedPlugin.Plugin
	version := resolvedPlugin.Version
	require.NotNil(t, plugin)
	require.Equal(t, "appA", plugin.Shortname)
	require.Equal(t, "2.0.1", version)
	require.Empty(t, resolvedPlugin.BinaryURL)

	_, err = fs.Stat("/plugins.toml")
	require.NoError(t, err)
}

func TestResolvePluginForUpgradeUsesMetadataEndpointWhenAvailable(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	localPlugin := Plugin{
		Shortname:        "sample-plugin",
		Shortdesc:        "Sample plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "hello",
				Desc: "Say hello",
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
				BinaryURL: "https://example.test/sample-plugin/latest",
				PluginManifest: fmt.Sprintf(`[[Plugin]]
  Shortname = "sample-plugin"
  Shortdesc = "Sample plugin"
  Binary = "stripe-cli-sample-plugin"
  MagicCookieValue = "SAMPLE-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "0.1.26"
    Sum = "def456"
`, runtime.GOARCH, runtime.GOOS),
			})
			require.NoError(t, err)
			res.Write(body)
		case "/v1/stripecli/get-plugin-url":
			t.Fatalf("upgrade resolution should not fall back to /v1/stripecli/get-plugin-url when plugin metadata is available")
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	resolvedPlugin, err := ResolvePluginForUpgrade(context.Background(), config, fs, "sample-plugin", stripeServer.URL, stripeServer.URL)
	require.NoError(t, err)
	plugin := resolvedPlugin.Plugin
	require.Equal(t, "0.1.26", plugin.LookUpLatestVersion())
	require.Equal(t, "0.1.26", resolvedPlugin.Version)
	require.Equal(t, "https://example.test/sample-plugin/latest", resolvedPlugin.BinaryURL)
	require.Len(t, plugin.Commands, 1)
	require.Equal(t, "hello", plugin.Commands[0].Name)
	require.Equal(t, 1, metadataLookups)

	_, err = fs.Stat("/plugins.toml")
	require.True(t, os.IsNotExist(err))
}

func TestResolvePluginForUpgradeUsesAnonymousMetadataEndpointWhenAPIKeyUnavailable(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""

	localPlugin := Plugin{
		Shortname:        "sample-plugin",
		Shortdesc:        "Sample plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "hello",
				Desc: "Say hello",
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
				BinaryURL: "https://example.test/sample-plugin/latest",
				PluginManifest: fmt.Sprintf(`[[Plugin]]
  Shortname = "sample-plugin"
  Shortdesc = "Sample plugin"
  Binary = "stripe-cli-sample-plugin"
  MagicCookieValue = "SAMPLE-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "0.1.26"
    Sum = "def456"
`, runtime.GOARCH, runtime.GOOS),
			})
			require.NoError(t, err)
			res.Write(body)
		case "/v1/stripecli/get-plugin-url":
			t.Fatalf("upgrade resolution should not fall back to /v1/stripecli/get-plugin-url when anonymous plugin metadata is available")
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer dashboardServer.Close()

	resolvedPlugin, err := ResolvePluginForUpgrade(context.Background(), config, fs, "sample-plugin", apiServer.URL, dashboardServer.URL)
	require.NoError(t, err)
	plugin := resolvedPlugin.Plugin
	require.Equal(t, "0.1.26", plugin.LookUpLatestVersion())
	require.Equal(t, "0.1.26", resolvedPlugin.Version)
	require.Equal(t, "https://example.test/sample-plugin/latest", resolvedPlugin.BinaryURL)
	require.Len(t, plugin.Commands, 1)
	require.Equal(t, "hello", plugin.Commands[0].Name)
	require.Equal(t, 1, metadataLookups)

	_, err = fs.Stat("/plugins.toml")
	require.True(t, os.IsNotExist(err))
}

func TestResolvePluginForUpgradeFallsBackToCachedMetadataWhenEndpointFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	localPlugin := Plugin{
		Shortname:        "sample-plugin",
		Shortdesc:        "Sample plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "hello",
				Desc: "Say hello",
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

	resolvedPlugin, err := ResolvePluginForUpgrade(context.Background(), config, fs, "sample-plugin", failingServer.URL, failingServer.URL)
	require.NoError(t, err)
	plugin := resolvedPlugin.Plugin
	require.Equal(t, localPlugin, *plugin)
	require.Equal(t, "0.1.25", resolvedPlugin.Version)
	require.Empty(t, resolvedPlugin.BinaryURL)
}

func TestResolveCachedPluginForUpgradeUsesLocalMetadataWhenManifestMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "sample-plugin",
		Shortdesc:        "Sample plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "hello",
				Desc: "Say hello",
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

	plugin, err := resolveCachedPluginForUpgrade(config, fs, "sample-plugin")
	require.NoError(t, err)
	require.Equal(t, localPlugin, *plugin)
}

func TestResolveCachedPluginForUpgradePrefersNewerManifestVersion(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "sample-plugin",
		Shortdesc:        "Sample plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "hello",
				Desc: "Say hello",
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

	manifestContent := fmt.Sprintf(`[[Plugin]]
  Shortname = "sample-plugin"
  Shortdesc = "Sample plugin"
  Binary = "stripe-cli-sample-plugin"
  MagicCookieValue = "SAMPLE-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "0.1.26"
    Sum = "def456"
`, runtime.GOARCH, runtime.GOOS)
	require.NoError(t, afero.WriteFile(fs, "/plugins.toml", []byte(manifestContent), os.ModePerm))

	plugin, err := resolveCachedPluginForUpgrade(config, fs, "sample-plugin")
	require.NoError(t, err)
	require.Equal(t, "0.1.26", plugin.LookUpLatestVersion())
	require.Len(t, plugin.Commands, 1)
	require.Equal(t, "hello", plugin.Commands[0].Name)
}

func TestRefreshPluginManifest(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	updatedManifestContent, _ := os.ReadFile("./test_artifacts/plugins_updated.toml")
	testServers := setUpServers(t, updatedManifestContent, nil)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.Nil(t, err)

	// We expect the /plugins.toml file in the test fs is updated
	pluginManifestContent, err := afero.ReadFile(fs, "/plugins.toml")
	require.Nil(t, err)

	actualPluginList := PluginList{}
	err = toml.Unmarshal(pluginManifestContent, &actualPluginList)
	require.Nil(t, err)

	expectedPluginList := PluginList{}
	err = toml.Unmarshal(updatedManifestContent, &expectedPluginList)
	require.Nil(t, err)

	require.Equal(t, expectedPluginList, actualPluginList)
}

func TestRefreshPluginManifestMergesAdditionalManifest(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	mergedManifestContent, _ := os.ReadFile("./test_artifacts/plugins-merged-foo-1.toml")

	fooManifest, _ := os.ReadFile("./test_artifacts/plugins-foo-1.toml")
	additionalManifests := map[string][]byte{
		"plugins-foo-1.toml": fooManifest,
	}

	testServers := setUpServers(t, manifestContent, additionalManifests)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.Nil(t, err)

	// We expect the /plugins.toml file in the test fs is updated
	pluginManifestContent, err := afero.ReadFile(fs, "/plugins.toml")
	require.Nil(t, err)

	actualPluginList := PluginList{}
	err = toml.Unmarshal(pluginManifestContent, &actualPluginList)
	require.Nil(t, err)

	expectedPluginList := PluginList{}
	err = toml.Unmarshal(mergedManifestContent, &expectedPluginList)
	require.Nil(t, err)

	require.Equal(t, expectedPluginList, actualPluginList)
}

func TestRefreshPluginManifestMergesExisitingPlugin(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins-2.toml")
	mergedManifestContent, _ := os.ReadFile("./test_artifacts/plugins-2-merged-foo-1.toml")

	fooManifest, _ := os.ReadFile("./test_artifacts/plugins-foo-1.toml")
	additionalManifests := map[string][]byte{
		"plugins-foo-1.toml": fooManifest,
	}

	testServers := setUpServers(t, manifestContent, additionalManifests)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.Nil(t, err)

	// We expect the /plugins.toml file in the test fs is updated
	pluginManifestContent, err := afero.ReadFile(fs, "/plugins.toml")
	require.Nil(t, err)

	actualPluginList := PluginList{}
	err = toml.Unmarshal(pluginManifestContent, &actualPluginList)
	require.Nil(t, err)

	expectedPluginList := PluginList{}
	err = toml.Unmarshal(mergedManifestContent, &expectedPluginList)
	require.Nil(t, err)

	require.Equal(t, expectedPluginList, actualPluginList)
}

func TestRefreshPluginManifestSortsPluginReleases(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins-3.toml")
	mergedManifestContent, _ := os.ReadFile("./test_artifacts/plugins-3-merged-foo-1.toml")

	fooManifest, _ := os.ReadFile("./test_artifacts/plugins-foo-1.toml")
	additionalManifests := map[string][]byte{
		"plugins-foo-1.toml": fooManifest,
	}

	testServers := setUpServers(t, manifestContent, additionalManifests)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.Nil(t, err)

	// We expect the /plugins.toml file in the test fs is updated
	pluginManifestContent, err := afero.ReadFile(fs, "/plugins.toml")
	require.Nil(t, err)

	actualPluginList := PluginList{}
	err = toml.Unmarshal(pluginManifestContent, &actualPluginList)
	require.Nil(t, err)

	expectedPluginList := PluginList{}
	err = toml.Unmarshal(mergedManifestContent, &expectedPluginList)
	require.Nil(t, err)

	require.Equal(t, expectedPluginList, actualPluginList)
}

func TestRefreshPluginManifestFailsInvalidManifest(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	emptyManifestContent := []byte{}
	testServers := setUpServers(t, emptyManifestContent, nil)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.NotNil(t, err)
	require.ErrorContains(t, err, "received an empty plugin manifest")
	// We expect the /plugins.toml file in the test fs has NOT been updated
	pluginManifestContent, err := afero.ReadFile(fs, "/plugins.toml")
	require.Nil(t, err)
	require.NotEqual(t, emptyManifestContent, pluginManifestContent)
}

func TestIsPluginCommand(t *testing.T) {
	pluginCmd := &cobra.Command{
		Annotations: map[string]string{"scope": "plugin"},
	}

	notPluginCmd := &cobra.Command{}

	require.True(t, IsPluginCommand(pluginCmd))
	require.False(t, IsPluginCommand(notPluginCmd))
}

func TestRefreshPluginManifestSucceedsIfNoAPIKey(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""
	testServers := setUpServers(t, nil, nil)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.Nil(t, err)
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

	err := validateRuntimeVersions(pluginList)
	require.Nil(t, err)
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
	require.NotNil(t, err)
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
	require.NotNil(t, err)
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

	err := validateRuntimeVersions(pluginList)
	require.Nil(t, err)
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
	require.NotNil(t, err)
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
	require.Nil(t, err)
	require.NotNil(t, pluginList)
	require.Equal(t, 1, len(pluginList.Plugins))
	require.Equal(t, "24", pluginList.Plugins[0].Releases[0].Runtime["node"])
}

func TestRefreshPluginSucceedsIfAdditionalManifestNotFound(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.String() {
		case "/plugins.toml":
			res.Write(manifestContent)
		case "/plugins-nonexistent.toml":
			res.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer func() { artifactoryServer.Close() }()

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case "/v1/stripecli/get-plugin-url":
			pd := requests.PluginData{
				PluginBaseURL:       artifactoryServer.URL,
				AdditionalManifests: []string{"plugins-nonexistent.toml"},
			}
			body, err := json.Marshal(pd)
			if err != nil {
				t.Error(err)
			}
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	err := RefreshPluginManifest(context.Background(), config, fs, stripeServer.URL)
	require.Nil(t, err)
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

	// Add a plugin with versions that would be sorted incorrectly by string comparison
	newPlugin := Plugin{
		Shortname:        "test-plugin",
		MagicCookieValue: "TEST-COOKIE-123",
		Releases: []Release{
			{Version: "1.10.0", OS: "darwin", Arch: "amd64"}, // String comparison would put this before 1.2.0
			{Version: "1.9.0", OS: "darwin", Arch: "amd64"},  // Should come after 1.2.0 but before 1.10.0
			{Version: "2.0.0", OS: "darwin", Arch: "amd64"},
			{Version: "1.0.1", OS: "darwin", Arch: "amd64"}, // Should come after 1.0.0 but before 1.2.0
		},
	}

	addPluginToList(pluginList, newPlugin)

	// Verify the plugin was merged (not added as a new one)
	require.Equal(t, 1, len(pluginList.Plugins))

	// Verify all releases are present
	require.Equal(t, 6, len(pluginList.Plugins[0].Releases))

	// Verify they are sorted by semver (not by string comparison)
	expectedOrder := []string{"1.0.0", "1.0.1", "1.2.0", "1.9.0", "1.10.0", "2.0.0"}
	for i, release := range pluginList.Plugins[0].Releases {
		require.Equal(t, expectedOrder[i], release.Version,
			"Expected release %d to be version %s, but got %s", i, expectedOrder[i], release.Version)
	}
}

func TestRefreshPluginManifestCreatesConfigDirectory(t *testing.T) {
	// Create a test config that uses a non-root directory
	testConfigPath := "/test-config-dir"
	customConfig := &CustomTestConfig{
		customConfigPath: testConfigPath,
	}
	customConfig.InitConfig()

	// Create a fresh filesystem without the config directory
	fs := afero.NewMemMapFs()

	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer func() { testServers.CloseAll() }()

	// Verify the config directory doesn't exist yet
	exists, err := afero.DirExists(fs, testConfigPath)
	require.Nil(t, err)
	require.False(t, exists)

	// Refresh the manifest which should create the config directory
	err = RefreshPluginManifest(context.Background(), customConfig, fs, testServers.StripeServer.URL)
	require.Nil(t, err)

	// Verify the config directory now exists
	exists, err = afero.DirExists(fs, testConfigPath)
	require.Nil(t, err)
	require.True(t, exists)

	// Verify the plugins.toml file was created
	pluginManifestPath := testConfigPath + "/plugins.toml"
	exists, err = afero.Exists(fs, pluginManifestPath)
	require.Nil(t, err)
	require.True(t, exists)
}

func TestRefreshPluginManifestRefusesSymlink(t *testing.T) {
	tempDir := t.TempDir()
	customConfig := &CustomTestConfig{
		customConfigPath: tempDir,
	}
	customConfig.InitConfig()

	fs := afero.NewOsFs()
	manifestContent, err := os.ReadFile("./test_artifacts/plugins_updated.toml")
	require.NoError(t, err)

	testServers := setUpServers(t, manifestContent, nil)
	defer func() { testServers.CloseAll() }()

	victimFile := filepath.Join(tempDir, "victim.toml")
	require.NoError(t, os.WriteFile(victimFile, []byte("original = true\n"), 0o644))

	pluginManifestPath := filepath.Join(tempDir, "plugins.toml")
	require.NoError(t, os.Symlink(victimFile, pluginManifestPath))

	err = RefreshPluginManifest(context.Background(), customConfig, fs, testServers.StripeServer.URL)
	require.ErrorContains(t, err, "symlink")

	victimContents, err := os.ReadFile(victimFile)
	require.NoError(t, err)
	require.Equal(t, "original = true\n", string(victimContents))
}

func TestRefreshPluginManifestRefusesSymlinkedParent(t *testing.T) {
	tempDir := t.TempDir()
	victimDir := filepath.Join(tempDir, "victim-config")
	require.NoError(t, os.MkdirAll(victimDir, 0o755))

	configPath := filepath.Join(tempDir, "config-link")
	require.NoError(t, os.Symlink(victimDir, configPath))

	customConfig := &CustomTestConfig{
		customConfigPath: configPath,
	}
	customConfig.InitConfig()

	fs := afero.NewOsFs()
	manifestContent, err := os.ReadFile("./test_artifacts/plugins_updated.toml")
	require.NoError(t, err)

	testServers := setUpServers(t, manifestContent, nil)
	defer func() { testServers.CloseAll() }()

	err = RefreshPluginManifest(context.Background(), customConfig, fs, testServers.StripeServer.URL)
	require.ErrorContains(t, err, "symlink")

	_, err = os.Stat(filepath.Join(victimDir, "plugins.toml"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestResolvePluginForInstallReturnsErrPluginNotFoundWhenLoggedIn(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	failingServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			res.WriteHeader(http.StatusNotFound)
			res.Write([]byte(`{"error":{"message":"not found"}}`))
		case "/v1/stripecli/get-plugin-url":
			body, _ := json.Marshal(requests.PluginData{
				PluginBaseURL:       testServers.ArtifactoryServer.URL,
				AdditionalManifests: nil,
			})
			res.Write(body)
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
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	originalPluginData := requests.DefaultPluginData
	requests.DefaultPluginData = requests.PluginData{
		PluginBaseURL:       testServers.ArtifactoryServer.URL,
		AdditionalManifests: nil,
	}
	defer func() {
		requests.DefaultPluginData = originalPluginData
	}()

	failingServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/ajax/stripecli/plugins_metadata":
			res.WriteHeader(http.StatusNotFound)
			res.Write([]byte(`{"error":{"message":"not found"}}`))
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
			res.Write(body)
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

	w.Close()
	os.Stderr = orig

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

func makeManifestWithPlugin(shortname, version string) string {
	return fmt.Sprintf(`[[Plugin]]
  Shortname = "%s"
  Binary = "stripe-cli-%s"
  MagicCookieValue = "TEST-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "%s"
    Sum = "abc123"
`, shortname, shortname, runtime.GOARCH, runtime.GOOS, version)
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

	// Simulate v1.0.0 installed on disk.
	pluginBinaryPath := fmt.Sprintf("/plugins/myplugin/1.0.0/stripe-cli-myplugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("binary"), 0755))

	// Manifest has v1.1.0.
	require.NoError(t, afero.WriteFile(fs, "/plugins.toml", []byte(makeManifestWithPlugin("myplugin", "1.1.0")), os.ModePerm))

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(config, fs, plugin)
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

	require.NoError(t, afero.WriteFile(fs, "/plugins.toml", []byte(makeManifestWithPlugin("myplugin", "1.1.0")), os.ModePerm))

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(config, fs, plugin)
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

	require.NoError(t, afero.WriteFile(fs, "/plugins.toml", []byte(makeManifestWithPlugin("myplugin", "1.1.0")), os.ModePerm))

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(config, fs, plugin)
	})

	require.Empty(t, output)
}

func TestCheckLatestPluginVersionSilentWhenNoManifest(t *testing.T) {
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
		CheckLatestPluginVersion(config, fs, plugin)
	})

	require.Empty(t, output)
}

func TestCheckLatestPluginVersionSilentInDevMode(t *testing.T) {
	orig := PluginsPath
	PluginsPath = "/some/local/dev/path"
	defer func() { PluginsPath = orig }()

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
	require.NoError(t, afero.WriteFile(fs, "/plugins.toml", []byte(makeManifestWithPlugin("myplugin", "1.1.0")), os.ModePerm))

	output := captureStderr(t, func() {
		CheckLatestPluginVersion(config, fs, plugin)
	})

	require.Empty(t, output)
}
