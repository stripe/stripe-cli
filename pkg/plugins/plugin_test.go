package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
)

type failRemoveAllFs struct {
	afero.Fs
	path string
	err  error
}

func (fs *failRemoveAllFs) RemoveAll(name string) error {
	if filepath.Clean(name) == filepath.Clean(fs.path) {
		return fs.err
	}

	return fs.Fs.RemoveAll(name)
}

func TestLookUpLatestVersion(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	version := plugin.LookUpLatestVersion()
	require.Equal(t, "2.0.1", version)
}

func TestInstall(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL, testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.Nil(t, err)
	require.True(t, fileExists)

	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())
}

func TestInstallRollsBackPersistedStateWhenConfigWriteFails(t *testing.T) {
	fs := setUpFS()
	config := &FailingWriteConfig{
		WriteErr:                 errors.New("boom"),
		MutateInstalledPluginsOn: true,
	}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL, testServers.StripeServer.URL)
	require.ErrorIs(t, err, config.WriteErr)

	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.NoError(t, err)
	require.False(t, fileExists)

	metadataPath, err := getLocalPluginMetadataPath(config, "appA")
	require.NoError(t, err)
	metadataExists, err := afero.Exists(fs, metadataPath)
	require.NoError(t, err)
	require.True(t, metadataExists)
	require.Empty(t, config.GetInstalledPlugins())
}

func TestInstallUsesPluginMetadataEndpointWhenAPIKeyAvailable(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch {
		case strings.Contains(req.URL.String(), "/appA/2.0.1"):
			res.Write([]byte("hello, I am appA_2.0.1"))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer artifactoryServer.Close()

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      fmt.Sprintf("%s/appA/2.0.1/%s/%s/stripe-cli-app-a", artifactoryServer.URL, runtime.GOOS, runtime.GOARCH),
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", stripeServer.URL, stripeServer.URL)
	require.NoError(t, err)
}

func TestInstallUsesAnonymousPluginMetadataEndpointWhenAPIKeyUnavailable(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch {
		case strings.Contains(req.URL.String(), "/appA/2.0.1"):
			res.Write([]byte("hello, I am appA_2.0.1"))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer artifactoryServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		t.Fatalf("anonymous plugin metadata install should not hit the API host: %s", req.URL.String())
	}))
	defer apiServer.Close()

	dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/ajax/stripecli/plugins_metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      fmt.Sprintf("%s/appA/2.0.1/%s/%s/stripe-cli-app-a", artifactoryServer.URL, runtime.GOOS, runtime.GOARCH),
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer dashboardServer.Close()

	plugin := &Plugin{Shortname: "appA"}
	err := plugin.Install(context.Background(), config, fs, "2.0.1", apiServer.URL, dashboardServer.URL)
	require.NoError(t, err)

	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.NoError(t, err)
	require.True(t, fileExists)

	cachedPlugin, err := readLocalPluginMetadata(config, fs, "appA")
	require.NoError(t, err)
	require.Equal(t, "stripe-cli-app-a", cachedPlugin.Binary)
	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())
}

func TestInstallFailsIfPluginMetadataEndpointFails(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()

	fallbackServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(`{"error":{"message":"boom"}}`))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer fallbackServer.Close()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", fallbackServer.URL, fallbackServer.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not resolve download URL for plugin")
	require.Contains(t, err.Error(), "failed to fetch plugin metadata")
	require.Contains(t, err.Error(), "boom")
}

func TestInstallFailsIfMetadataBinaryURLReturnsNotFound(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case fmt.Sprintf("/appA/2.0.1/%s/%s/binary", runtime.GOOS, runtime.GOARCH):
			res.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer artifactoryServer.Close()

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      fmt.Sprintf("%s/appA/2.0.1/%s/%s/binary", artifactoryServer.URL, runtime.GOOS, runtime.GOARCH),
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", stripeServer.URL, stripeServer.URL)
	require.Error(t, err)
}

