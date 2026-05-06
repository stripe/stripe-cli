package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
)

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
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.Nil(t, err)
	require.True(t, fileExists)

	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())
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
		case "/v1/stripecli/get-plugin-url":
			t.Fatalf("install should not fall back to /v1/stripecli/get-plugin-url when plugin metadata is available")
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", stripeServer.URL)
	require.NoError(t, err)
}

func TestInstallFallsBackIfPluginMetadataEndpointFails(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	fallbackServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(`{"error":{"message":"boom"}}`))
		case "/v1/stripecli/get-plugin-url":
			body, err := json.Marshal(requests.PluginData{
				PluginBaseURL:       testServers.ArtifactoryServer.URL,
				AdditionalManifests: nil,
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer fallbackServer.Close()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", fallbackServer.URL)
	require.NoError(t, err)
}

func TestInstallFallsBackIfMetadataBinaryURLReturnsNotFound(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	var pluginURLLookups int

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case fmt.Sprintf("/appA/2.0.1/%s/%s/binary", runtime.GOOS, runtime.GOARCH):
			res.WriteHeader(http.StatusNotFound)
		case fmt.Sprintf("/appA/2.0.1/%s/%s/stripe-cli-app-a", runtime.GOOS, runtime.GOARCH):
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
				BinaryURL:      fmt.Sprintf("%s/appA/2.0.1/%s/%s/binary", artifactoryServer.URL, runtime.GOOS, runtime.GOARCH),
				PluginManifest: string(singlePluginManifest(t, "appA", manifestContent, nil)),
			})
			require.NoError(t, err)
			res.Write(body)
		case "/v1/stripecli/get-plugin-url":
			pluginURLLookups++
			body, err := json.Marshal(requests.PluginData{
				PluginBaseURL:       artifactoryServer.URL,
				AdditionalManifests: nil,
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", stripeServer.URL)
	require.NoError(t, err)
	require.Equal(t, 1, pluginURLLookups)

	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.NoError(t, err)
	require.True(t, fileExists)
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

func TestInstallSucceedsIfNoAPIKey(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	originalPluginBaseURL := requests.DefaultPluginData.PluginBaseURL
	requests.DefaultPluginData.PluginBaseURL = testServers.ArtifactoryServer.URL
	defer func() {
		requests.DefaultPluginData.PluginBaseURL = originalPluginBaseURL
	}()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.Nil(t, err)
	require.True(t, fileExists)

	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())
}

func TestInstallFailsIfChecksumCouldNotBeFound(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "0.0.0", testServers.StripeServer.URL)
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
	err := plugin.Install(context.Background(), config, fs, "1.2.1", testServers.StripeServer.URL)
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
	err := plugin.Install(context.Background(), config, fs, "0.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/0.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, _ := afero.Exists(fs, file)
	require.True(t, fileExists, "Test setup failed -- did not download plugin version 0.0.1")

	// Download valid plugin
	err = plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
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
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, _ := afero.Exists(fs, file)
	require.True(t, fileExists, "Test setup failed -- did not download valid plugin")

	// Install fails for the same plugin because the checksum could not be found in manifest
	err = plugin.Install(context.Background(), config, fs, "0.0.0", testServers.StripeServer.URL)
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

func TestUninstall(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	// install a plugin to be uninstalled
	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)

	pluginDir := "/plugins/appA"
	err = plugin.Uninstall(context.Background(), config, fs)
	require.Nil(t, err)
	dirExists, _ := afero.Exists(fs, pluginDir)
	require.False(t, dirExists)

	require.Equal(t, 0, len(config.GetInstalledPlugins()))
}
