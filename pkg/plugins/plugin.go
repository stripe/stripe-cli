// Package plugins provides the plugin system for extending the CLI.
package plugins

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/fsutil"
	"github.com/stripe/stripe-cli/pkg/plugins/proto"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"

	hclog "github.com/hashicorp/go-hclog"
	hcplugin "github.com/hashicorp/go-plugin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// dev mode vars
var (
	PluginDev   = false
	PluginsPath string
)

const localDevelopmentVersion = "local.build.dev"

// CommandInfo describes a plugin subcommand for tree display (e.g. in --map).
type CommandInfo struct {
	Name     string        `toml:"Name" json:"name"`
	Desc     string        `toml:"Desc" json:"desc,omitempty"`
	Commands []CommandInfo `toml:"Command,omitempty" json:"commands,omitempty"`
}

// Plugin contains the plugin properties
type Plugin struct {
	Shortname        string        `toml:"Shortname"`
	Shortdesc        string        `toml:"Shortdesc"`
	Binary           string        `toml:"Binary"`
	Releases         []Release     `toml:"Release"`
	MagicCookieValue string        `toml:"MagicCookieValue"`
	Commands         []CommandInfo `toml:"Command,omitempty"`
}

// PluginList contains a list of plugins
type PluginList struct {
	Plugins []Plugin `toml:"Plugin"`
}

// Release is the type that holds release data for a specific build of a plugin
type Release struct {
	Arch    string            `toml:"Arch"`
	OS      string            `toml:"OS"`
	Version string            `toml:"Version"`
	Sum     string            `toml:"Sum"`
	Runtime map[string]string `toml:"Runtime,omitempty"`
}

// getPluginInterface computes the correct metadata needed for starting the hcplugin client
func (p *Plugin) getPluginInterface() (hcplugin.HandshakeConfig, map[int]hcplugin.PluginSet) {
	handshakeConfig := hcplugin.HandshakeConfig{
		MagicCookieKey:   fmt.Sprintf("plugin_%s", p.Shortname),
		MagicCookieValue: p.MagicCookieValue,
	}

	// pluginMap is the map of interfaces we can dispense from the plugin itself
	// we just have one called "main" for each of our plugins for now
	pluginSetMap := map[int]hcplugin.PluginSet{
		1: {
			"main": &CLIPluginV1{},
		},
		2: {
			"main": &CLIPluginGRPC{},
		},
		3: {
			"main": &CLIPluginV3{},
		},
	}

	return handshakeConfig, pluginSetMap
}

// getPluginInstallPath computes the absolute path of a specific plugin version's installation dir
func (p *Plugin) getPluginInstallPath(config config.IConfig, version string) string {
	pluginsDir := getPluginsDir(config)
	pluginPath := filepath.Join(pluginsDir, p.Shortname, version)
	cleanedPath := filepath.Clean(pluginPath)

	return cleanedPath
}

func isLocalDevelopmentVersion(version string) bool {
	return version == localDevelopmentVersion
}

func (p *Plugin) lookUpInstalledVersion(config config.IConfig, fs afero.Fs) (string, error) {
	localDevPath := p.getPluginInstallPath(config, localDevelopmentVersion)
	localDevExists, err := afero.DirExists(fs, localDevPath)
	if err != nil {
		return "", err
	}
	if localDevExists {
		return localDevelopmentVersion, nil
	}

	localPluginDir := filepath.Join(getPluginsDir(config), p.Shortname, "*.*.*")
	existingLocalPlugin, err := afero.Glob(fs, localPluginDir)
	if err != nil {
		return "", err
	}
	if len(existingLocalPlugin) == 0 {
		return "", nil
	}

	return filepath.Base(existingLocalPlugin[0]), nil
}

