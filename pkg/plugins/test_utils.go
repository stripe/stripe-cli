package plugins

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
)

// TestConfig Implementations out several methods
type TestConfig struct {
	config.Config
}

type FailingWriteConfig struct {
	TestConfig
	WriteErr                 error
	MutateInstalledPluginsOn bool
}

// WriteConfigField mocks out the method so that we can ensure installed plugins data is written
func (c *TestConfig) WriteConfigField(field string, value interface{}) error {
	c.InstalledPlugins = value.([]string)

	return nil
}

func (c *FailingWriteConfig) WriteConfigField(field string, value interface{}) error {
	if c.MutateInstalledPluginsOn {
		c.InstalledPlugins = append([]string(nil), value.([]string)...)
	}

	if c.WriteErr == nil {
		return c.TestConfig.WriteConfigField(field, value)
	}

	return c.WriteErr
}

// GetConfigFolder returns the absolute path for the TestConfig
func (c *TestConfig) GetConfigFolder(xdgPath string) string {
	return "/"
}

// GetInstalledPlugins returns the mocked out list of installed plugins
func (c *TestConfig) GetInstalledPlugins() []string {
	return c.InstalledPlugins
}

// InitConfig initializes the config with the values we need
func (c *TestConfig) InitConfig() {
	c.Profile.APIKey = "rk_test_11111111111111111111111111"
}

// setUpFS Sets up a memMap that contains the manifest
func setUpFS() afero.Fs {
	// test plugin manifest
	// Note that only some of entries have actual checksums that match with what the test server returns.
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	fs := afero.NewMemMapFs()
	// fs.Mkdir("test_config_folder", os.ModePerm)
	afero.WriteFile(fs, "/plugins.toml", manifestContent, os.ModePerm)
	return fs
}

// TestServers is a struct containing test servers that will be useful for unit testing plugin logic
type TestServers struct {
	ArtifactoryServer *httptest.Server
	StripeServer      *httptest.Server
}

// CloseAll calls Close() on each of the httptest servers.
func (ts *TestServers) CloseAll() {
	ts.ArtifactoryServer.Close()
	ts.StripeServer.Close()
}

// setUpServers sets up a local stripe server and artifactory server for unit tests
func setUpServers(t *testing.T, manifestContent []byte, additionalManifests map[string][]byte) TestServers {
	additionalManifestNames := []string{}
	for name := range additionalManifests {
		additionalManifestNames = append(additionalManifestNames, name)
	}

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); {
		case url == "/plugins.toml":
			res.Write(manifestContent)
		case contains(additionalManifestNames, strings.TrimPrefix(url, "/")):
			res.Write(additionalManifests[strings.TrimPrefix(url, "/")])
		case strings.Contains(url, "/appA/2.0.1"):
			res.Write([]byte("hello, I am appA_2.0.1"))
		case strings.Contains(url, "/appA/1.0.1"):
			res.Write([]byte("hello, I am appA_1.0.1"))
		case strings.Contains(url, "/appA/0.0.1"):
			res.Write([]byte("hello, I am appA_0.0.1"))
		case strings.Contains(url, "/appA/0.0.0"):
			// Binary exists that is not in the manifest
			res.Write([]byte("hello, I am appA_0.0.0"))
		case strings.Contains(url, "/appB/1.2.1"):
			// Mismatching checksums
			res.Write([]byte("hello, I am appB_1.2.1"))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	// The checksums in the test toml files are the same for each OS variation of the release for unit testing purposes
	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-url":
			pd := requests.PluginData{
				PluginBaseURL:       artifactoryServer.URL,
				AdditionalManifests: additionalManifestNames,
			}
			body, err := json.Marshal(pd)
			if err != nil {
				t.Error(err)
			}
			res.Write(body)
		case "/v1/stripecli/get-plugin-metadata":
			pluginName := req.URL.Query().Get("plugin")
			version := req.URL.Query().Get("version")
			opsystem := req.URL.Query().Get("os")
			arch := req.URL.Query().Get("arch")

			pluginManifest := singlePluginManifest(t, pluginName, manifestContent, additionalManifests)
			pd := requests.PluginMetadata{
				BinaryURL:      artifactoryServer.URL + "/" + pluginName + "/" + version + "/" + opsystem + "/" + arch + "/stripe-cli-" + strings.ReplaceAll(pluginName, "_", "-"),
				PluginManifest: string(pluginManifest),
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

	return TestServers{
		ArtifactoryServer: artifactoryServer,
		StripeServer:      stripeServer,
	}
}

func singlePluginManifest(t *testing.T, pluginName string, manifestContent []byte, additionalManifests map[string][]byte) []byte {
	t.Helper()

	merged := &PluginList{}
	if len(manifestContent) > 0 {
		requireNoError(t, toml.Unmarshal(manifestContent, merged))
	}

	for _, content := range additionalManifests {
		additionalPluginList := &PluginList{}
		requireNoError(t, toml.Unmarshal(content, additionalPluginList))
		mergePluginLists(merged, []*PluginList{additionalPluginList})
	}

	for _, plugin := range merged.Plugins {
		if plugin.Shortname != pluginName {
			continue
		}

		buffer := &bytes.Buffer{}
		requireNoError(t, toml.NewEncoder(buffer).Encode(PluginList{Plugins: []Plugin{plugin}}))
		return buffer.Bytes()
	}

	t.Fatalf("plugin %s not found in test manifests", pluginName)
	return nil
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func contains(sl []string, str string) bool {
	for _, s := range sl {
		if s == str {
			return true
		}
	}
	return false
}
