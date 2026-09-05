package plugins

import (
	"bytes"
	"context"
	"encoding/json"
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
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/BurntSushi/toml"

	hcplugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-version"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type installedPluginStateSnapshot struct {
	installedPlugins []string
	localMetadata    []byte
	hasLocalMetadata bool
}

// ResolvedPluginVersion contains the resolved plugin metadata needed to
// install a specific plugin version, including a resolved download URL when
// the metadata endpoint already returned one.
type ResolvedPluginVersion struct {
	Plugin    *Plugin
	Version   string
	BinaryURL string
	// AutoInstall carries the metadata endpoint's answer to "may this machine
	// install this plugin without being asked?". It is only ever true for a
	// resolution that came from a live metadata response: a resolution that fell
	// back to cached metadata leaves it false, so a machine that cannot reach the
	// endpoint keeps prompting rather than installing on a stale answer.
	AutoInstall bool
}

// checkLatestPluginVersionResolver is swappable for test injection.
var checkLatestPluginVersionResolver = ResolvePluginForUpgrade

// checkLatestPluginVersionTimeout bounds the best-effort upgrade hint lookup so
// plugin commands do not hang on exit when the metadata endpoint is slow.
var checkLatestPluginVersionTimeout = 2 * time.Second

// ValidatePluginShortname rejects names that could escape the plugin install or
// metadata directories when joined onto local filesystem paths.
func ValidatePluginShortname(pluginName string) error {
	switch {
	case pluginName == "":
		return errorcategory.New(errorcategory.UserInput, "plugin name cannot be empty")
	case pluginName == "." || pluginName == "..":
		return errorcategory.Errorf(errorcategory.UserInput, "invalid plugin name %q", pluginName)
	case filepath.IsAbs(pluginName):
		return errorcategory.Errorf(errorcategory.UserInput, "invalid plugin name %q", pluginName)
	case strings.ContainsAny(pluginName, `/\`):
		return errorcategory.Errorf(errorcategory.UserInput, "invalid plugin name %q", pluginName)
	case filepath.Clean(pluginName) != pluginName:
		return errorcategory.Errorf(errorcategory.UserInput, "invalid plugin name %q", pluginName)
	case filepath.Base(pluginName) != pluginName:
		return errorcategory.Errorf(errorcategory.UserInput, "invalid plugin name %q", pluginName)
	default:
		return nil
	}
}

// ValidatePluginBinaryName rejects binary names from the server manifest that
// could escape the plugin install directory when joined onto local filesystem
// paths.
func ValidatePluginBinaryName(binaryName string) error {
	switch {
	case binaryName == "":
		return errors.New("plugin binary name cannot be empty")
	case binaryName == "." || binaryName == "..":
		return fmt.Errorf("invalid plugin binary name %q", binaryName)
	case filepath.IsAbs(binaryName):
		return fmt.Errorf("invalid plugin binary name %q", binaryName)
	case strings.ContainsAny(binaryName, `/\`):
		return fmt.Errorf("invalid plugin binary name %q", binaryName)
	case filepath.Clean(binaryName) != binaryName:
		return fmt.Errorf("invalid plugin binary name %q", binaryName)
	case filepath.Base(binaryName) != binaryName:
		return fmt.Errorf("invalid plugin binary name %q", binaryName)
	default:
		return nil
	}
}

// Install installs the resolved plugin version. If the metadata lookup already
// resolved a concrete binary URL, it reuses that result and skips a second
// metadata request. Otherwise it retries metadata during install so cached
// local metadata can still recover fresh release details.
func (r *ResolvedPluginVersion) Install(ctx context.Context, config config.IConfig, fs afero.Fs, apiBaseURL, dashboardBaseURL string) error {
	switch {
	case r == nil:
		return errorcategory.New(errorcategory.Internal, "missing resolved plugin version")
	case r.Plugin == nil:
		return errorcategory.New(errorcategory.Internal, "missing plugin metadata")
	case r.Version == "":
		return errorcategory.New(errorcategory.Internal, "missing plugin version")
	default:
		return r.Plugin.install(ctx, config, fs, r.Version, apiBaseURL, dashboardBaseURL, r.BinaryURL, r.BinaryURL != "")
	}
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

func getLocalPluginMetadataPath(config config.IConfig, pluginName string) (string, error) {
	if err := ValidatePluginShortname(pluginName); err != nil {
		return "", err
	}

	return filepath.Join(getLocalPluginMetadataDir(config), pluginName+".toml"), nil
}

func getCachedPluginManifestPath(config config.IConfig) string {
	configPath := config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	return filepath.Join(configPath, "plugins.toml")
}

func snapshotInstalledPluginState(config config.IConfig, fs afero.Fs, pluginName string) (installedPluginStateSnapshot, error) {
	snapshot := installedPluginStateSnapshot{
		installedPlugins: append([]string(nil), config.GetInstalledPlugins()...),
	}

	metadataPath, err := getLocalPluginMetadataPath(config, pluginName)
	if err != nil {
		return installedPluginStateSnapshot{}, err
	}

	body, err := afero.ReadFile(fs, metadataPath)
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
		return errorcategory.New(errorcategory.Internal, strings.Join(rollbackErrors, "; "))
	}

	return nil
}

func restoreLocalPluginMetadata(config config.IConfig, fs afero.Fs, pluginName string, snapshot installedPluginStateSnapshot) error {
	if snapshot.hasLocalMetadata {
		metadataPath, err := getLocalPluginMetadataPath(config, pluginName)
		if err != nil {
			return err
		}

		return afero.WriteFile(fs, metadataPath, snapshot.localMetadata, 0644)
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
	if err := ValidatePluginShortname(plugin.Shortname); err != nil {
		return err
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

// ListPlugins fetches the live plugin list visible to the current caller for
// the current platform using the list-plugins API endpoints.
func ListPlugins(ctx context.Context, config config.IConfig, apiBaseURL, dashboardBaseURL string) (PluginList, error) {
	creds, err := config.GetProfile().ResolveCredentialsForAnyMode(false)
	if err != nil && !errors.Is(err, validators.ErrAPIKeyNotConfigured) {
		return PluginList{}, err
	}
	apiKey := creds.Token

	if dashboardBaseURL == "" {
		dashboardBaseURL = stripe.DashboardBaseURLForAPIBaseURL(apiBaseURL)
	}

	body, err := requests.GetPluginList(
		ctx,
		apiBaseURL,
		dashboardBaseURL,
		stripe.APIVersion,
		apiKey,
		config.GetProfile(),
		runtime.GOOS,
		runtime.GOARCH,
	)
	if err != nil {
		return PluginList{}, err
	}

	var pluginList PluginList
	if err := json.Unmarshal(body, &pluginList); err != nil {
		return PluginList{}, fmt.Errorf("failed to decode plugin list response: %w", err)
	}

	if err := validatePluginListResponse(&pluginList); err != nil {
		return PluginList{}, err
	}

	return pluginList, nil
}

// BackfillMissingInstalledPluginMetadata refreshes local metadata for plugins
// that were installed before `plugin-metadata/*.toml` became the source of truth.
// It first migrates from the legacy cached `plugins.toml` on disk when that
// cache still describes the installed version, then falls back to the live
// metadata endpoint for plugins not present in that cache.
// Failures are best-effort: a failed backfill should not prevent existing
// plugin commands from being registered via the same cached-manifest fallback.
func BackfillMissingInstalledPluginMetadata(ctx context.Context, config config.IConfig, fs afero.Fs, apiBaseURL, dashboardBaseURL string) error {
	if dashboardBaseURL == "" {
		dashboardBaseURL = stripe.DashboardBaseURLForAPIBaseURL(apiBaseURL)
	}

	creds, err := config.GetProfile().ResolveCredentialsForAnyMode(false)
	if err != nil && !errors.Is(err, validators.ErrAPIKeyNotConfigured) {
		return err
	}
	apiKey := creds.Token

	pluginNames, err := GetInstalledPluginNames(config, fs)
	if err != nil {
		return err
	}

	for _, pluginName := range pluginNames {
		if pluginName == "" {
			continue
		}

		if err := ValidatePluginShortname(pluginName); err != nil {
			log.WithFields(log.Fields{
				"prefix": "plugins.BackfillMissingInstalledPluginMetadata",
				"plugin": pluginName,
			}).Debugf("skipping invalid plugin name during backfill: %s", err)
			continue
		}

		if _, err := readLocalPluginMetadata(config, fs, pluginName); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			log.WithFields(log.Fields{
				"prefix": "plugins.BackfillMissingInstalledPluginMetadata",
				"plugin": pluginName,
			}).Debugf("could not read local plugin metadata before backfill: %s", err)
			continue
		}

		installedVersion, err := (&Plugin{Shortname: pluginName}).lookUpInstalledVersion(config, fs)
		if err != nil {
			log.WithFields(log.Fields{
				"prefix": "plugins.BackfillMissingInstalledPluginMetadata",
				"plugin": pluginName,
			}).Debugf("could not determine installed plugin version for backfill: %s", err)
			continue
		}
		if installedVersion == "" || isLocalDevelopmentVersion(installedVersion) {
			continue
		}

		manifestPlugin, manifestErr := lookUpPluginInCachedManifest(config, fs, pluginName)
		if manifestErr == nil {
			if manifestPlugin.getReleaseForVersion(installedVersion) != nil {
				if err := PersistInstalledPluginState(config, fs, manifestPlugin); err != nil {
					log.WithFields(log.Fields{
						"prefix": "plugins.BackfillMissingInstalledPluginMetadata",
						"plugin": pluginName,
					}).Debugf("could not persist backfilled plugin metadata from cached manifest: %s", err)
				}
				continue
			}

			log.WithFields(log.Fields{
				"prefix":  "plugins.BackfillMissingInstalledPluginMetadata",
				"plugin":  pluginName,
				"version": installedVersion,
			}).Debug("cached manifest did not include the installed version; falling back to network metadata")
		}

		var pluginNotFound *ErrPluginNotFound
		if manifestErr != nil && !os.IsNotExist(manifestErr) && !errors.As(manifestErr, &pluginNotFound) {
			log.WithFields(log.Fields{
				"prefix": "plugins.BackfillMissingInstalledPluginMetadata",
				"plugin": pluginName,
			}).Debugf("could not read cached manifest plugin metadata before network fallback: %s", manifestErr)
		}

		resolvedPlugin, err := resolvePluginFromMetadata(ctx, config, fs, pluginName, installedVersion, apiBaseURL, dashboardBaseURL, apiKey)
		if err != nil {
			log.WithFields(log.Fields{
				"prefix":  "plugins.BackfillMissingInstalledPluginMetadata",
				"plugin":  pluginName,
				"version": installedVersion,
			}).Debugf("could not backfill local plugin metadata: %s", err)
			continue
		}

		if err := PersistInstalledPluginState(config, fs, *resolvedPlugin.Plugin); err != nil {
			log.WithFields(log.Fields{
				"prefix":  "plugins.BackfillMissingInstalledPluginMetadata",
				"plugin":  pluginName,
				"version": installedVersion,
			}).Debugf("could not persist backfilled local plugin metadata: %s", err)
		}
	}

	return nil
}

// LookUpPlugin returns persisted local metadata for an installed plugin,
// falling back to the legacy cached manifest during the compatibility window.
func LookUpPlugin(_ context.Context, config config.IConfig, fs afero.Fs, pluginName string) (Plugin, error) {
	plugin, err := readLocalPluginMetadata(config, fs, pluginName)
	if err == nil {
		return plugin, nil
	}

	if !os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"prefix": "plugins.LookUpPlugin",
			"plugin": pluginName,
		}).Debugf("could not read local plugin metadata, falling back to cached manifest: %s", err)
	}

	plugin, manifestErr := lookUpPluginInCachedManifest(config, fs, pluginName)
	if manifestErr == nil {
		return plugin, nil
	}

	var pluginNotFound *ErrPluginNotFound
	switch {
	case errors.As(manifestErr, &pluginNotFound):
		return Plugin{}, pluginNotFound
	case !os.IsNotExist(err):
		return Plugin{}, err
	case os.IsNotExist(manifestErr):
		return Plugin{}, &ErrPluginNotFound{Name: pluginName}
	default:
		return Plugin{}, manifestErr
	}
}

// ResolvePluginForInstall resolves the plugin metadata needed by `stripe plugin install`
// using the metadata endpoint first and cached plugin metadata as fallback.
func ResolvePluginForInstall(ctx context.Context, config config.IConfig, fs afero.Fs, pluginName, version, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
	if err := ValidatePluginShortname(pluginName); err != nil {
		return nil, err
	}

	creds, err := config.GetProfile().ResolveCredentialsForAnyMode(false)
	if err != nil && !errors.Is(err, validators.ErrAPIKeyNotConfigured) {
		return nil, err
	}
	apiKey := creds.Token

	resolvedPlugin, err := resolvePluginFromMetadata(ctx, config, fs, pluginName, version, apiBaseURL, dashboardBaseURL, apiKey)
	if err == nil {
		return resolvedPlugin, nil
	}

	if requiresNewerCLI(err) {
		return nil, err
	}

	log.WithFields(log.Fields{
		"prefix":  "plugins.ResolvePluginForInstall",
		"plugin":  pluginName,
		"version": version,
	}).Debugf("could not resolve plugin via plugin metadata endpoint, falling back to cached plugin metadata: %s", err)

	cachedPlugin, cachedErr := resolveCachedPluginForInstall(config, fs, pluginName, version)
	if cachedErr == nil {
		resolvedVersion, versionErr := resolveVersionFromReleases(cachedPlugin, pluginName, version, cachedPluginMetadata)
		if versionErr != nil {
			return nil, versionErr
		}

		return &ResolvedPluginVersion{
			Plugin:  cachedPlugin,
			Version: resolvedVersion,
		}, nil
	}

	if normalizedErr := normalizePluginMetadataError(pluginName, err); normalizedErr != nil && os.IsNotExist(cachedErr) {
		return nil, normalizedErr
	}

	return nil, fmt.Errorf("could not resolve plugin %s via plugin metadata endpoint: %v; cached lookup failed: %w", pluginName, err, cachedErr)
}

// ResolvePluginForUpgrade resolves the latest plugin metadata for
// `stripe plugin upgrade` using the plugin metadata endpoint first and cached
// plugin metadata as fallback.
func ResolvePluginForUpgrade(ctx context.Context, config config.IConfig, fs afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
	if err := ValidatePluginShortname(pluginName); err != nil {
		return nil, err
	}

	creds, err := config.GetProfile().ResolveCredentialsForAnyMode(false)
	if err != nil && !errors.Is(err, validators.ErrAPIKeyNotConfigured) {
		return nil, err
	}
	apiKey := creds.Token

	resolvedPlugin, endpointErr := resolvePluginFromMetadata(ctx, config, fs, pluginName, "", apiBaseURL, dashboardBaseURL, apiKey)
	if endpointErr == nil {
		return resolvedPlugin, nil
	}

	if requiresNewerCLI(endpointErr) {
		return nil, endpointErr
	}

	log.WithFields(log.Fields{
		"prefix": "plugins.ResolvePluginForUpgrade",
		"plugin": pluginName,
	}).Debugf("could not resolve latest plugin via plugin metadata endpoint, falling back to cached plugin metadata: %s", endpointErr)

	cachedPlugin, cachedErr := resolveCachedPluginForUpgrade(config, fs, pluginName)
	if cachedErr == nil {
		version, versionErr := resolveVersionFromReleases(cachedPlugin, pluginName, "", cachedPluginMetadata)
		if versionErr != nil {
			return nil, versionErr
		}

		return &ResolvedPluginVersion{
			Plugin:  cachedPlugin,
			Version: version,
		}, nil
	}

	if normalizedErr := normalizePluginMetadataError(pluginName, endpointErr); normalizedErr != nil && os.IsNotExist(cachedErr) {
		return nil, normalizedErr
	}

	return nil, fmt.Errorf("could not resolve plugin %s via plugin metadata endpoint: %v; cached lookup failed: %w", pluginName, endpointErr, cachedErr)
}

func resolveCachedPluginForInstall(config config.IConfig, fs afero.Fs, pluginName, version string) (*Plugin, error) {
	return readCachedPluginMetadata(config, fs, pluginName, "plugins.ResolvePluginForInstall", func(localPlugin, manifestPlugin *Plugin) *Plugin {
		return selectPluginForInstall(localPlugin, manifestPlugin, version)
	})
}

// resolveCachedPluginForUpgrade resolves plugin metadata for upgrade using
// persisted local metadata and the cached manifest.
func resolveCachedPluginForUpgrade(config config.IConfig, fs afero.Fs, pluginName string) (*Plugin, error) {
	return readCachedPluginMetadata(config, fs, pluginName, "plugins.ResolvePluginForUpgrade", selectPluginForUpgrade)
}

type cachedPluginSelector func(localPlugin, manifestPlugin *Plugin) *Plugin

func readCachedPluginMetadata(config config.IConfig, fs afero.Fs, pluginName, logPrefix string, selector cachedPluginSelector) (*Plugin, error) {
	var localPlugin *Plugin
	localPluginValue, err := readLocalPluginMetadata(config, fs, pluginName)
	if err == nil {
		localPlugin = &localPluginValue
	} else if !os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"prefix": logPrefix,
			"plugin": pluginName,
		}).Debugf("could not read local plugin metadata: %s", err)
	}

	var manifestPlugin *Plugin
	manifestPluginValue, manifestErr := lookUpPluginInCachedManifest(config, fs, pluginName)
	if manifestErr == nil {
		manifestPlugin = &manifestPluginValue
	} else {
		var pluginNotFound *ErrPluginNotFound
		if !os.IsNotExist(manifestErr) && !errors.As(manifestErr, &pluginNotFound) {
			log.WithFields(log.Fields{
				"prefix": logPrefix,
				"plugin": pluginName,
			}).Debugf("could not read cached manifest plugin metadata: %s", manifestErr)
		}
	}

	plugin := selector(localPlugin, manifestPlugin)
	if plugin != nil {
		return plugin, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	return nil, manifestErr
}

func selectPluginForInstall(localPlugin, manifestPlugin *Plugin, requestedVersion string) *Plugin {
	if requestedVersion == "" {
		return selectPluginForUpgrade(localPlugin, manifestPlugin)
	}

	localHasVersion := localPlugin != nil && localPlugin.getReleaseForVersion(requestedVersion) != nil
	manifestHasVersion := manifestPlugin != nil && manifestPlugin.getReleaseForVersion(requestedVersion) != nil

	switch {
	case localHasVersion && !manifestHasVersion:
		return mergePluginMetadata(localPlugin, manifestPlugin)
	case manifestHasVersion && !localHasVersion:
		return mergePluginMetadata(manifestPlugin, localPlugin)
	default:
		return selectPluginForUpgrade(localPlugin, manifestPlugin)
	}
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

	return &pluginCopy
}

// resolvePluginForAutoInstall resolves the version to re-download for a plugin
// that is already installed but whose binary is missing. This repairs a broken
// install rather than adding a plugin the user never asked for, so it is
// deliberately not gated on the metadata endpoint's auto_install answer.
func resolvePluginForAutoInstall(ctx context.Context, config config.IConfig, fs afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
	resolvedPlugin, err := ResolvePluginForInstall(ctx, config, fs, pluginName, "", apiBaseURL, dashboardBaseURL)
	if err == nil {
		return resolvedPlugin, nil
	}

	// Returning now also keeps the version the user needs out of this path's combined
	// "latest lookup failed; cached lookup failed" wrapper, which buries it.
	if requiresNewerCLI(err) {
		return nil, err
	}

	log.WithFields(log.Fields{
		"prefix": "plugins.resolvePluginForAutoInstall",
		"plugin": pluginName,
	}).Debugf("could not resolve latest plugin metadata for auto-install, falling back to cached metadata: %s", err)

	cachedPlugin, cachedErr := resolveCachedPluginForUpgrade(config, fs, pluginName)
	if cachedErr != nil {
		return nil, fmt.Errorf("could not resolve plugin %s for auto-install: latest lookup failed: %v; cached lookup failed: %w", pluginName, err, cachedErr)
	}

	version, versionErr := resolveVersionFromReleases(cachedPlugin, pluginName, "", cachedPluginMetadata)
	if versionErr != nil {
		return nil, versionErr
	}

	return &ResolvedPluginVersion{
		Plugin:  cachedPlugin,
		Version: version,
	}, nil
}

func findPlugin(pluginList PluginList, pluginName string) (Plugin, error) {
	for _, p := range pluginList.Plugins {
		if pluginName == p.Shortname {
			return p, nil
		}
	}

	return Plugin{}, errorcategory.Errorf(errorcategory.UserInput, "could not find a plugin named %s", pluginName)
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

func resolvePluginFromMetadata(ctx context.Context, config config.IConfig, fs afero.Fs, pluginName, version, apiBaseURL, dashboardBaseURL, apiKey string) (*ResolvedPluginVersion, error) {
	if err := ValidatePluginShortname(pluginName); err != nil {
		return nil, err
	}

	basePlugin := &Plugin{Shortname: pluginName}
	if cachedPlugin, err := readLocalPluginMetadata(config, fs, pluginName); err == nil {
		basePlugin = &cachedPlugin
	} else if cachedPlugin, err := lookUpPluginInCachedManifest(config, fs, pluginName); err == nil {
		basePlugin = &cachedPlugin
	}

	pluginMetadata, err := requests.GetPluginMetadata(ctx, apiBaseURL, dashboardBaseURL, stripe.APIVersion, apiKey, config.GetProfile(), pluginName, version, runtime.GOOS, runtime.GOARCH, config.GetMachineUUID())
	if err != nil {
		// Translated here rather than left to normalizePluginMetadataError, which only
		// runs once the cached lookup has failed too. The endpoint is the only thing that
		// knows this constraint, so its answer has to be captured where it arrives or it
		// becomes an ordinary lookup failure and the cache gets to answer instead.
		if minCoreVersion, requiresNewerCLI := requests.PluginRequiresNewerCLI(err); requiresNewerCLI {
			return nil, newErrPluginRequiresNewerCLI(pluginName, version, minCoreVersion)
		}

		return nil, err
	}

	plugin, err := basePlugin.pluginFromMetadata(pluginMetadata.PluginManifest)
	if err != nil {
		return nil, err
	}

	resolvedVersion, err := resolveVersionFromReleases(plugin, pluginName, version, livePluginMetadata)
	if err != nil {
		return nil, err
	}

	return &ResolvedPluginVersion{
		Plugin:      plugin,
		Version:     resolvedVersion,
		BinaryURL:   pluginMetadata.BinaryURL,
		AutoInstall: pluginMetadata.AutoInstall,
	}, nil
}

func getCachedPluginList(config config.IConfig, fs afero.Fs) (PluginList, error) {
	var pluginList PluginList

	body, err := afero.ReadFile(fs, getCachedPluginManifestPath(config))
	if err != nil {
		return pluginList, err
	}

	validatedPluginList, err := validatePluginManifest(body)
	if err != nil {
		return pluginList, err
	}

	return *validatedPluginList, nil
}

// Where a plugin's releases came from. Resolution treats the two identically and
// only needs to tell them apart when reporting a version it could not find, since
// a response that omitted a release and a cache that never held one are different
// problems for the user.
const (
	livePluginMetadata   = "plugin metadata response"
	cachedPluginMetadata = "cached plugin metadata"
)

// resolveVersionFromReleases picks the version to install out of a plugin's
// releases: the one that was asked for, or the newest listed.
//
// Every resolve path shares it so they cannot drift apart, and is left describing
// where the metadata came from rather than repeating the lookup itself. None of
// them weighs a release's min_core_version; see ErrPluginRequiresNewerCLI.
func resolveVersionFromReleases(plugin *Plugin, pluginName, requestedVersion, source string) (string, error) {
	if plugin == nil {
		return "", errorcategory.Errorf(errorcategory.API, "%s did not include plugin %s", source, pluginName)
	}

	// Only a version the caller named has to be looked for. LookUpLatestVersion
	// reports a version it read off a release for this platform, so asking the same
	// releases to produce that release again can only ever succeed.
	if requestedVersion == "" {
		latestVersion := plugin.LookUpLatestVersion()
		if latestVersion == "" {
			return "", errorcategory.Errorf(errorcategory.API, "%s did not include a release for %s on %s/%s", source, pluginName, runtime.GOOS, runtime.GOARCH)
		}

		return latestVersion, nil
	}

	if plugin.getReleaseForVersion(requestedVersion) == nil {
		return "", errorcategory.Errorf(errorcategory.API, "%s did not include plugin %s version %s for %s/%s", source, pluginName, requestedVersion, runtime.GOOS, runtime.GOARCH)
	}

	return requestedVersion, nil
}

func lookUpPluginInCachedManifest(config config.IConfig, fs afero.Fs, pluginName string) (Plugin, error) {
	if err := ValidatePluginShortname(pluginName); err != nil {
		return Plugin{}, err
	}

	pluginList, err := getCachedPluginList(config, fs)
	if err != nil {
		return Plugin{}, err
	}

	plugin, err := findPlugin(pluginList, pluginName)
	if err != nil {
		return Plugin{}, &ErrPluginNotFound{Name: pluginName}
	}

	return plugin, nil
}

func readLocalPluginMetadata(config config.IConfig, fs afero.Fs, pluginName string) (Plugin, error) {
	metadataPath, err := getLocalPluginMetadataPath(config, pluginName)
	if err != nil {
		return Plugin{}, err
	}

	body, err := afero.ReadFile(fs, metadataPath)
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
	metadataPath, err := getLocalPluginMetadataPath(config, plugin.Shortname)
	if err != nil {
		return err
	}

	pluginMetadataDir := getLocalPluginMetadataDir(config)
	if err := fs.MkdirAll(pluginMetadataDir, 0755); err != nil {
		return err
	}

	body := new(bytes.Buffer)
	if err := toml.NewEncoder(body).Encode(PluginList{Plugins: []Plugin{plugin}}); err != nil {
		return err
	}

	return afero.WriteFile(fs, metadataPath, body.Bytes(), 0644)
}

func removeLocalPluginMetadata(config config.IConfig, fs afero.Fs, pluginName string) error {
	metadataPath, err := getLocalPluginMetadataPath(config, pluginName)
	if err != nil {
		return err
	}

	err = fs.Remove(metadataPath)
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

func validatePluginListResponse(pluginList *PluginList) error {
	if pluginList == nil {
		return errorcategory.New(errorcategory.API, "received an empty plugin list response")
	}
	if pluginList.Plugins == nil {
		pluginList.Plugins = []Plugin{}
	}
	for i := range pluginList.Plugins {
		sortPluginReleases(pluginList.Plugins[i].Releases)
	}
	return nil
}

func sortPluginReleases(releases []Release) {
	sort.Slice(releases, func(i, j int) bool {
		vi, errI := version.NewVersion(releases[i].Version)
		vj, errJ := version.NewVersion(releases[j].Version)

		// If either version fails to parse, fall back to string comparison.
		if errI != nil || errJ != nil {
			return releases[i].Version < releases[j].Version
		}

		return vi.LessThan(vj)
	})
}

func validatePluginManifest(body []byte) (*PluginList, error) {
	var manifestBody PluginList

	if err := toml.Unmarshal(body, &manifestBody); err != nil {
		return nil, errorcategory.Errorf(errorcategory.API, "received an invalid plugin manifest: %s", err)
	}
	if len(manifestBody.Plugins) == 0 {
		return nil, errorcategory.Errorf(errorcategory.API, "received an empty plugin manifest")
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
		sortPluginReleases(pluginList.Plugins[idx].Releases)
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

func normalizePluginMetadataError(pluginName string, err error) error {
	if err == nil {
		return nil
	}

	var requestErr requests.RequestError
	if errors.As(err, &requestErr) && requestErr.StatusCode == http.StatusNotFound {
		return &ErrPluginNotFound{Name: pluginName}
	}

	switch {
	case strings.Contains(err.Error(), fmt.Sprintf("plugin metadata response did not include plugin %s version", pluginName)):
		return &ErrPluginNotFound{Name: pluginName}
	case strings.Contains(err.Error(), fmt.Sprintf("plugin metadata response did not include plugin %s", pluginName)):
		return &ErrPluginNotFound{Name: pluginName}
	default:
		return nil
	}
}

// ErrPluginNotFound is returned when a plugin cannot be found via the
// metadata endpoint or in cached local plugin metadata.
type ErrPluginNotFound struct {
	Name string
}

func (e *ErrPluginNotFound) Error() string {
	return fmt.Sprintf("no plugin named %q exists", e.Name)
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
			return nil, errorcategory.Errorf(errorcategory.Network, "failed to find the plugin repository. Make sure you are on the latest version of the Stripe CLI: https://docs.stripe.com/stripe-cli/upgrade")
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

// CheckLatestPluginVersion prints an upgrade hint to stderr if live metadata
// has a newer version of the plugin than what is currently installed.
func CheckLatestPluginVersion(ctx context.Context, config config.IConfig, fs afero.Fs, plugin Plugin, apiBaseURL, dashboardBaseURL string) {
	if PluginsPath != "" {
		return
	}

	installedVersion := plugin.InstalledVersion(config, fs)
	if installedVersion == "" {
		return
	}

	if dashboardBaseURL == "" {
		dashboardBaseURL = stripe.DashboardBaseURLForAPIBaseURL(apiBaseURL)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, checkLatestPluginVersionTimeout)
	defer cancel()

	resolvedPlugin, err := checkLatestPluginVersionResolver(ctx, config, fs, plugin.Shortname, apiBaseURL, dashboardBaseURL)
	if err != nil {
		return
	}

	latestVersion := resolvedPlugin.Version
	if latestVersion == "" && resolvedPlugin.Plugin != nil {
		latestVersion = resolvedPlugin.Plugin.LookUpLatestVersion()
	}
	if latestVersion == "" {
		return
	}

	if comparePluginVersions(installedVersion, latestVersion) < 0 {
		c := ansi.Color(os.Stderr)
		msg := fmt.Sprintf(
			"A newer version of the %s plugin is available (v%s → v%s). Run `stripe plugin upgrade %s` to update.",
			plugin.Shortname, installedVersion, latestVersion, plugin.Shortname,
		)
		fmt.Fprintln(os.Stderr, c.Yellow(msg).String())
	}
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