// cleanUpPluginPath empties the plugin folder except for the version specified
func (p *Plugin) cleanUpPluginPath(config config.IConfig, fs afero.Fs, versionToKeep string) error {
	logger := log.WithFields(log.Fields{
		"prefix": "plugins.plugin.cleanUpPluginPath",
	})
	logger.Debug("Cleaning up other plugin versions...")

	pluginsDir := getPluginsDir(config)
	pluginPath := filepath.Join(pluginsDir, p.Shortname)
	versionPathToKeep := filepath.Join(pluginPath, versionToKeep)

	afero.Walk(fs, pluginPath, filepath.WalkFunc(func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		switch {
		case path == pluginPath:
			// Pass the root directory
			logger.Debugf("Skipping directory: %s", path)
			return nil
		case info.IsDir() && path == versionPathToKeep:
			logger.Debugf("Skipping directory: %s", path)
			return filepath.SkipDir
		default:
			logger.Debugf("Removing old plugin: %s", path)
			fs.RemoveAll(path)
			return nil
		}
	}))

	return nil
}

// getChecksum does what it says on the tin - it returns the checksum for a specific plugin version
func (p *Plugin) getChecksum(version string) ([]byte, error) {
	opsystem := runtime.GOOS
	arch := runtime.GOARCH

	var expectedSum string
	for _, pkg := range p.Releases {
		if pkg.OS == opsystem && pkg.Arch == arch && pkg.Version == version {
			expectedSum = pkg.Sum
		}
	}

	if expectedSum == "" {
		return nil, fmt.Errorf("could not locate a valid checksum for %s version %s", p.Shortname, version)
	}

	decoded, err := hex.DecodeString(expectedSum)
	if err != nil {
		return nil, fmt.Errorf("could not decode checksum for %s version %s", p.Shortname, version)
	}

	return decoded, nil
}

// LookUpLatestVersion gets latest CLI version
// note: assumes versions are listed in asc order
func (p *Plugin) LookUpLatestVersion() string {
	opsystem := runtime.GOOS
	arch := runtime.GOARCH

	var version string
	for _, pkg := range p.Releases {
		if pkg.OS == opsystem && pkg.Arch == arch {
			version = pkg.Version
		}
	}

	return version
}

// getReleaseForVersion finds the release object for a specific version on the current platform
func (p *Plugin) getReleaseForVersion(version string) *Release {
	return p.getRelease(version, runtime.GOOS, runtime.GOARCH)
}

func (p *Plugin) getRelease(version, opsystem, arch string) *Release {
	for _, release := range p.Releases {
		if release.Version == version && release.OS == opsystem && release.Arch == arch {
			releaseCopy := release
			return &releaseCopy
		}
	}

	return nil
}

func copyRuntime(runtimeRequirements map[string]string) map[string]string {
	if len(runtimeRequirements) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(runtimeRequirements))
	for name, version := range runtimeRequirements {
		cloned[name] = version
	}

	return cloned
}

func (p *Plugin) pluginFromMetadata(pluginManifest string) (*Plugin, error) {
	pluginList, err := validatePluginManifest([]byte(pluginManifest))
	if err != nil {
		return nil, err
	}

	for _, candidate := range pluginList.Plugins {
		if candidate.Shortname != p.Shortname {
			continue
		}

		if len(candidate.Commands) == 0 && len(p.Commands) > 0 {
			candidate.Commands = p.Commands
		}

		for i := range candidate.Releases {
			if len(candidate.Releases[i].Runtime) != 0 {
				continue
			}

			existingRelease := p.getRelease(candidate.Releases[i].Version, candidate.Releases[i].OS, candidate.Releases[i].Arch)
			if existingRelease == nil || len(existingRelease.Runtime) == 0 {
				continue
			}

			candidate.Releases[i].Runtime = copyRuntime(existingRelease.Runtime)
		}

		return &candidate, nil
	}

	return nil, fmt.Errorf("plugin metadata response did not include plugin %s", p.Shortname)
}