func TestInstallFailsIfMetadataBinaryDownloadFails(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case fmt.Sprintf("/appA/2.0.1/%s/%s/binary", runtime.GOOS, runtime.GOARCH):
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte("html error page"))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer artifactoryServer.Close()

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      fmt.Sprintf("%s/appA/2.0.1/%s/%s/binary", artifactoryServer.URL, runtime.GOOS, runtime.GOARCH),
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", stripeServer.URL, stripeServer.URL)
	require.Error(t, err)
}

func TestInstallPersistsLocalMetadataWithoutManifest(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch {
		case strings.Contains(req.URL.String(), "/appA/2.0.1"):
			res.Write([]byte("hello, I am appA_2.0.1"))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer artifactoryServer.Close()

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      fmt.Sprintf("%s/appA/2.0.1/%s/%s/stripe-cli-app-a", artifactoryServer.URL, runtime.GOOS, runtime.GOARCH),
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	plugin := &Plugin{Shortname: "appA"}
	err := plugin.Install(context.Background(), config, fs, "2.0.1", stripeServer.URL, stripeServer.URL)
	require.NoError(t, err)

	cachedPlugin, err := readLocalPluginMetadata(config, fs, "appA")
	require.NoError(t, err)
	require.Equal(t, "stripe-cli-app-a", cachedPlugin.Binary)
	require.NotNil(t, cachedPlugin.getReleaseForVersion("2.0.1"))
	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())

	lookedUpPlugin, err := LookUpPlugin(context.Background(), config, fs, "appA")
	require.NoError(t, err)
	require.Equal(t, cachedPlugin, lookedUpPlugin)

	_, err = fs.Stat("/plugins.toml")
	require.True(t, os.IsNotExist(err))
}

func TestResolvePluginForInstallUsesMetadataWithoutCachedManifest(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	var metadataLookups int
	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			metadataLookups++
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
	defer stripeServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", stripeServer.URL, stripeServer.URL)
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

func TestResolvePluginForInstallResolvesLatestVersionFromMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	var metadataLookups int
	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			metadataLookups++
			require.Equal(t, "", req.URL.Query().Get("version"))
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "https://example.test/appA/latest",
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "", stripeServer.URL, stripeServer.URL)
	require.NoError(t, err)
	plugin := resolvedPlugin.Plugin
	version := resolvedPlugin.Version
	require.NotNil(t, plugin)
	require.Equal(t, "2.0.1", version)
	require.Equal(t, "https://example.test/appA/latest", resolvedPlugin.BinaryURL)
	require.Equal(t, 1, metadataLookups)
}

func TestResolvePluginForInstallFallsBackToCachedLocalMetadataWhenMetadataFails(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()

	fallbackServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			res.WriteHeader(http.StatusInternalServerError)
			_, _ = res.Write([]byte(`{"error":{"message":"boom"}}`))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer fallbackServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", fallbackServer.URL, fallbackServer.URL)
	require.NoError(t, err)
	plugin := resolvedPlugin.Plugin
	version := resolvedPlugin.Version
	require.NotNil(t, plugin)
	require.Equal(t, "appA", plugin.Shortname)
	require.Equal(t, "2.0.1", version)
	require.Empty(t, resolvedPlugin.BinaryURL)
}

