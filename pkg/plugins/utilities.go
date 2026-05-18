package plugins

import (
	"bytes"
	"context"
	"errors"
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
	"github.com/hashicorp/go-version"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/fsutil"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type installedPluginStateSnapshot struct {
	installedPlugins []string
	localMetadata    []byte
	hasLocalMetadata bool
}

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

func getLocalPluginMetadataDir(config config.IConfig) string {
	configPath := config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	return filepath.Join(configPath, "plugin-metadata")
}

func getLocalPluginMetadataPath(config config.IConfig, pluginName string) string {
	return filepath.Join(getLocalPluginMetadataDir(config), pluginName+".toml")
}

func snapshotInstalledPluginState(config config.IConfig, fs afero.Fs, pluginName string) (installedPluginStateSnapshot, error) {
	snapshot := installedPluginStateSnapshot{
		installedPlugins: append([]string(nil), config.GetInstalledPlugins()...),
	}

	body, err := afero.ReadFile(fs, getLocalPluginMetadataPath(config, pluginName))
	if err != nil {
		if os.IsNotExist(err) {
			return snapshot, nil
		}
		return installedPluginStateSnapshot{}, err
	}

	snapshot.hasLocalMetadata = true
	snapshot.localMetadata = append([]byte(nil), body...)
	return snapshot, nil
}

func rollbackInstalledPluginState(config config.IConfig, fs afero.Fs, pluginName string, snapshot installedPluginStateSnapshot) error {
	rollbackErrors := make([]string, 0, 2)

	if err := restoreLocalPluginMetadata(config, fs, pluginName, snapshot); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Sprintf("restore local plugin metadata: %v", err))
	}

	if err := restoreInstalledPluginList(config, snapshot.installedPlugins); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Sprintf("restore installed_plugins: %v", err))
	}

	if len(rollbackErrors) != 0 {
		return errors.New(strings.Join(rollbackErrors, "; "))
	}

	return nil
}

func restoreLocalPluginMetadata(config config.IConfig, fs afero.Fs, pluginName string, snapshot installedPluginStateSnapshot) error {
	if snapshot.hasLocalMetadata {
		return afero.WriteFile(fs, getLocalPluginMetadataPath(config, pluginName), snapshot.localMetadata, 0644)
	}

	return removeLocalPluginMetadata(config, fs, pluginName)
}

func restoreInstalledPluginList(config config.IConfig, installedPlugins []string) error {
	if stringSlicesEqual(config.GetInstalledPlugins(), installedPlugins) {
		return nil
	}

	err := config.WriteConfigField("installed_plugins", installedPlugins)
	if err != nil && !stringSlicesEqual(config.GetInstalledPlugins(), installedPlugins) {
		return err
	}

	return nil
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}

// GetInstalledPluginNames returns the union of plugin names recorded in config
// and plugin names with persisted local metadata.
func GetInstalledPluginNames(config config.IConfig, fs afero.Fs) ([]string, error) {
	names := make([]string, 0)
	seen := make(map[string]struct{})

	for _, pluginName := range config.GetInstalledPlugins() {
		if pluginName == "" {
			continue
		}
		if _, exists := seen[pluginName]; exists {
			continue
		}
		seen[pluginName] = struct{}{}
		names = append(names, pluginName)
	}

	localMetadataNames, err := getLocalPluginMetadataNames(config, fs)
	if err != nil {
		return names, err
	}

	for _, pluginName := range localMetadataNames {
		if _, exists := seen[pluginName]; exists {
			continue
		}
		seen[pluginName] = struct{}{}
		names = append(names, pluginName)
	}

	return names, nil
}

// RecordInstalledPlugin ensures a plugin name is persisted in installed_plugins.
func RecordInstalledPlugin(config config.IConfig, pluginName string) error {
	if pluginName == "" {
		return nil
	}

	installedPlugins := config.GetInstalledPlugins()
	for _, installedPlugin := range installedPlugins {
		if installedPlugin == pluginName {
			return nil
		}
	}

	installedPlugins = append(installedPlugins, pluginName)
	return config.WriteConfigField("installed_plugins", installedPlugins)
}

// RemoveInstalledPlugin removes a plugin name from installed_plugins if present.
func RemoveInstalledPlugin(config config.IConfig, pluginName string) error {
	if pluginName == "" {
		return nil
	}

	installedPlugins := config.GetInstalledPlugins()
	updatedPlugins := make([]string, 0, len(installedPlugins))
	removed := false
	for _, installedPlugin := range installedPlugins {
		if installedPlugin == pluginName {
			removed = true
			continue
		}
		updatedPlugins = append(updatedPlugins, installedPlugin)
	}

	if !removed {
		return nil
	}

	return config.WriteConfigField("installed_plugins", updatedPlugins)
}