// IsVersionInstalled returns true if the given version of the plugin is already installed on disk.
func (p *Plugin) IsVersionInstalled(config config.IConfig, fs afero.Fs, version string) bool {
	pluginDir := p.getPluginInstallPath(config, version)
	pluginBinaryPath := filepath.Join(pluginDir, p.Binary) + GetBinaryExtension()
	_, err := fs.Stat(pluginBinaryPath)
	return err == nil
}

// InstalledVersion returns the currently installed version of the plugin, or empty string if none.
func (p *Plugin) InstalledVersion(config config.IConfig, fs afero.Fs) string {
	pluginsDir := getPluginsDir(config)
	pluginDir := filepath.Join(pluginsDir, p.Shortname)

	entries, err := afero.ReadDir(fs, pluginDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return entry.Name()
		}
	}

	return ""
}

// Install installs the plugin of the given version
func (p *Plugin) Install(ctx context.Context, cfg config.IConfig, fs afero.Fs, version string, baseURL string) error {
	spinner := ansi.StartNewSpinner(ansi.Faint(fmt.Sprintf("installing '%s' v%s...", p.Shortname, version)), os.Stdout)

	apiKey, _ := cfg.GetProfile().GetAPIKey(false)
	pluginToInstall := p
	var pluginDownloadURL string
	downloadURLFromMetadata := false

	if apiKey != "" {
		log.WithFields(log.Fields{
			"prefix":   "plugins.plugin.Install",
			"endpoint": "/v1/stripecli/get-plugin-metadata",
			"plugin":   p.Shortname,
			"version":  version,
			"os":       runtime.GOOS,
			"arch":     runtime.GOARCH,
		}).Debug("Fetching plugin metadata for install")

		pluginMetadata, err := requests.GetPluginMetadata(ctx, baseURL, stripe.APIVersion, apiKey, cfg.GetProfile(), p.Shortname, version, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			log.WithFields(log.Fields{
				"prefix": "plugins.plugin.Install",
			}).Debugf("could not fetch plugin metadata, falling back to plugin URL lookup: %s", err)
		} else {
			pluginFromMetadata, err := p.pluginFromMetadata(pluginMetadata.PluginManifest)
			if err != nil {
				ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s'", p.Shortname)), os.Stdout)
				return err
			}

			pluginToInstall = pluginFromMetadata
			pluginDownloadURL = pluginMetadata.BinaryURL
			downloadURLFromMetadata = true
		}
	}

	if pluginDownloadURL == "" {
		var err error
		pluginDownloadURL, err = getLegacyPluginDownloadURL(ctx, cfg, apiKey, baseURL, version, pluginToInstall)
		if err != nil {
			ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s'", p.Shortname)), os.Stdout)

			log.WithFields(log.Fields{
				"prefix": "plugins.plugin.Install",
			}).Debugf("install error: %s", err)

			return errors.New("you don't seem to have access to this plugin")
		}
	}

	// Check if this plugin requires a runtime and install it if needed
	release := pluginToInstall.getReleaseForVersion(version)
	if release != nil {
		if nodeVersion, requiresNode := GetRuntimeRequirement(*release); requiresNode {
			ansi.StopSpinner(spinner, "", os.Stdout)
			if err := InstallNodeRuntime(ctx, cfg, fs, nodeVersion); err != nil {
				return fmt.Errorf("failed to install required Node.js runtime: %w", err)
			}
			spinner = ansi.StartNewSpinner(ansi.Faint(fmt.Sprintf("installing '%s' v%s...", p.Shortname, version)), os.Stdout)
		}
	}

	// Pull down bin, verify, and save to disk
	err := pluginToInstall.downloadAndSavePlugin(cfg, pluginDownloadURL, fs, version)
	if err != nil && downloadURLFromMetadata {
		log.WithFields(log.Fields{
			"prefix": "plugins.plugin.Install",
		}).Debugf("could not download plugin from metadata URL, falling back to plugin URL lookup: %s", err)

		fallbackURL, fallbackErr := getLegacyPluginDownloadURL(ctx, cfg, apiKey, baseURL, version, pluginToInstall)
		if fallbackErr != nil {
			log.WithFields(log.Fields{
				"prefix": "plugins.plugin.Install",
			}).Debugf("could not look up fallback plugin URL after metadata download failed: %s", fallbackErr)
		} else {
			err = pluginToInstall.downloadAndSavePlugin(cfg, fallbackURL, fs, version)
		}
	}

	if err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s': %s", p.Shortname, err)), os.Stdout)
		return err
	}

	if err := PersistInstalledPluginState(cfg, fs, *pluginToInstall); err != nil {
		pluginPath := pluginToInstall.getPluginInstallPath(cfg, version)
		if cleanupErr := fs.RemoveAll(pluginPath); cleanupErr != nil {
			log.WithFields(log.Fields{
				"prefix": "plugins.plugin.Install",
				"path":   pluginPath,
			}).Debugf("could not clean up plugin after local metadata write failure: %s", cleanupErr)
		}

		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s': %s", p.Shortname, err)), os.Stdout)
		return err
	}

	// Once the plugin is successfully downloaded, clean up other versions
	p.cleanUpPluginPath(cfg, fs, version)

	ansi.StopSpinner(spinner, "", os.Stdout)

	return nil
}