func TestResolvedPluginInstallUsesResolvedMetadataWithoutSecondLookup(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	var metadataLookups int
	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch {
		case strings.Contains(req.URL.String(), "/appA/2.0.1"):
			res.Write([]byte("hello, I am appA_2.0.1"))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer artifactoryServer.Close()

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			metadataLookups++
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      fmt.Sprintf("%s/appA/2.0.1/%s/%s/stripe-cli-app-a", artifactoryServer.URL, runtime.GOOS, runtime.GOARCH),
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", stripeServer.URL, stripeServer.URL)
	require.NoError(t, err)
	require.Equal(t, 1, metadataLookups)

	err = resolvedPlugin.Install(context.Background(), config, fs, stripeServer.URL, stripeServer.URL)
	require.NoError(t, err)
	require.Equal(t, 1, metadataLookups)
}

func TestResolvedPluginInstallRetriesMetadataAfterCachedLocalFallback(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	binaryBody := []byte("hello, I am generate_1.0.0")
	binarySum := fmt.Sprintf("%x", sha256.Sum256(binaryBody))

	metadataManifest := fmt.Sprintf(`[[Plugin]]
  Shortname = "generate"
  Shortdesc = "Generate things"
  Binary = "stripe-cli-generate"
  MagicCookieValue = "GENERATE-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "1.0.0"
    Sum = "%s"
    Runtime = {node = "20"}
`, runtime.GOARCH, runtime.GOOS, binarySum)

	var metadataLookups int

	nodeBinaryPath := GetNodeBinaryPath(config, "20")
	require.NoError(t, fs.MkdirAll(filepath.Dir(nodeBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, nodeBinaryPath, []byte("node"), 0755))
	require.NoError(t, writeLocalPluginMetadata(config, fs, Plugin{
		Shortname:        "generate",
		Shortdesc:        "Generate things",
		Binary:           "stripe-cli-generate",
		MagicCookieValue: "GENERATE-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     binarySum,
			},
		},
	}))

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case fmt.Sprintf("/generate/1.0.0/%s/%s/stripe-cli-generate", runtime.GOOS, runtime.GOARCH):
			_, _ = res.Write(binaryBody)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer artifactoryServer.Close()

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			metadataLookups++
			if metadataLookups == 1 {
				res.WriteHeader(http.StatusInternalServerError)
				_, _ = res.Write([]byte(`{"error":{"message":"boom"}}`))
				return
			}

			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      fmt.Sprintf("%s/generate/1.0.0/%s/%s/stripe-cli-generate", artifactoryServer.URL, runtime.GOOS, runtime.GOARCH),
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
	require.Equal(t, 1, metadataLookups)

	err = resolvedPlugin.Install(context.Background(), config, fs, stripeServer.URL, stripeServer.URL)
	require.NoError(t, err)
	require.Equal(t, 2, metadataLookups)

	cachedPlugin, err := readLocalPluginMetadata(config, fs, "generate")
	require.NoError(t, err)
	release := cachedPlugin.getReleaseForVersion("1.0.0")
	require.NotNil(t, release)
	require.Equal(t, "20", release.Runtime["node"])
}

func TestResolvePluginForAutoInstallPrefersFreshMetadataWhenLocalMetadataIsStale(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	stalePlugin := Plugin{
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
	require.NoError(t, writeLocalPluginMetadata(config, fs, stalePlugin))

	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	resolvedPlugin, err := resolvePluginForAutoInstall(context.Background(), config, fs, "appA", testServers.StripeServer.URL, testServers.StripeServer.URL)
	require.NoError(t, err)
	plugin := resolvedPlugin.Plugin
	version := resolvedPlugin.Version
	require.NotNil(t, plugin)
	require.Equal(t, "2.0.1", version)
	require.Equal(t, "2.0.1", plugin.LookUpLatestVersion())
}

func TestResolvePluginForAutoInstallFallsBackToCachedLocalMetadataWhenFreshLookupFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	stalePlugin := Plugin{
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
	require.NoError(t, writeLocalPluginMetadata(config, fs, stalePlugin))

	failingServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusInternalServerError)
		_, _ = res.Write([]byte(`{"error":{"message":"boom"}}`))
	}))
	defer failingServer.Close()

	resolvedPlugin, err := resolvePluginForAutoInstall(context.Background(), config, fs, "appA", failingServer.URL, failingServer.URL)
	require.NoError(t, err)
	plugin := resolvedPlugin.Plugin
	version := resolvedPlugin.Version
	require.NotNil(t, plugin)
	require.Equal(t, "1.0.1", version)
	require.Equal(t, "1.0.1", plugin.LookUpLatestVersion())
}

func TestVerifyChecksumSkipsLocalDevelopmentVersion(t *testing.T) {
	plugin := Plugin{Shortname: "appA"}

	err := plugin.verifyChecksum(strings.NewReader("locally built binary"), localDevelopmentVersion)
	require.NoError(t, err)
}