// PersistInstalledPluginState ensures local metadata and installed_plugins are
// both updated for a locally installed plugin.
func PersistInstalledPluginState(config config.IConfig, fs afero.Fs, plugin Plugin) error {
	if plugin.Shortname == "" {
		return nil
	}

	previousState, err := snapshotInstalledPluginState(config, fs, plugin.Shortname)
	if err != nil {
		return err
	}

	if err := writeLocalPluginMetadata(config, fs, plugin); err != nil {
		if rollbackErr := rollbackInstalledPluginState(config, fs, plugin.Shortname, previousState); rollbackErr != nil {
			return fmt.Errorf("failed to write local plugin metadata: %w; rollback failed: %v", err, rollbackErr)
		}
		return err
	}

	if err := RecordInstalledPlugin(config, plugin.Shortname); err != nil {
		if rollbackErr := rollbackInstalledPluginState(config, fs, plugin.Shortname, previousState); rollbackErr != nil {
			return fmt.Errorf("failed to record installed plugin %s: %w; rollback failed: %v", plugin.Shortname, err, rollbackErr)
		}
		return err
	}

	return nil
}

// GetPluginList builds a list of allowed plugins to be installed and run by the CLI
func GetPluginList(ctx context.Context, config config.IConfig, fs afero.Fs) (PluginList, error) {
	pluginList, err := getCachedPluginList(config, fs)
	if os.IsNotExist(err) {
		log.Debug("The plugin manifest file does not exist. Downloading...")
		err = RefreshPluginManifest(ctx, config, fs, stripe.DefaultAPIBaseURL)
		if err != nil {
			log.Debug("Could not download plugin manifest")
			return pluginList, err
		}
		return getCachedPluginList(config, fs)
	}

	if err != nil {
		return pluginList, err
	}

	return pluginList, nil
}

// LookUpPlugin returns the matching plugin object
func LookUpPlugin(ctx context.Context, config config.IConfig, fs afero.Fs, pluginName string) (Plugin, error) {
	plugin, err := readLocalPluginMetadata(config, fs, pluginName)
	if err == nil {
		return plugin, nil
	}

	if !os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"prefix": "plugins.LookUpPlugin",
			"plugin": pluginName,
		}).Debugf("could not read local plugin metadata, falling back to manifest lookup: %s", err)
	}

	return LookUpPluginInManifest(ctx, config, fs, pluginName)
}

// LookUpPluginInManifest returns plugin metadata from the cached global manifest.
func LookUpPluginInManifest(ctx context.Context, config config.IConfig, fs afero.Fs, pluginName string) (Plugin, error) {
	pluginList, err := GetPluginList(ctx, config, fs)
	if err != nil {
		return Plugin{}, err
	}

	return findPlugin(pluginList, pluginName)
}

// ResolvePluginForInstall resolves the plugin metadata needed by `stripe plugin install`
// without requiring a manifest refresh on the metadata-backed path.
func ResolvePluginForInstall(ctx context.Context, config config.IConfig, fs afero.Fs, pluginName, version, baseURL string) (*Plugin, string, error) {
	apiKey, err := config.GetProfile().GetAPIKey(false)
	if err != nil && !errors.Is(err, validators.ErrAPIKeyNotConfigured) {
		return nil, "", err
	}

	if apiKey != "" {
		plugin, resolvedVersion, err := resolvePluginFromMetadata(ctx, config, fs, pluginName, version, baseURL, apiKey)
		if err == nil {
			return plugin, resolvedVersion, nil
		}

		log.WithFields(log.Fields{
			"prefix":  "plugins.ResolvePluginForInstall",
			"plugin":  pluginName,
			"version": version,
		}).Debugf("could not resolve plugin via metadata, falling back to manifest lookup: %s", err)
	}

	if err := RefreshPluginManifest(ctx, config, fs, baseURL); err != nil {
		return nil, "", err
	}

	plugin, err := LookUpPluginInManifest(ctx, config, fs, pluginName)
	if err != nil {
		return nil, "", err
	}

	resolvedVersion := version
	if resolvedVersion == "" {
		resolvedVersion = plugin.LookUpLatestVersion()
	}
	if resolvedVersion == "" {
		return nil, "", fmt.Errorf("could not determine latest version for plugin %s", pluginName)
	}

	return &plugin, resolvedVersion, nil
}