func getLegacyPluginDownloadURL(ctx context.Context, cfg config.IConfig, apiKey, baseURL, version string, plugin *Plugin) (string, error) {
	pluginData, err := requests.GetPluginData(ctx, baseURL, stripe.APIVersion, apiKey, cfg.GetProfile())
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s/%s/%s/%s", pluginData.PluginBaseURL, plugin.Shortname, version, runtime.GOOS, runtime.GOARCH, plugin.Binary), nil
}

// Uninstall removes a plugin from the disk and from the config's installed plugins list
func (p *Plugin) Uninstall(ctx context.Context, config config.IConfig, fs afero.Fs) error {
	pluginList := config.GetInstalledPlugins()
	pluginIdx := -1

	for i, name := range pluginList {
		if name == p.Shortname {
			pluginIdx = i
		}
	}

	pluginDir := p.getPluginInstallPath(config, "")
	dirExists, err := afero.DirExists(fs, pluginDir)
	if err != nil {
		return err
	}
	metadataPath := getLocalPluginMetadataPath(config, p.Shortname)
	metadataExists, err := afero.Exists(fs, metadataPath)
	if err != nil {
		return err
	}

	if pluginIdx == -1 && !dirExists && !metadataExists {
		return errors.New("this plugin doesn't seem to be installed, canceling")
	}

	previousState, err := snapshotInstalledPluginState(config, fs, p.Shortname)
	if err != nil {
		return err
	}

	if err := removeLocalPluginMetadata(config, fs, p.Shortname); err != nil {
		return err
	}
	if err := RemoveInstalledPlugin(config, p.Shortname); err != nil {
		if rollbackErr := rollbackInstalledPluginState(config, fs, p.Shortname, previousState); rollbackErr != nil {
			return fmt.Errorf("failed to update uninstall state for plugin %s: %w; rollback failed: %v", p.Shortname, err, rollbackErr)
		}
		return err
	}

	err = fs.RemoveAll(pluginDir)
	if err != nil {
		if rollbackErr := rollbackInstalledPluginState(config, fs, p.Shortname, previousState); rollbackErr != nil {
			return fmt.Errorf("failed to remove plugin files for %s: %w; rollback failed: %v", p.Shortname, err, rollbackErr)
		}
		return err
	}

	return nil
}

