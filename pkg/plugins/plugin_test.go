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
	"time"

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
			// The server keys the auto-install rollout on this, so every metadata
			// request has to carry it.
			require.Equal(t, TestMachineUUID, req.URL.Query().Get("machine_uuid"))
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
			// The anonymous endpoint keys the auto-install rollout on this too, and it
			// is the only identifier it gets.
			require.Equal(t, TestMachineUUID, req.URL.Query().Get("machine_uuid"))
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
`, runtime.GOARCH, runtime.GOOS, binarySum)

	var metadataLookups int

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

func TestRunVersionOverrideNotInstalled(t *testing.T) {
	fs := setUpFS()
	cfg := &TestConfig{}
	cfg.InitConfig()

	t.Setenv("STRIPE_PLUGINS_PATH", "/plugins")

	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")

	// Only local.build.dev is on disk
	require.NoError(t, fs.MkdirAll("/plugins/appA/local.build.dev", 0755))
	afero.WriteFile(fs, "/plugins/appA/local.build.dev/stripe-cli-app-a"+GetBinaryExtension(), []byte("bin"), 0755)

	// Temporarily clear PluginsPath so it doesn't force local.build.dev
	origPluginsPath := PluginsPath
	PluginsPath = ""
	defer func() { PluginsPath = origPluginsPath }()

	err := plugin.Run(context.Background(), &cfg.Config, fs, nil, "", "9.9.9", "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), `plugin "appA" version "9.9.9" is not installed`)
	require.Contains(t, err.Error(), "installed version is local.build.dev")
}

func TestRunVersionOverrideSelectsSpecifiedVersion(t *testing.T) {
	fs := setUpFS()
	cfg := &TestConfig{}
	cfg.InitConfig()

	t.Setenv("STRIPE_PLUGINS_PATH", "/plugins")

	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")

	// Both local.build.dev and 1.0.1 exist on disk
	require.NoError(t, fs.MkdirAll("/plugins/appA/local.build.dev", 0755))
	afero.WriteFile(fs, "/plugins/appA/local.build.dev/stripe-cli-app-a"+GetBinaryExtension(), []byte("bin"), 0755)
	require.NoError(t, fs.MkdirAll("/plugins/appA/1.0.1", 0755))
	afero.WriteFile(fs, "/plugins/appA/1.0.1/stripe-cli-app-a"+GetBinaryExtension(), []byte("bin"), 0755)

	origPluginsPath := PluginsPath
	PluginsPath = ""
	defer func() { PluginsPath = origPluginsPath }()

	// Run with override "1.0.1" — this will get past version resolution but fail
	// at the go-plugin handshake (expected; we're testing that version resolution works).
	err := plugin.Run(context.Background(), &cfg.Config, fs, nil, "", "1.0.1", "", "", "")
	// The error should NOT be about version not installed; it should be a runtime error
	// from trying to actually execute the fake binary.
	require.Error(t, err)
	require.NotContains(t, err.Error(), "is not installed")
}

func TestRunVersionOverrideBypassesLocalBuildDev(t *testing.T) {
	fs := setUpFS()
	cfg := &TestConfig{}
	cfg.InitConfig()

	t.Setenv("STRIPE_PLUGINS_PATH", "/plugins")

	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")

	// Both exist
	require.NoError(t, fs.MkdirAll("/plugins/appA/local.build.dev", 0755))
	afero.WriteFile(fs, "/plugins/appA/local.build.dev/stripe-cli-app-a"+GetBinaryExtension(), []byte("bin"), 0755)
	require.NoError(t, fs.MkdirAll("/plugins/appA/2.0.1", 0755))
	afero.WriteFile(fs, "/plugins/appA/2.0.1/stripe-cli-app-a"+GetBinaryExtension(), []byte("bin"), 0755)

	origPluginsPath := PluginsPath
	PluginsPath = ""
	defer func() { PluginsPath = origPluginsPath }()

	// Without override, lookUpInstalledVersion would return local.build.dev.
	// With override "2.0.1", it should bypass that and use 2.0.1.
	err := plugin.Run(context.Background(), &cfg.Config, fs, nil, "", "2.0.1", "", "", "")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "is not installed")
}

// runAutoUpgradeResolution is what the upgrade lookup returns for appA, at a version
// the test manifest deliberately does not have. Resolving to a version only the
// returned plugin's own metadata knows about is what lets a test tell "Run ran the
// upgraded plugin" apart from "Run ran the old plugin under a new version number":
// only the former can find a checksum.
func runAutoUpgradeResolution(version string) *ResolvedPluginVersion {
	return &ResolvedPluginVersion{
		Plugin: &Plugin{
			Shortname:        "appA",
			Binary:           "stripe-cli-app-a",
			MagicCookieValue: "0337A75A-C3C4-4DCF-A9EF-E7A144E5A291",
			Releases: []Release{
				{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: version, Sum: "3c909ec628b7d32536a65fd15545db527a7509e734995d4ef4806694fbd89c3a"},
			},
		},
		Version:   version,
		BinaryURL: "https://artifacts.example/appA/" + version,
	}
}

// setUpRunAutoUpgrade puts appA on disk at the given version and returns everything
// the Run wiring tests below need to call it.
func setUpRunAutoUpgrade(t *testing.T, installedVersion string) (*autoUpgradeStubs, *TestConfig, afero.Fs) {
	t.Helper()

	stubs := stubAutoUpgrade(t)
	stubs.resolved = runAutoUpgradeResolution("3.0.0")

	fs := setUpFS()
	cfg := &TestConfig{}
	cfg.InitConfig()

	t.Setenv("STRIPE_PLUGINS_PATH", "/plugins")

	installDir := filepath.Join("/plugins/appA", installedVersion)
	require.NoError(t, fs.MkdirAll(installDir, 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(installDir, "stripe-cli-app-a"+GetBinaryExtension()), []byte("bin"), 0755))

	return stubs, cfg, fs
}

func TestRunAutoUpgradesTheInstalledVersion(t *testing.T) {
	stubs, cfg, fs := setUpRunAutoUpgrade(t, "1.0.1")

	plugin, err := LookUpPlugin(context.Background(), cfg, fs, "appA")
	require.NoError(t, err)

	runErr := plugin.Run(context.Background(), &cfg.Config, fs, nil, "", "", "", "", "")

	require.Equal(t, []string{"appA"}, stubs.settingReads)
	require.Len(t, stubs.resolveCalls, 1)
	require.Equal(t, "appA", stubs.resolveCalls[0].pluginName)

	// 1.0.1 is what was on disk, so it is what the upgrade had to be told is installed
	// -- both to decide 3.0.0 is newer and to tell the plugin what to migrate from.
	require.Equal(t, []autoUpgradeInstallCall{{
		version:          "3.0.0",
		apiBaseURL:       "https://api.stripe.com",
		dashboardBaseURL: "https://dashboard.stripe.com",
	}}, stubs.installCalls)
	require.Equal(t, []autoUpgradePostInstallCall{{version: "3.0.0", previousVersion: "1.0.1"}}, stubs.postInstallCalls)

	// Run gets no further than launching the binary, because the stubbed install never
	// wrote one. Which version it got that far with is the point: appA's own metadata
	// has no 3.0.0 release, so failing anywhere later than the checksum lookup means
	// Run took both the version and the plugin metadata the upgrade handed back.
	require.Error(t, runErr)
	require.NotContains(t, runErr.Error(), "could not locate a valid checksum")
}

func TestRunSkipsAutoUpgradeForPinnedVersion(t *testing.T) {
	stubs, cfg, fs := setUpRunAutoUpgrade(t, "1.0.1")

	plugin, err := LookUpPlugin(context.Background(), cfg, fs, "appA")
	require.NoError(t, err)

	require.Error(t, plugin.Run(context.Background(), &cfg.Config, fs, nil, "", "1.0.1", "", "", ""))

	// Someone who named a version wants that version. The setting is not even read:
	// there is nothing for it to decide here.
	require.Empty(t, stubs.settingReads)
	require.Empty(t, stubs.resolveCalls)
	require.Empty(t, stubs.installCalls)
}

func TestRunSkipsAutoUpgradeForLocalDevelopmentBuild(t *testing.T) {
	stubs, cfg, fs := setUpRunAutoUpgrade(t, localDevelopmentVersion)

	plugin, err := LookUpPlugin(context.Background(), cfg, fs, "appA")
	require.NoError(t, err)

	// A localdev build of the CLI, which is what PluginsPath marks. stubAutoUpgrade
	// cleared it, and restores it on cleanup.
	PluginsPath = "/plugins"

	require.Error(t, plugin.Run(context.Background(), &cfg.Config, fs, nil, "", "", "", "", ""))

	// Replacing a plugin developer's own build with a published release would throw
	// away the thing they are working on. Two things stop that -- the switch branch
	// this takes has no upgrade check, and maybeAutoUpgrade refuses on PluginsPath
	// regardless -- so this asserts the outcome rather than either mechanism.
	require.Empty(t, stubs.settingReads)
	require.Empty(t, stubs.resolveCalls)
	require.Empty(t, stubs.installCalls)
}

func TestRunWithoutAutoUpgradeSkipsTheCheck(t *testing.T) {
	stubs, cfg, fs := setUpRunAutoUpgrade(t, "1.0.1")

	plugin, err := LookUpPlugin(context.Background(), cfg, fs, "appA")
	require.NoError(t, err)

	require.Error(t, plugin.RunWithoutAutoUpgrade(context.Background(), &cfg.Config, fs, []string{"--help"}, "", "", "", "", ""))

	// Not even the setting is read. Both callers of this reach a plugin with nothing to
	// gain from the check -- printing help, or running something whose install just
	// resolved the newest release -- so neither should pay a request for it.
	require.Empty(t, stubs.settingReads)
	require.Empty(t, stubs.resolveCalls)
	require.Empty(t, stubs.installCalls)
}

func TestRunPeerPluginSkipsAutoUpgrade(t *testing.T) {
	stubs, cfg, fs := setUpRunAutoUpgrade(t, "1.0.1")

	// RunPeerPlugin needs a real *config.Config to run anything at all, and setUpFS
	// wrote appA's metadata under TestConfig's folder, which is somewhere else. Without
	// this the peer lookup fails and every assertion below passes for the wrong reason.
	plugin, err := LookUpPlugin(context.Background(), cfg, fs, "appA")
	require.NoError(t, err)
	require.NoError(t, writeLocalPluginMetadata(&cfg.Config, fs, plugin))

	helper := NewCoreCLIHelper(context.Background(), &cfg.Config, fs, "", "", "")

	runErr := helper.RunPeerPlugin("appA", nil, "")
	require.Error(t, runErr)
	require.NotContains(t, runErr.Error(), "not found")

	// The user ran the plugin that called this one, and is waiting on it. Interrupting
	// its output to announce a download of something they never named, and stalling it
	// on that download, is not what they asked for.
	require.Empty(t, stubs.settingReads)
	require.Empty(t, stubs.resolveCalls)
	require.Empty(t, stubs.installCalls)
}

func TestPostInstallSelectsSpecifiedVersion(t *testing.T) {
	fs := setUpFS()
	cfg := &TestConfig{}
	cfg.InitConfig()

	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")

	require.NoError(t, fs.MkdirAll("/plugins/appA/2.0.1", 0755))
	afero.WriteFile(fs, "/plugins/appA/2.0.1/stripe-cli-app-a"+GetBinaryExtension(), []byte("bin"), 0755)

	// PostInstall will fail at the go-plugin handshake against the fake binary; we're only
	// verifying that it attempts to dispense the correct version rather than erroring out early.
	err := plugin.PostInstall(context.Background(), &cfg.Config, fs, "2.0.1", "1.0.1", "", "", "")
	require.Error(t, err)
}

func TestPreUninstallSelectsSpecifiedVersion(t *testing.T) {
	fs := setUpFS()
	cfg := &TestConfig{}
	cfg.InitConfig()

	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")

	require.NoError(t, fs.MkdirAll("/plugins/appA/2.0.1", 0755))
	afero.WriteFile(fs, "/plugins/appA/2.0.1/stripe-cli-app-a"+GetBinaryExtension(), []byte("bin"), 0755)

	err := plugin.PreUninstall(context.Background(), &cfg.Config, fs, "2.0.1", "", "", "")
	require.Error(t, err)
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

	// One version, so what this pins is only that a release is found with no local build
	// present. It used to install two and expect the lower of them, which made it a test
	// of glob ordering wearing this name.
	require.NoError(t, fs.MkdirAll("/plugins/appA/2.0.1", 0755))

	version, err := plugin.lookUpInstalledVersion(config, fs)
	require.NoError(t, err)
	require.Equal(t, "2.0.1", version)
}

// TestLookUpInstalledVersionPrefersTheNewestRelease covers a plugin directory holding more
// than one version. Installing is supposed to leave exactly one -- cleanUpPluginPath
// deletes the rest -- but an interrupted or concurrent install can leave several, and this
// is the function every caller trusts to say which one is installed.
func TestLookUpInstalledVersionPrefersTheNewestRelease(t *testing.T) {
	tests := []struct {
		name      string
		installed []string
		want      string
	}{
		{
			// The old behavior, and a downgrade: glob order is lexical, so 1.0.1 came
			// back first and was reported as the installed version over 2.0.1.
			name:      "a newer version alongside an older one",
			installed: []string{"1.0.1", "2.0.1"},
			want:      "2.0.1",
		},
		{
			// Ordered as versions rather than as strings, which is a different answer
			// here: 1.9.0 is the lexically greater of the two.
			name:      "a double-digit patch series",
			installed: []string{"1.9.0", "1.10.0"},
			want:      "1.10.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := setUpFS()
			config := &TestConfig{}

			plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")

			for _, installed := range tt.installed {
				require.NoError(t, fs.MkdirAll(filepath.Join("/plugins/appA", installed), 0755))
			}

			version, err := plugin.lookUpInstalledVersion(config, fs)
			require.NoError(t, err)
			require.Equal(t, tt.want, version)

			// The same question, so the same answer. These were two separate walks of this
			// directory that disagreed, and InstalledVersion is the one handed to a
			// plugin's PostInstall hook as the version being migrated from.
			require.Equal(t, tt.want, plugin.InstalledVersion(config, fs))
		})
	}
}

// TestInstalledVersionReportsALocalDevelopmentBuild pins the half of that disagreement with
// teeth in it. InstalledVersion had no notion of a local build, so with one sitting beside a
// release it named the release -- a version the CLI would not run, offered to the upgrade
// hint as the thing to compare against, telling a plugin developer their own build was out
// of date.
func TestInstalledVersionReportsALocalDevelopmentBuild(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")

	require.NoError(t, fs.MkdirAll("/plugins/appA/2.0.1", 0755))
	require.NoError(t, fs.MkdirAll("/plugins/appA/"+localDevelopmentVersion, 0755))

	require.Equal(t, localDevelopmentVersion, plugin.InstalledVersion(config, fs))
}

// TestLookUpInstalledVersionIgnoresFiles pins the directory check. The glob this walks keys
// on the two dots in a name, not on being a directory, and InstalledVersion used to screen
// for directories itself before it was folded into here.
func TestLookUpInstalledVersionIgnoresFiles(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")

	require.NoError(t, fs.MkdirAll("/plugins/appA/1.0.1", 0755))
	require.NoError(t, afero.WriteFile(fs, "/plugins/appA/9.9.9", []byte("not a version"), 0644))

	version, err := plugin.lookUpInstalledVersion(config, fs)
	require.NoError(t, err)
	require.Equal(t, "1.0.1", version)
	require.Equal(t, "1.0.1", plugin.InstalledVersion(config, fs))
}

func TestInstalledVersionEmptyWithNothingInstalled(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")

	require.Empty(t, plugin.InstalledVersion(config, fs))
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

// The check stamp is the one piece of per-plugin state that does not live in the
// metadata file or the plugin directory, so nothing else in Uninstall reaches it.
func TestUninstallRemovesTheAutoUpgradeCheckStamp(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	plugin := Plugin{
		Shortname:        "sample-plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
		Releases: []Release{
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.0.0", Sum: "abc123"},
		},
	}

	require.NoError(t, writeLocalPluginMetadata(config, fs, plugin))
	require.NoError(t, fs.MkdirAll("/plugins/sample-plugin/1.0.0", 0755))
	writeAutoUpgradeCheckStamp(t, config, fs, "sample-plugin", time.Now())
	require.True(t, autoUpgradeCheckStampExists(t, config, fs, "sample-plugin"))

	require.NoError(t, plugin.Uninstall(context.Background(), config, fs))

	require.False(t, autoUpgradeCheckStampExists(t, config, fs, "sample-plugin"))
}

// The ordinary case, since auto-update is off by default: there is no stamp to remove,
// and an uninstall must not report that as a problem.
func TestUninstallSucceedsWithoutAnAutoUpgradeCheckStamp(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	plugin := Plugin{
		Shortname:        "sample-plugin",
		Binary:           "stripe-cli-sample-plugin",
		MagicCookieValue: "SAMPLE-COOKIE",
		Releases: []Release{
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.0.0", Sum: "abc123"},
		},
	}

	require.NoError(t, writeLocalPluginMetadata(config, fs, plugin))
	require.NoError(t, fs.MkdirAll("/plugins/sample-plugin/1.0.0", 0755))
	require.False(t, autoUpgradeCheckStampExists(t, config, fs, "sample-plugin"))

	require.NoError(t, plugin.Uninstall(context.Background(), config, fs))
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
