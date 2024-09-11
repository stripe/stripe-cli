package plugins

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/BurntSushi/toml"

	hcplugin "github.com/hashicorp/go-plugin"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// GetBinaryExtension returns the appropriate file extension for plugin binary
func GetBinaryExtension() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}

	return ""
}

// getPluginsDir computes where plugins are installed locally
func getPluginsDir(config config.IConfig) string {
	var pluginsDir string
	tempEnvPluginsPath := os.Getenv("STRIPE_PLUGINS_PATH")

	switch {
	case tempEnvPluginsPath != "":
		pluginsDir = tempEnvPluginsPath
	case PluginsPath != "":
		pluginsDir = PluginsPath
	default:
		configPath := config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
		pluginsDir = filepath.Join(configPath, "plugins")
	}

	return pluginsDir
}

// GetPluginList builds a list of allowed plugins to be installed and run by the CLI
func GetPluginList(ctx context.Context, config config.IConfig, fs afero.Fs) (PluginList, error) {
	var pluginList PluginList
	configPath := config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginManifestPath := filepath.Join(configPath, "plugins.toml")

	file, err := afero.ReadFile(fs, pluginManifestPath)
	if os.IsNotExist(err) {
		log.Debug("The plugin manifest file does not exist. Downloading...")
		err = RefreshPluginManifest(ctx, config, fs, stripe.DefaultAPIBaseURL)
		if err != nil {
			log.Debug("Could not download plugin manifest")
			return pluginList, err
		}
		file, err = afero.ReadFile(fs, pluginManifestPath)
	}

	if err != nil {
		return pluginList, err
	}

	_, err = toml.Decode(string(file), &pluginList)
	if err != nil {
		return pluginList, err
	}

	return pluginList, nil
}

// LookUpPlugin returns the matching plugin object
func LookUpPlugin(ctx context.Context, config config.IConfig, fs afero.Fs, pluginName string) (Plugin, error) {
	var plugin Plugin
	pluginList, err := GetPluginList(ctx, config, fs)
	if err != nil {
		return plugin, err
	}

	for _, p := range pluginList.Plugins {
		if pluginName == p.Shortname {
			return p, nil
		}
	}

	return plugin, fmt.Errorf("Could not find a plugin named %s", pluginName)
}

// RefreshPluginManifest refreshes the plugin manifest
func RefreshPluginManifest(ctx context.Context, config config.IConfig, fs afero.Fs, baseURL string) error {
	apiKey, err := config.GetProfile().GetAPIKey(false)
	if err != nil {
		return err
	}

	pluginData, err := requests.GetPluginData(ctx, baseURL, stripe.APIVersion, apiKey, config.GetProfile())
	if err != nil {
		return err
	}

	pluginList, err := fetchAndMergeManifests(pluginData)
	if err != nil {
		return err
	}

	configPath := config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginManifestPath := filepath.Join(configPath, "plugins.toml")

	body := new(bytes.Buffer)
	if err := toml.NewEncoder(body).Encode(pluginList); err != nil {
		return err
	}

	err = afero.WriteFile(fs, pluginManifestPath, body.Bytes(), 0644)

	if err != nil {
		return err
	}

	return nil
}

func fetchAndMergeManifests(pluginData requests.PluginData) (*PluginList, error) {
	pluginList, err := fetchPluginList(pluginData.PluginBaseURL, "plugins.toml")
	if err != nil {
		return nil, err
	}

	additionalPluginLists := []*PluginList{}
	for _, filename := range pluginData.AdditionalManifests {
		additionalPluginList, err := fetchPluginList(pluginData.PluginBaseURL, filename)
		if err != nil {
			return nil, err
		}
		additionalPluginLists = append(additionalPluginLists, additionalPluginList)
	}

	mergePluginLists(pluginList, additionalPluginLists)

	return pluginList, nil
}

func fetchPluginList(baseURL, manifestFilename string) (*PluginList, error) {
	pluginManifestURL := fmt.Sprintf("%s/%s", baseURL, manifestFilename)
	body, err := FetchRemoteResource(pluginManifestURL)
	if err != nil {
		return nil, err
	}
	return validatePluginManifest(body)
}

func validatePluginManifest(body []byte) (*PluginList, error) {
	var manifestBody PluginList

	if err := toml.Unmarshal(body, &manifestBody); err != nil {
		return nil, fmt.Errorf("Received an invalid plugin manifest. Error: %s", err)
	}
	if len(manifestBody.Plugins) == 0 {
		return nil, fmt.Errorf("Received an empty plugin manifest")
	}
	return &manifestBody, nil
}

// mergePluginLists merges additional plugin lists into the main plugin list, in place
func mergePluginLists(pluginList *PluginList, additionalPluginLists []*PluginList) {
	for _, list := range additionalPluginLists {
		for _, pl := range list.Plugins {
			addPluginToList(pluginList, pl)
		}
	}
}

func addPluginToList(pluginList *PluginList, pl Plugin) {
	idx := findPluginIndex(pluginList, pl)
	if idx == -1 {
		pluginList.Plugins = append(pluginList.Plugins, pl)
	} else {
		pluginList.Plugins[idx].Releases = append(pluginList.Plugins[idx].Releases, pl.Releases...)

		// Other code assumes the releases are sorted with latest version last.
		sort.Slice(pluginList.Plugins[idx].Releases, func(i, j int) bool {
			return pluginList.Plugins[idx].Releases[i].Version < pluginList.Plugins[idx].Releases[j].Version
		})
	}
}

func findPluginIndex(list *PluginList, p Plugin) int {
	for i, pp := range list.Plugins {
		if pp.MagicCookieValue == p.MagicCookieValue {
			return i
		}
	}
	return -1
}

// FetchRemoteResource returns the remote resource body
func FetchRemoteResource(url string) ([]byte, error) {
	t := &requests.TracedTransport{}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	trace := &httptrace.ClientTrace{
		GotConn: t.GotConn,
		DNSDone: t.DNSDone,
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	client := &http.Client{Transport: t}

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("remote resource not found: url=%s", url)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

// CleanupAllClients tears down and disconnects all "managed" plugin clients
func CleanupAllClients() {
	log.Debug("Tearing down plugin before exit")
	hcplugin.CleanupClients()
}

// IsPluginCommand returns true if the command invoked is for a plugin
// false otherwise
func IsPluginCommand(cmd *cobra.Command) bool {
	isPlugin := false

	for key, value := range cmd.Annotations {
		if key == "scope" && value == "plugin" {
			isPlugin = true
		}
	}

	return isPlugin
}