// ResolvePluginForUpgrade resolves plugin metadata for `stripe plugin upgrade`
// using persisted local metadata and the cached global manifest.
func ResolvePluginForUpgrade(config config.IConfig, fs afero.Fs, pluginName string) (*Plugin, error) {
	var localPlugin *Plugin
	localPluginValue, localErr := readLocalPluginMetadata(config, fs, pluginName)
	if localErr == nil {
		localPlugin = &localPluginValue
	} else if !os.IsNotExist(localErr) {
		log.WithFields(log.Fields{
			"prefix": "plugins.ResolvePluginForUpgrade",
			"plugin": pluginName,
		}).Debugf("could not read local plugin metadata for upgrade: %s", localErr)
	}

	var manifestPlugin *Plugin
	manifestPluginValue, manifestErr := lookUpPluginInCachedManifest(config, fs, pluginName)
	if manifestErr == nil {
		manifestPlugin = &manifestPluginValue
	}

	plugin := selectPluginForUpgrade(localPlugin, manifestPlugin)
	if plugin != nil {
		return plugin, nil
	}

	if localErr != nil && !os.IsNotExist(localErr) {
		return nil, localErr
	}

	return nil, manifestErr
}

// resolvePluginForAutoInstall resolves the freshest plugin metadata to use when
// a command is invoked but the local plugin binary is missing.
func resolvePluginForAutoInstall(ctx context.Context, config config.IConfig, fs afero.Fs, pluginName, baseURL string) (*Plugin, string, error) {
	plugin, version, err := ResolvePluginForInstall(ctx, config, fs, pluginName, "", baseURL)
	if err == nil {
		return plugin, version, nil
	}

	log.WithFields(log.Fields{
		"prefix": "plugins.resolvePluginForAutoInstall",
		"plugin": pluginName,
	}).Debugf("could not resolve latest plugin metadata for auto-install, falling back to cached metadata: %s", err)

	plugin, cachedErr := ResolvePluginForUpgrade(config, fs, pluginName)
	if cachedErr != nil {
		return nil, "", fmt.Errorf("could not resolve plugin %s for auto-install: latest lookup failed: %v; cached lookup failed: %w", pluginName, err, cachedErr)
	}

	version = plugin.LookUpLatestVersion()
	if version == "" {
		return nil, "", fmt.Errorf("could not determine latest version for plugin %s", pluginName)
	}

	return plugin, version, nil
}

func getCachedPluginList(config config.IConfig, fs afero.Fs) (PluginList, error) {
	var pluginList PluginList
	configPath := config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginManifestPath := filepath.Join(configPath, "plugins.toml")

	file, err := afero.ReadFile(fs, pluginManifestPath)
	if err != nil {
		return pluginList, err
	}

	_, err = toml.Decode(string(file), &pluginList)
	if err != nil {
		return pluginList, err
	}

	return pluginList, nil
}

func findPlugin(pluginList PluginList, pluginName string) (Plugin, error) {
	for _, p := range pluginList.Plugins {
		if pluginName == p.Shortname {
			return p, nil
		}
	}

	return Plugin{}, fmt.Errorf("could not find a plugin named %s", pluginName)
}

func selectPluginForUpgrade(localPlugin, manifestPlugin *Plugin) *Plugin {
	switch {
	case localPlugin == nil:
		return mergePluginMetadata(manifestPlugin, nil)
	case manifestPlugin == nil:
		return mergePluginMetadata(localPlugin, nil)
	case comparePluginVersions(localPlugin.LookUpLatestVersion(), manifestPlugin.LookUpLatestVersion()) >= 0:
		return mergePluginMetadata(localPlugin, manifestPlugin)
	default:
		return mergePluginMetadata(manifestPlugin, localPlugin)
	}
}

func comparePluginVersions(left, right string) int {
	switch {
	case left == "" && right == "":
		return 0
	case left == "":
		return -1
	case right == "":
		return 1
	}

	leftVersion, leftErr := version.NewVersion(left)
	rightVersion, rightErr := version.NewVersion(right)
	if leftErr == nil && rightErr == nil {
		switch {
		case leftVersion.GreaterThan(rightVersion):
			return 1
		case leftVersion.LessThan(rightVersion):
			return -1
		default:
			return 0
		}
	}

	switch {
	case left > right:
		return 1
	case left < right:
		return -1
	default:
		return 0
	}
}