func (p *Plugin) downloadAndSavePlugin(config config.IConfig, pluginDownloadURL string, fs afero.Fs, version string) error {
	body, err := FetchRemoteResource(pluginDownloadURL)
	if err != nil {
		return err
	}

	err = p.verifychecksumAndSavePlugin(body, config, fs, version)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) verifychecksumAndSavePlugin(pluginData []byte, config config.IConfig, fs afero.Fs, version string) error {
	logger := log.WithFields(log.Fields{
		"prefix": "plugins.plugin.Install",
	})

	pluginDir := p.getPluginInstallPath(config, version)
	pluginFilePath := filepath.Join(pluginDir, p.Binary)
	pluginFilePath += GetBinaryExtension()

	logger.Debugf("installing %s to %s...", p.Shortname, pluginFilePath)

	reader := bytes.NewReader(pluginData)

	err := p.verifyChecksum(reader, version)
	if err != nil {
		logger.Debug("could not match checksum of plugin")
		return err
	}

	if err := fsutil.RefuseWriteThroughSymlink(fs, pluginFilePath, filepath.Dir(getPluginsDir(config)), filepath.Base(pluginFilePath)); err != nil {
		return err
	}

	err = fs.MkdirAll(pluginDir, 0755)
	if err != nil {
		logger.Debugf("could not create plugin directory: %s", pluginDir)
		return err
	}

	err = afero.WriteFile(fs, pluginFilePath, pluginData, 0755)
	if err != nil {
		logger.Debug("could not save plugin to disk")
		return err
	}

	return nil
}

// verifyChecksum is to be used during installation only
// hcplugins takes care of the boot time verification for us
func (p *Plugin) verifyChecksum(binary io.Reader, version string) error {
	if isLocalDevelopmentVersion(version) {
		return nil
	}

	expectedSum, err := p.getChecksum(version)
	if err != nil {
		return err
	}

	hash := sha256.New()
	_, err = io.Copy(hash, binary)
	if err != nil {
		return err
	}

	actualSum := hash.Sum(nil)
	if !bytes.Equal(actualSum, expectedSum) {
		return fmt.Errorf("installed plugin '%s' could not be verified, aborting installation", p.Shortname)
	}

	return nil
}

func buildAdditionalInfo(logger *log.Entry) *proto.AdditionalInfo {
	var terminalDimensions *proto.TerminalDimensions
	if term.IsTerminal(int(os.Stdout.Fd())) {
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			terminalDimensions = &proto.TerminalDimensions{
				Width:  uint32(width),
				Height: uint32(height),
			}
		} else {
			// Fail silently, this shouldn't block the plugin from running
			logger.Debugf("could not get terminal dimensions: %s", err)
			terminalDimensions = &proto.TerminalDimensions{
				Width:  0,
				Height: 0,
			}
		}
	}
	return &proto.AdditionalInfo{
		IsTerminal: &proto.IsTerminal{
			Stdin:  term.IsTerminal(int(os.Stdin.Fd())),
			Stdout: term.IsTerminal(int(os.Stdout.Fd())),
			Stderr: term.IsTerminal(int(os.Stderr.Fd())),
		},
		TerminalDimensions: terminalDimensions,
	}
}