func TestPluginFromMetadataPreservesRuntimeRequirements(t *testing.T) {
	plugin := &Plugin{
		Shortname:        "generate",
		Binary:           "stripe-cli-generate",
		MagicCookieValue: "GENERATE-COOKIE",
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

	resolved, err := plugin.pluginFromMetadata(metadataManifest)
	require.NoError(t, err)
	release := resolved.getReleaseForVersion("1.0.0")
	require.NotNil(t, release)
	require.Equal(t, "20", release.Runtime["node"])
}

func TestInstallFailsIfNoAPIKeyAndMetadataReturnsNoBinaryURL(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/ajax/stripecli/plugins_metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "",
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer dashboardServer.Close()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", dashboardServer.URL, dashboardServer.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not resolve download URL for plugin")
}

func TestInstallFailsIfChecksumCouldNotBeFound(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "0.0.0", testServers.StripeServer.URL, testServers.StripeServer.URL)
	require.EqualError(t, err, "could not locate a valid checksum for appA version 0.0.0")

	// Require that we don't save the binary if checkum does not match
	file := fmt.Sprintf("/plugins/appA/0.0.0/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.Nil(t, err)
	require.False(t, fileExists)

	require.Equal(t, 0, len(config.GetInstalledPlugins()))
}

func TestInstallationFailsIfChecksumDoesNotMatch(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appB")
	err := plugin.Install(context.Background(), config, fs, "1.2.1", testServers.StripeServer.URL, testServers.StripeServer.URL)
	require.EqualError(t, err, "installed plugin 'appB' could not be verified, aborting installation")

	// Require that we don't save the binary if checkum does not match
	file := fmt.Sprintf("/plugins/appB/1.2.1/stripe-cli-app-b%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.Nil(t, err)
	require.False(t, fileExists)

	require.Equal(t, 0, len(config.GetInstalledPlugins()))
}

func TestInstallCleansOtherVersionsOfPlugin(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	// Download plugin version 0.0.1
	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "0.0.1", testServers.StripeServer.URL, testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/0.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, _ := afero.Exists(fs, file)
	require.True(t, fileExists, "Test setup failed -- did not download plugin version 0.0.1")

	// Download valid plugin
	err = plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL, testServers.StripeServer.URL)
	require.Nil(t, err)
	newFile := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, _ = afero.Exists(fs, newFile)
	require.True(t, fileExists, "Test setup failed -- did not download plugin version 2.0.1")

	// Require that the older version got removed from the fs
	fileExists, _ = afero.Exists(fs, file)
	require.False(t, fileExists, "Expected the original version of the plugin to be deleted.")

	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())
}