func mergePluginMetadata(primary, fallback *Plugin) *Plugin {
	if primary == nil {
		if fallback == nil {
			return nil
		}
		pluginCopy := *fallback
		return &pluginCopy
	}

	pluginCopy := *primary
	if fallback == nil {
		return &pluginCopy
	}

	if pluginCopy.Shortdesc == "" {
		pluginCopy.Shortdesc = fallback.Shortdesc
	}
	if pluginCopy.Binary == "" {
		pluginCopy.Binary = fallback.Binary
	}
	if pluginCopy.MagicCookieValue == "" {
		pluginCopy.MagicCookieValue = fallback.MagicCookieValue
	}
	if len(pluginCopy.Commands) == 0 && len(fallback.Commands) > 0 {
		pluginCopy.Commands = fallback.Commands
	}

	for i := range pluginCopy.Releases {
		if len(pluginCopy.Releases[i].Runtime) != 0 {
			continue
		}

		fallbackRelease := fallback.getRelease(pluginCopy.Releases[i].Version, pluginCopy.Releases[i].OS, pluginCopy.Releases[i].Arch)
		if fallbackRelease == nil || len(fallbackRelease.Runtime) == 0 {
			continue
		}

		pluginCopy.Releases[i].Runtime = copyRuntime(fallbackRelease.Runtime)
	}

	return &pluginCopy
}

func resolvePluginFromMetadata(ctx context.Context, config config.IConfig, fs afero.Fs, pluginName, version, baseURL, apiKey string) (*Plugin, string, error) {
	basePlugin := &Plugin{Shortname: pluginName}
	if cachedPlugin, err := readLocalPluginMetadata(config, fs, pluginName); err == nil {
		basePlugin = &cachedPlugin
	} else if cachedPlugin, err := lookUpPluginInCachedManifest(config, fs, pluginName); err == nil {
		basePlugin = &cachedPlugin
	}

	pluginMetadata, err := requests.GetPluginMetadata(ctx, baseURL, stripe.APIVersion, apiKey, config.GetProfile(), pluginName, version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, "", err
	}

	plugin, err := basePlugin.pluginFromMetadata(pluginMetadata.PluginManifest)
	if err != nil {
		return nil, "", err
	}

	resolvedVersion := version
	if resolvedVersion == "" {
		resolvedVersion = plugin.LookUpLatestVersion()
	}
	if resolvedVersion == "" {
		return nil, "", fmt.Errorf("plugin metadata response did not include a release for %s on %s/%s", pluginName, runtime.GOOS, runtime.GOARCH)
	}
	if plugin.getReleaseForVersion(resolvedVersion) == nil {
		return nil, "", fmt.Errorf("plugin metadata response did not include plugin %s version %s for %s/%s", pluginName, resolvedVersion, runtime.GOOS, runtime.GOARCH)
	}

	return plugin, resolvedVersion, nil
}

func lookUpPluginInCachedManifest(config config.IConfig, fs afero.Fs, pluginName string) (Plugin, error) {
	pluginList, err := getCachedPluginList(config, fs)
	if err != nil {
		return Plugin{}, err
	}

	return findPlugin(pluginList, pluginName)
}

func readLocalPluginMetadata(config config.IConfig, fs afero.Fs, pluginName string) (Plugin, error) {
	body, err := afero.ReadFile(fs, getLocalPluginMetadataPath(config, pluginName))
	if err != nil {
		return Plugin{}, err
	}

	pluginList, err := validatePluginManifest(body)
	if err != nil {
		return Plugin{}, err
	}

	return findPlugin(*pluginList, pluginName)
}

func writeLocalPluginMetadata(config config.IConfig, fs afero.Fs, plugin Plugin) error {
	pluginMetadataDir := getLocalPluginMetadataDir(config)
	if err := fs.MkdirAll(pluginMetadataDir, 0755); err != nil {
		return err
	}

	body := new(bytes.Buffer)
	if err := toml.NewEncoder(body).Encode(PluginList{Plugins: []Plugin{plugin}}); err != nil {
		return err
	}

	return afero.WriteFile(fs, getLocalPluginMetadataPath(config, plugin.Shortname), body.Bytes(), 0644)
}

func removeLocalPluginMetadata(config config.IConfig, fs afero.Fs, pluginName string) error {
	err := fs.Remove(getLocalPluginMetadataPath(config, pluginName))
	if os.IsNotExist(err) {
		return nil
	}

	return err
}

