package plugins

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/BurntSushi/toml"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

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

	return plugin, fmt.Errorf("could not find a plugin named %s", pluginName)
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

	pluginManifestURL := fmt.Sprintf("%s/%s", pluginData.PluginBaseURL, "plugins.toml")
	body, err := FetchRemoteResource(pluginManifestURL)
	if err != nil {
		return err
	}

	configPath := config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginManifestPath := filepath.Join(configPath, "plugins.toml")

	err = afero.WriteFile(fs, pluginManifestPath, body, 0644)

	if err != nil {
		return err
	}

	return nil
}

// FetchRemoteResource returns the remote resource body
func FetchRemoteResource(url string) ([]byte, error) {
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}