func TestInstallDoesNotCleanIfInstallFails(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	// Download valid plugin
	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL, testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, _ := afero.Exists(fs, file)
	require.True(t, fileExists, "Test setup failed -- did not download valid plugin")

	// Install fails for the same plugin because the checksum could not be found in manifest
	err = plugin.Install(context.Background(), config, fs, "0.0.0", testServers.StripeServer.URL, testServers.StripeServer.URL)
	require.EqualError(t, err, "could not locate a valid checksum for appA version 0.0.0")
	failedFile := fmt.Sprintf("/plugins/appA/0.0.0/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, _ = afero.Exists(fs, failedFile)
	require.False(t, fileExists, "Test setup failed -- did not expect plugin to be downloaded")

	// Require that we did not delete the initial version of the plugin
	fileExists, _ = afero.Exists(fs, file)
	require.True(t, fileExists, "Did not expect the original version of the plugin to be deleted.")
}

func TestLookUpInstalledVersionPrefersLocalDevelopmentVersion(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")

	require.NoError(t, fs.MkdirAll("/plugins/appA/local.build.dev", 0755))
	require.NoError(t, fs.MkdirAll("/plugins/appA/2.0.1", 0755))

	version, err := plugin.lookUpInstalledVersion(config, fs)
	require.NoError(t, err)
	require.Equal(t, localDevelopmentVersion, version)
}

func TestLookUpInstalledVersionFallsBackToInstalledRelease(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")

	require.NoError(t, fs.MkdirAll("/plugins/appA/1.0.1", 0755))
	require.NoError(t, fs.MkdirAll("/plugins/appA/2.0.1", 0755))

	version, err := plugin.lookUpInstalledVersion(config, fs)
	require.NoError(t, err)
	require.Equal(t, "1.0.1", version)
}

func TestCommandInfoParsedFromManifest(t *testing.T) {
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	var pluginList PluginList
	_, err := toml.Decode(string(manifestContent), &pluginList)
	require.Nil(t, err)

	// appC should have Commands metadata
	var appC *Plugin
	for i, p := range pluginList.Plugins {
		if p.Shortname == "appC" {
			appC = &pluginList.Plugins[i]
			break
		}
	}
	require.NotNil(t, appC, "appC should be present in manifest")
	require.Equal(t, 2, len(appC.Commands))

	require.Equal(t, "create", appC.Commands[0].Name)
	require.Equal(t, "Create a resource", appC.Commands[0].Desc)
	require.Equal(t, 0, len(appC.Commands[0].Commands))

	require.Equal(t, "logs", appC.Commands[1].Name)
	require.Equal(t, "View logs", appC.Commands[1].Desc)
	require.Equal(t, 1, len(appC.Commands[1].Commands))
	require.Equal(t, "tail", appC.Commands[1].Commands[0].Name)
	require.Equal(t, "Tail logs in real-time", appC.Commands[1].Commands[0].Desc)
}

func TestCommandInfoNilWhenAbsent(t *testing.T) {
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	var pluginList PluginList
	_, err := toml.Decode(string(manifestContent), &pluginList)
	require.Nil(t, err)

	// appA has no Commands metadata — field should be nil
	var appA *Plugin
	for i, p := range pluginList.Plugins {
		if p.Shortname == "appA" {
			appA = &pluginList.Plugins[i]
			break
		}
	}
	require.NotNil(t, appA)
	require.Nil(t, appA.Commands)
}

func TestDescriptionParsedFromManifest(t *testing.T) {
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	var pluginList PluginList
	_, err := toml.Decode(string(manifestContent), &pluginList)
	require.Nil(t, err)

	// appC has a Description field
	var appC *Plugin
	for i, p := range pluginList.Plugins {
		if p.Shortname == "appC" {
			appC = &pluginList.Plugins[i]
			break
		}
	}
	require.NotNil(t, appC, "appC should be present in manifest")
	require.Equal(t, "A plugin with subcommands that demonstrates multi-line description support. Use stripe appC --help to see the available subcommands.", appC.Description)
}

func TestDescriptionEmptyWhenAbsent(t *testing.T) {
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	var pluginList PluginList
	_, err := toml.Decode(string(manifestContent), &pluginList)
	require.Nil(t, err)

	// appA has no Description field — should be empty string
	var appA *Plugin
	for i, p := range pluginList.Plugins {
		if p.Shortname == "appA" {
			appA = &pluginList.Plugins[i]
			break
		}
	}
	require.NotNil(t, appA)
	require.Empty(t, appA.Description)
}

func TestDescriptionPreservedInLocalMetadataRoundTrip(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	plugin := Plugin{
		Shortname:        "myPlugin",
		Shortdesc:        "Short description",
		Description:      "A longer multi-line description for help output.",
		Binary:           "stripe-cli-my-plugin",
		MagicCookieValue: "COOKIE-VALUE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}

	require.NoError(t, writeLocalPluginMetadata(config, fs, plugin))

	cached, err := readLocalPluginMetadata(config, fs, "myPlugin")
	require.NoError(t, err)
	require.Equal(t, "A longer multi-line description for help output.", cached.Description)
}

func TestUninstall(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	// install a plugin to be uninstalled
	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL, testServers.StripeServer.URL)
	require.Nil(t, err)
	metadataPath, err := getLocalPluginMetadataPath(config, "appA")
	require.NoError(t, err)
	cacheExists, err := afero.Exists(fs, metadataPath)
	require.NoError(t, err)
	require.True(t, cacheExists)

	pluginDir := "/plugins/appA"
	err = plugin.Uninstall(context.Background(), config, fs)
	require.Nil(t, err)
	dirExists, _ := afero.Exists(fs, pluginDir)
	require.False(t, dirExists)
	cacheExists, err = afero.Exists(fs, metadataPath)
	require.NoError(t, err)
	require.False(t, cacheExists)

	require.Equal(t, 0, len(config.GetInstalledPlugins()))
}

func TestUninstallSucceedsWithLocalMetadataOnly(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	plugin := Plugin{
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

	require.NoError(t, writeLocalPluginMetadata(config, fs, plugin))
	require.NoError(t, fs.MkdirAll("/plugins/sample-plugin/1.0.0", 0755))

	err := plugin.Uninstall(context.Background(), config, fs)
	require.NoError(t, err)

	metadataPath, err := getLocalPluginMetadataPath(config, "sample-plugin")
	require.NoError(t, err)
	cacheExists, err := afero.Exists(fs, metadataPath)
	require.NoError(t, err)
	require.False(t, cacheExists)

	dirExists, err := afero.Exists(fs, "/plugins/sample-plugin")
	require.NoError(t, err)
	require.False(t, dirExists)

	require.Equal(t, 0, len(config.GetInstalledPlugins()))
}

func TestUninstallRejectsInvalidPluginShortnames(t *testing.T) {
	tests := []string{"../victim", "..\\victim"}

	for _, shortname := range tests {
		t.Run(shortname, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			config := &TestConfig{}
			config.InitConfig()
			config.InstalledPlugins = []string{shortname}

			require.NoError(t, fs.MkdirAll("/victim", 0755))
			require.NoError(t, afero.WriteFile(fs, "/victim/data.txt", []byte("keep me"), 0644))

			err := (&Plugin{Shortname: shortname}).Uninstall(context.Background(), config, fs)
			require.ErrorContains(t, err, "invalid plugin name")

			victimExists, statErr := afero.Exists(fs, "/victim/data.txt")
			require.NoError(t, statErr)
			require.True(t, victimExists)
			require.Equal(t, []string{shortname}, config.GetInstalledPlugins())
		})
	}
}

func TestUninstallReturnsErrorWithoutRemovingFilesWhenMetadataRemovalFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.InstalledPlugins = []string{"sample-plugin"}
	plugin := Plugin{
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

	require.NoError(t, writeLocalPluginMetadata(config, fs, plugin))
	pluginFile := fmt.Sprintf("/plugins/sample-plugin/1.0.0/stripe-cli-sample-plugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll("/plugins/sample-plugin/1.0.0", 0755))
	require.NoError(t, afero.WriteFile(fs, pluginFile, []byte("installed"), 0755))

	err := plugin.Uninstall(context.Background(), config, afero.NewReadOnlyFs(fs))
	require.Error(t, err)

	metadataPath, err := getLocalPluginMetadataPath(config, "sample-plugin")
	require.NoError(t, err)
	cacheExists, err := afero.Exists(fs, metadataPath)
	require.NoError(t, err)
	require.True(t, cacheExists)

	fileExists, err := afero.Exists(fs, pluginFile)
	require.NoError(t, err)
	require.True(t, fileExists)
	require.Equal(t, []string{"sample-plugin"}, config.GetInstalledPlugins())
}

func TestUninstallRollsBackStateWhenConfigWriteFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &FailingWriteConfig{
		WriteErr:                 errors.New("boom"),
		MutateInstalledPluginsOn: true,
	}
	config.InitConfig()
	config.InstalledPlugins = []string{"sample-plugin"}
	plugin := Plugin{
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

	require.NoError(t, writeLocalPluginMetadata(config, fs, plugin))
	pluginFile := fmt.Sprintf("/plugins/sample-plugin/1.0.0/stripe-cli-sample-plugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll("/plugins/sample-plugin/1.0.0", 0755))
	require.NoError(t, afero.WriteFile(fs, pluginFile, []byte("installed"), 0755))

	err := plugin.Uninstall(context.Background(), config, fs)
	require.ErrorIs(t, err, config.WriteErr)

	metadataPath, err := getLocalPluginMetadataPath(config, "sample-plugin")
	require.NoError(t, err)
	cacheExists, err := afero.Exists(fs, metadataPath)
	require.NoError(t, err)
	require.True(t, cacheExists)

	fileExists, err := afero.Exists(fs, pluginFile)
	require.NoError(t, err)
	require.True(t, fileExists)
	require.Equal(t, []string{"sample-plugin"}, config.GetInstalledPlugins())
}

func TestUninstallRollsBackStateWhenPluginRemovalFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	config.InstalledPlugins = []string{"sample-plugin"}
	plugin := Plugin{
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

	require.NoError(t, writeLocalPluginMetadata(config, fs, plugin))
	pluginFile := fmt.Sprintf("/plugins/sample-plugin/1.0.0/stripe-cli-sample-plugin%s", GetBinaryExtension())
	require.NoError(t, fs.MkdirAll("/plugins/sample-plugin/1.0.0", 0755))
	require.NoError(t, afero.WriteFile(fs, pluginFile, []byte("installed"), 0755))

	removeErr := errors.New("remove all failed")
	failingFS := &failRemoveAllFs{
		Fs:   fs,
		path: "/plugins/sample-plugin",
		err:  removeErr,
	}

	err := plugin.Uninstall(context.Background(), config, failingFS)
	require.ErrorIs(t, err, removeErr)

	metadataPath, err := getLocalPluginMetadataPath(config, "sample-plugin")
	require.NoError(t, err)
	cacheExists, err := afero.Exists(fs, metadataPath)
	require.NoError(t, err)
	require.True(t, cacheExists)

	fileExists, err := afero.Exists(fs, pluginFile)
	require.NoError(t, err)
	require.True(t, fileExists)
	require.Equal(t, []string{"sample-plugin"}, config.GetInstalledPlugins())
}

func TestVerifyChecksumAndSavePluginRefusesSymlink(t *testing.T) {
	manifestContent, err := os.ReadFile("./test_artifacts/plugins.toml")
	require.NoError(t, err)

	var pluginList PluginList
	_, err = toml.Decode(string(manifestContent), &pluginList)
	require.NoError(t, err)

	var plugin Plugin
	for _, candidate := range pluginList.Plugins {
		if candidate.Shortname == "appA" {
			plugin = candidate
			break
		}
	}

	require.Equal(t, "appA", plugin.Shortname)

	tempDir := t.TempDir()
	config := &CustomTestConfig{customConfigPath: tempDir}
	fs := afero.NewOsFs()

	pluginFilePath := filepath.Join(tempDir, "plugins", "appA", "2.0.1", "stripe-cli-app-a"+GetBinaryExtension())
	require.NoError(t, os.MkdirAll(filepath.Dir(pluginFilePath), 0o755))

	victimFile := filepath.Join(tempDir, "victim")
	require.NoError(t, os.WriteFile(victimFile, []byte("original"), 0o644))
	require.NoError(t, os.Symlink(victimFile, pluginFilePath))

	err = plugin.verifychecksumAndSavePlugin([]byte("hello, I am appA_2.0.1"), config, fs, "2.0.1")
	require.ErrorContains(t, err, "symlink")

	victimContents, err := os.ReadFile(victimFile)
	require.NoError(t, err)
	require.Equal(t, "original", string(victimContents))
}

func TestVerifyChecksumAndSavePluginRefusesSymlinkedParent(t *testing.T) {
	manifestContent, err := os.ReadFile("./test_artifacts/plugins.toml")
	require.NoError(t, err)

	var pluginList PluginList
	_, err = toml.Decode(string(manifestContent), &pluginList)
	require.NoError(t, err)

	var plugin Plugin
	for _, candidate := range pluginList.Plugins {
		if candidate.Shortname == "appA" {
			plugin = candidate
			break
		}
	}

	require.Equal(t, "appA", plugin.Shortname)

	tempDir := t.TempDir()
	victimDir := filepath.Join(tempDir, "victim-config")
	require.NoError(t, os.MkdirAll(victimDir, 0o755))

	configPath := filepath.Join(tempDir, "config-link")
	require.NoError(t, os.Symlink(victimDir, configPath))

	config := &CustomTestConfig{customConfigPath: configPath}
	fs := afero.NewOsFs()

	err = plugin.verifychecksumAndSavePlugin([]byte("hello, I am appA_2.0.1"), config, fs, "2.0.1")
	require.ErrorContains(t, err, "symlink")

	_, err = os.Stat(filepath.Join(victimDir, "plugins", "appA", "2.0.1", "stripe-cli-app-a"+GetBinaryExtension()))
	require.ErrorIs(t, err, os.ErrNotExist)
}