func getLocalPluginMetadataNames(config config.IConfig, fs afero.Fs) ([]string, error) {
	entries, err := afero.ReadDir(fs, getLocalPluginMetadataDir(config))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".toml"))
	}
	sort.Strings(names)

	return names, nil
}

// RefreshPluginManifest refreshes the plugin manifest
func RefreshPluginManifest(ctx context.Context, config config.IConfig, fs afero.Fs, baseURL string) error {
	apiKey, err := config.GetProfile().GetAPIKey(false)

	if err != nil {
		if err != validators.ErrAPIKeyNotConfigured {
			return err
		}
		// If the API key is not configured, that's fine, continue with the fallback plugin data
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

	if err := fsutil.RefuseWriteThroughSymlink(fs, pluginManifestPath, filepath.Dir(configPath), filepath.Base(pluginManifestPath)); err != nil {
		return err
	}

	// Ensure the config directory exists
	err = fs.MkdirAll(configPath, os.FileMode(0755))
	if err != nil {
		return err
	}

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
			var remoteResourceNotFoundError *remoteResourceNotFoundError
			if errors.As(err, &remoteResourceNotFoundError) {
				log.Debugf("Additional plugin manifest not found, silently skipping: url=%s", remoteResourceNotFoundError.URL)
				continue
			}
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

// validateRuntimeVersions validates that Runtime specifications only contain valid LTS Node.js versions
func validateRuntimeVersions(pluginList *PluginList) error {
	for _, plugin := range pluginList.Plugins {
		for _, release := range plugin.Releases {
			if err := validateReleaseRuntimes(plugin.Shortname, release); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateReleaseRuntimes validates the runtime specifications for a single release
func validateReleaseRuntimes(pluginName string, release Release) error {
	// Skip releases without runtime requirements
	if release.Runtime == nil {
		return nil
	}

	// Validate each runtime specification
	for runtime, version := range release.Runtime {
		// Only validate Node.js versions (skip other runtimes)
		if runtime != "node" {
			continue
		}

		// Check if the Node.js version is valid
		if !isValidNodeLTSVersion(version) {
			return fmt.Errorf(
				"invalid Node.js version '%s' for plugin '%s' version '%s'. Only LTS major versions are allowed (18, 20, 22, 24, etc.)",
				version,
				pluginName,
				release.Version,
			)
		}
	}

	return nil
}

// isValidNodeLTSVersion checks if a Node.js version string is a valid LTS major version
// Valid LTS versions are even-numbered major versions starting from 18
func isValidNodeLTSVersion(version string) bool {
	// Empty string is invalid
	if version == "" {
		return false
	}

	// Parse the version as an integer - must be a valid integer string
	var majorVersion int
	n, err := fmt.Sscanf(version, "%d", &majorVersion)
	if err != nil || n != 1 {
		return false
	}

	// Verify the parsed integer matches the original string (no extra characters)
	// This ensures "20.0" or "v20" etc. are rejected
	if fmt.Sprintf("%d", majorVersion) != version {
		return false
	}

	if majorVersion < 18 {
		return false
	}

	return majorVersion%2 == 0
}

func validatePluginManifest(body []byte) (*PluginList, error) {
	var manifestBody PluginList

	if err := toml.Unmarshal(body, &manifestBody); err != nil {
		return nil, fmt.Errorf("received an invalid plugin manifest: %s", err)
	}
	if len(manifestBody.Plugins) == 0 {
		return nil, fmt.Errorf("received an empty plugin manifest")
	}
	if err := validateRuntimeVersions(&manifestBody); err != nil {
		return nil, err
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
			vi, errI := version.NewVersion(pluginList.Plugins[idx].Releases[i].Version)
			vj, errJ := version.NewVersion(pluginList.Plugins[idx].Releases[j].Version)

			// If either version fails to parse, fall back to string comparison
			if errI != nil || errJ != nil {
				return pluginList.Plugins[idx].Releases[i].Version < pluginList.Plugins[idx].Releases[j].Version
			}

			return vi.LessThan(vj)
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

type remoteResourceNotFoundError struct {
	URL string
}

func (e *remoteResourceNotFoundError) Error() string {
	return fmt.Sprintf("remote resource not found: url=%s", e.URL)
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
		if strings.Contains(err.Error(), "no such host") {
			return nil, fmt.Errorf("failed to find the plugin repository. Make sure you are on the latest version of the Stripe CLI: https://docs.stripe.com/stripe-cli/upgrade")
		}
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &remoteResourceNotFoundError{URL: url}
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