// Run boots up the binary and then sends the command to it via RPC.
// cwd sets the working directory for the plugin process; an empty string uses the current directory.
func (p *Plugin) Run(ctx context.Context, config *config.Config, fs afero.Fs, args []string, cwd string) error {
	logger := log.WithFields(log.Fields{
		"prefix": "plugins.plugin.Run",
	})

	var version string

	if PluginsPath != "" {
		version = localDevelopmentVersion
	} else {
		var err error
		version, err = p.lookUpInstalledVersion(config, fs)
		if err != nil {
			return err
		}

		// If the plugin binary is missing locally, resolve the freshest metadata
		// before reinstalling so stale cached local metadata does not pin us to an
		// older release.
		if version == "" {
			pluginToInstall, resolvedVersion, err := resolvePluginForAutoInstall(ctx, config, fs, p.Shortname, stripe.DefaultAPIBaseURL)
			if err != nil {
				return err
			}

			p = pluginToInstall
			version = resolvedVersion
			err = p.Install(ctx, config, fs, version, stripe.DefaultAPIBaseURL)
			if err != nil {
				return err
			}
		}
	}

	pluginDir := p.getPluginInstallPath(config, version)
	pluginBinaryPath := filepath.Join(pluginDir, p.Binary)
	pluginBinaryPath += GetBinaryExtension()

	// Check if this plugin requires a runtime
	var cmd *exec.Cmd
	var usesRuntime bool
	release := p.getReleaseForVersion(version)
	if release != nil {
		if nodeVersion, requiresNode := GetRuntimeRequirement(*release); requiresNode {
			// Plugin requires Node.js runtime - execute via node
			nodePath := GetNodeBinaryPath(config, nodeVersion)
			if nodePath == "" {
				return fmt.Errorf("required Node.js runtime v%s is not installed", nodeVersion)
			}
			logger.Debugf("Executing plugin via Node.js runtime: %s %s", nodePath, pluginBinaryPath)
			cmd = exec.Command(nodePath, pluginBinaryPath)
			usesRuntime = true
		} else {
			// No runtime required - execute binary directly
			cmd = exec.Command(pluginBinaryPath)
			usesRuntime = false
		}
	} else {
		// Couldn't find release info, assume it's a standalone binary
		cmd = exec.Command(pluginBinaryPath)
		usesRuntime = false
	}

	if cwd != "" {
		cmd.Dir = cwd
	}

	handshakeConfig, pluginSetMap := p.getPluginInterface()
	timeout, _ := time.ParseDuration("10s")

	pluginLogger := hclog.New(&hclog.LoggerOptions{
		Name:  fmt.Sprintf("plugin.child.%s", p.Shortname),
		Level: hclog.LevelFromString("ERROR"),
	})

	clientConfig := &hcplugin.ClientConfig{
		HandshakeConfig:  handshakeConfig,
		VersionedPlugins: pluginSetMap,
		Cmd:              cmd,
		SyncStdout:       os.Stdout,
		SyncStderr:       os.Stderr,
		Logger:           pluginLogger,
		Managed:          true,
		StartTimeout:     timeout,
		AllowedProtocols: []hcplugin.Protocol{
			hcplugin.ProtocolGRPC, hcplugin.ProtocolNetRPC,
		},
	}

	// Only validate checksum for standalone binaries, not when using a runtime
	// When using a runtime, cmd.Path points to the node binary, not the plugin
	if !usesRuntime && !isLocalDevelopmentVersion(version) {
		sum, err := p.getChecksum(version)
		if err != nil {
			return err
		}

		clientConfig.SecureConfig = &hcplugin.SecureConfig{
			Checksum: sum,
			Hash:     sha256.New(),
		}
	}

	// start by launching the plugin process / binary
	client := hcplugin.NewClient(clientConfig)

	// Connect via RPC to the plugin
	rpcClient, err := client.Client()
	if err != nil {
		logger.Debugf("Could not connect to plugin: %s", err)
		return err
	}

	// Request the plugin's main interface
	raw, err := rpcClient.Dispense("main")
	if err != nil {
		logger.Debugf("Could not dispense plugin interface: %s", err)
		return err
	}

	// get the native golang interface for the plugin so that we can call it directly
	switch d := raw.(type) {
	case Dispatcher:
		logger.Debug("negotiated net/rpc with plugin process")
		if _, err = d.RunCommand(args); err != nil {
			return err
		}
	case DispatcherGRPC:
		logger.Debug("negotiated gRPC with plugin process")
		if err = d.RunCommand(buildAdditionalInfo(logger), args); err != nil {
			return err
		}
	case DispatcherV3:
		logger.Debug("negotiated gRPC with plugin process (v3)")
		if err = d.RunCommand(buildAdditionalInfo(logger), args, NewCoreCLIHelper(ctx, config, fs)); err != nil {
			return err
		}
	default:
		return errors.New("dispensed an unknown plugin interface")
	}
	return nil
}
