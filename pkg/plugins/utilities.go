package plugins

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/BurntSushi/toml"

	hcplugin "github.com/hashicorp/go-plugin"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
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

// AddEntryToPluginManifest update plugins.toml with a new release version
func AddEntryToPluginManifest(ctx context.Context, config config.IConfig, fs afero.Fs, entry Plugin) error {
	color := ansi.Color(os.Stdout)
	currentPluginList, err := GetPluginList(ctx, config, fs)
	if err != nil {
		return nil
	}

	foundPlugin := false
	for i, plugin := range currentPluginList.Plugins {
		// already a plugin in the manfest with the same name, so use this instead of making a new one
		if plugin.Shortname == entry.Shortname {
			// plugin already installed. append a new release version
			foundPlugin = true

			for _, entryRelease := range entry.Releases {
				foundRelease := false
				for _, release := range plugin.Releases {
					if release.Version == entryRelease.Version {
						if release.Sum != entryRelease.Sum {
							return fmt.Errorf("release version '%s/%s/%s' is already installed but checksums do not match", release.Arch, release.OS, release.Version)
						}
						foundRelease = true
						break
					}
				}

				if !foundRelease {
					currentPluginList.Plugins[i].Releases = append(currentPluginList.Plugins[i].Releases, entryRelease)
				}
			}
		}
	}

	if !foundPlugin {
		// plugin does not exist. add a new plugin with a new dev release
		currentPluginList.Plugins = append(currentPluginList.Plugins, entry)
	}

	buf := new(bytes.Buffer)
	err = toml.NewEncoder(buf).Encode(currentPluginList)
	if err != nil {
		return err
	}

	configPath := config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginManifestPath := filepath.Join(configPath, "plugins.toml")
	err = os.WriteFile(pluginManifestPath, buf.Bytes(), 0644)
	if err != nil {
		return err
	}

	fmt.Println(color.Green(fmt.Sprintf("✔ updated '%s' with a release entry for 'stripe-cli-%s'", pluginManifestPath, entry.Shortname)))

	config.InitConfig()
	installedList := config.GetInstalledPlugins()

	// check for plugin already in list (ie. in the case of an upgrade)
	isInstalled := false
	for _, name := range installedList {
		if name == entry.Shortname {
			isInstalled = true
		}
	}

	if !isInstalled {
		installedList = append(installedList, entry.Shortname)
	}

	// sync list of installed plugins to file
	config.WriteConfigField("installed_plugins", installedList)

	return nil
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

// ExtractStdoutArchive extracts the archive from stdout
func ExtractStdoutArchive(ctx context.Context, config config.IConfig) error {
	gzf, err := gzip.NewReader(os.Stdin)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(gzf)
	err = extractAndInstall(ctx, config, tarReader)
	if err != nil {
		return err
	}

	return nil
}

// ExtractLocalArchive extracts the local tarball body
func ExtractLocalArchive(ctx context.Context, config config.IConfig, source string) error {
	color := ansi.Color(os.Stdout)
	fmt.Println(color.Yellow(fmt.Sprintf("extracting tarball at %s...", source)))

	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(gzf)
	err = extractAndInstall(ctx, config, tarReader)
	if err != nil {
		return err
	}

	return nil
}

// FetchAndExtractRemoteArchive fetches and extracts the remote tarball body
func FetchAndExtractRemoteArchive(ctx context.Context, config config.IConfig, url string) error {
	color := ansi.Color(os.Stdout)
	fmt.Println(color.Yellow(fmt.Sprintf("fetching tarball at %s...", url)))

	t := &requests.TracedTransport{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	trace := &httptrace.ClientTrace{
		GotConn: t.GotConn,
		DNSDone: t.DNSDone,
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	client := &http.Client{Transport: t}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	archive, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}

	defer archive.Close()

	tarReader := tar.NewReader(archive)
	err = extractAndInstall(ctx, config, tarReader)
	if err != nil {
		return err
	}

	return nil
}

// extractAndInstall extracts plugin tarball
func extractAndInstall(ctx context.Context, config config.IConfig, tarReader *tar.Reader) error {
	var manifest PluginList
	var pluginData []byte
	fs := afero.NewOsFs()
	color := ansi.Color(os.Stdout)
	extractedPluginName := ""

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			filename := filepath.Base(name)
			if strings.HasPrefix(filename, "._") {
				continue
			}

			if filename == "manifest.toml" {
				tomlBytes, _ := io.ReadAll(tarReader)
				err = toml.Unmarshal(tomlBytes, &manifest)
				if err != nil {
					return err
				}

				fmt.Println(color.Green(fmt.Sprintf("✔ extracted manifest '%s'", filename)))
			} else if strings.Contains(filename, "stripe-cli-") {
				extractedPluginName = filename
				pluginData, _ = io.ReadAll(tarReader)
				fmt.Println(color.Green(fmt.Sprintf("✔ extracted plugin '%s'", filename)))
			}

		default:
			return fmt.Errorf("unrecognized file type for file %s: %c", name, header.Typeflag)
		}
	}

	// update plugin manifest and config manifest
	if len(manifest.Plugins) == 1 && len(manifest.Plugins[0].Releases) >= 1 && len(pluginData) > 0 {
		plugin := manifest.Plugins[0]

		if extractedPluginName != plugin.Binary {
			return fmt.Errorf(
				"extracted plugin '%s' does not match the plugin '%s' in the manifest",
				extractedPluginName,
				plugin.Shortname)
		}

		err := AddEntryToPluginManifest(ctx, config, fs, plugin)
		if err != nil {
			return err
		}

		version := plugin.Releases[0].Version
		err = plugin.verifychecksumAndSavePlugin(pluginData, config, fs, version)
		if err != nil {
			return err
		}

		// clean up other plugin versions
		plugin.cleanUpPluginPath(config, fs, version)
	} else {
		return fmt.Errorf("missing required manifest.toml or plugin in the archive")
	}

	return nil
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
