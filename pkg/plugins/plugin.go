// Package plugins provides the plugin system for extending the CLI.
package plugins

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/stripe/stripe-cli/pkg/errorcategory"
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
	Shortname        string        `toml:"Shortname" json:"shortname"`
	Shortdesc        string        `toml:"Shortdesc" json:"shortdesc"`
	Description      string        `toml:"Description,omitempty" json:"description,omitempty"`
	Binary           string        `toml:"Binary" json:"binary"`
	Releases         []Release     `toml:"Release" json:"releases"`
	MagicCookieValue string        `toml:"MagicCookieValue" json:"magic_cookie_value,omitempty"`
	Commands         []CommandInfo `toml:"Command,omitempty" json:"commands,omitempty"`
}

// PluginList contains a list of plugins
type PluginList struct {
	Plugins []Plugin `toml:"Plugin" json:"plugins"`
}

// Release is the type that holds release data for a specific build of a plugin
//
// The manifest also carries each release's MinCoreVersion, which is deliberately
// not decoded; see ErrPluginRequiresNewerCLI for why the API owns that constraint.
type Release struct {
	Arch    string `toml:"Arch" json:"arch"`
	OS      string `toml:"OS" json:"os"`
	Version string `toml:"Version" json:"version"`
	Sum     string `toml:"Sum" json:"sum,omitempty"`
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

// getPluginInstallPath computes the absolute path of a specific plugin version's installation dir.
func (p *Plugin) getPluginInstallPath(config config.IConfig, version string) (string, error) {
	if err := ValidatePluginShortname(p.Shortname); err != nil {
		return "", err
	}

	pluginsDir := getPluginsDir(config)
	pluginPath := filepath.Join(pluginsDir, p.Shortname, version)
	cleanedPath := filepath.Clean(pluginPath)

	return cleanedPath, nil
}

func isLocalDevelopmentVersion(version string) bool {
	return version == localDevelopmentVersion
}

func (p *Plugin) lookUpInstalledVersion(config config.IConfig, fs afero.Fs) (string, error) {
	localDevPath, err := p.getPluginInstallPath(config, localDevelopmentVersion)
	if err != nil {
		return "", err
	}
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

	pluginPath, err := p.getPluginInstallPath(config, "")
	if err != nil {
		return err
	}
	versionPathToKeep, err := p.getPluginInstallPath(config, versionToKeep)
	if err != nil {
		return err
	}

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
		return nil, errorcategory.Errorf(errorcategory.API, "could not locate a valid checksum for %s version %s", p.Shortname, version)
	}

	decoded, err := hex.DecodeString(expectedSum)
	if err != nil {
		return nil, errorcategory.Errorf(errorcategory.API, "could not decode checksum for %s version %s", p.Shortname, version)
	}

	return decoded, nil
}

// LookUpLatestVersion gets the latest version of the plugin for this platform.
// It weighs no min_core_version of its own; see ErrPluginRequiresNewerCLI.
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

		return &candidate, nil
	}

	return nil, errorcategory.Errorf(errorcategory.API, "plugin metadata response did not include plugin %s", p.Shortname)
}

// IsVersionInstalled returns true if the given version of the plugin is already installed on disk.
func (p *Plugin) IsVersionInstalled(config config.IConfig, fs afero.Fs, version string) bool {
	pluginDir, err := p.getPluginInstallPath(config, version)
	if err != nil {
		return false
	}
	pluginBinaryPath := filepath.Join(pluginDir, p.Binary) + GetBinaryExtension()
	_, err = fs.Stat(pluginBinaryPath)
	return err == nil
}

// InstalledVersion returns the currently installed version of the plugin, or empty string if none.
func (p *Plugin) InstalledVersion(config config.IConfig, fs afero.Fs) string {
	pluginDir, err := p.getPluginInstallPath(config, "")
	if err != nil {
		return ""
	}

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

// Install installs the plugin of the given version.
func (p *Plugin) Install(ctx context.Context, cfg config.IConfig, fs afero.Fs, version string, apiBaseURL, dashboardBaseURL string) error {
	return p.install(ctx, cfg, fs, version, apiBaseURL, dashboardBaseURL, "", false)
}

func (p *Plugin) install(ctx context.Context, cfg config.IConfig, fs afero.Fs, version string, apiBaseURL, dashboardBaseURL, resolvedBinaryURL string, skipMetadataLookup bool) error {
	spinner := ansi.StartNewSpinner(ansi.Faint(fmt.Sprintf("installing '%s' v%s...", p.Shortname, version)), os.Stderr)

	creds, _ := cfg.GetProfile().ResolveCredentialsForAnyMode(false)
	apiKey := creds.Token
	pluginToInstall := p
	pluginDownloadURL := resolvedBinaryURL
	var metadataLookupErr error
	metadataBaseURL := apiBaseURL
	if apiKey == "" && dashboardBaseURL != "" {
		metadataBaseURL = dashboardBaseURL
	}

	if !skipMetadataLookup {
		metadataEndpoint := "/v1/stripecli/get-plugin-metadata"
		if apiKey == "" {
			metadataEndpoint = "/ajax/stripecli/plugins_metadata"
		}

		log.WithFields(log.Fields{
			"prefix":   "plugins.plugin.Install",
			"base_url": metadataBaseURL,
			"endpoint": metadataEndpoint,
			"plugin":   p.Shortname,
			"version":  version,
			"os":       runtime.GOOS,
			"arch":     runtime.GOARCH,
		}).Debug("Fetching plugin metadata for install")

		pluginMetadata, err := requests.GetPluginMetadata(ctx, apiBaseURL, dashboardBaseURL, stripe.APIVersion, apiKey, cfg.GetProfile(), p.Shortname, version, runtime.GOOS, runtime.GOARCH, cfg.GetMachineUUID())
		if err != nil {
			// Returned rather than kept as metadataLookupErr, which surfaces further down
			// as a missing download URL. The install fails either way -- this lookup only
			// runs when no binary URL was resolved earlier, and a refusal carries none --
			// so what this branch changes is that the user is told their CLI version is
			// the reason, which nothing below can state.
			if minCoreVersion, requiresNewerCLI := requests.PluginRequiresNewerCLI(err); requiresNewerCLI {
				ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s'", p.Shortname)), os.Stderr)
				return newErrPluginRequiresNewerCLI(p.Shortname, version, minCoreVersion)
			}

			metadataLookupErr = err
			log.WithFields(log.Fields{
				"prefix": "plugins.plugin.Install",
			}).Debugf("could not fetch plugin metadata: %s", err)
		} else {
			pluginFromMetadata, err := p.pluginFromMetadata(pluginMetadata.PluginManifest)
			if err != nil {
				ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s'", p.Shortname)), os.Stderr)
				return err
			}

			pluginToInstall = pluginFromMetadata
			pluginDownloadURL = pluginMetadata.BinaryURL
		}
	}

	if pluginDownloadURL == "" {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s'", p.Shortname)), os.Stderr)
		if metadataLookupErr != nil {
			return fmt.Errorf("could not resolve download URL for plugin '%s' v%s: failed to fetch plugin metadata: %w", p.Shortname, version, metadataLookupErr)
		}
		return errorcategory.Errorf(errorcategory.API, "could not resolve download URL for plugin '%s' v%s: the plugin metadata endpoint did not return a binary URL", p.Shortname, version)
	}

	// Pull down bin, verify, and save to disk
	if err := pluginToInstall.downloadAndSavePlugin(cfg, pluginDownloadURL, fs, version); err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s': %s", p.Shortname, err)), os.Stderr)
		return err
	}

	if err := PersistInstalledPluginState(cfg, fs, *pluginToInstall); err != nil {
		pluginPath, pathErr := pluginToInstall.getPluginInstallPath(cfg, version)
		if pathErr != nil {
			log.WithFields(log.Fields{
				"prefix": "plugins.plugin.Install",
			}).Debugf("could not determine plugin path for cleanup after local metadata write failure: %s", pathErr)
		} else if cleanupErr := fs.RemoveAll(pluginPath); cleanupErr != nil {
			log.WithFields(log.Fields{
				"prefix": "plugins.plugin.Install",
				"path":   pluginPath,
			}).Debugf("could not clean up plugin after local metadata write failure: %s", cleanupErr)
		}

		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s': %s", p.Shortname, err)), os.Stderr)
		return err
	}

	// Once the plugin is successfully downloaded, clean up other versions
	p.cleanUpPluginPath(cfg, fs, version)

	ansi.StopSpinner(spinner, "", os.Stderr)

	return nil
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

	pluginDir, err := p.getPluginInstallPath(config, "")
	if err != nil {
		return err
	}
	dirExists, err := afero.DirExists(fs, pluginDir)
	if err != nil {
		return err
	}
	metadataPath, err := getLocalPluginMetadataPath(config, p.Shortname)
	if err != nil {
		return err
	}
	metadataExists, err := afero.Exists(fs, metadataPath)
	if err != nil {
		return err
	}

	if pluginIdx == -1 && !dirExists && !metadataExists {
		return errorcategory.New(errorcategory.UserInput, "this plugin doesn't seem to be installed, canceling")
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

	pluginDir, err := p.getPluginInstallPath(config, version)
	if err != nil {
		return err
	}
	pluginFilePath := filepath.Join(pluginDir, p.Binary)
	pluginFilePath += GetBinaryExtension()

	logger.Debugf("installing %s to %s...", p.Shortname, pluginFilePath)

	reader := bytes.NewReader(pluginData)

	err = p.verifyChecksum(reader, version)
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
		return errorcategory.Errorf(errorcategory.API, "installed plugin '%s' could not be verified, aborting installation", p.Shortname)
	}

	return nil
}

func buildAdditionalInfo(logger *log.Entry, apiBaseURL, dashboardBaseURL, accessBaseURL string) *proto.AdditionalInfo {
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
		ApiBaseUrl:         apiBaseURL,
		DashboardBaseUrl:   dashboardBaseURL,
		AccessBaseUrl:      accessBaseURL,
	}
}

// dispensePluginInterface launches the plugin binary at the given installed version and
// returns its dispensed "main" interface, which is one of Dispatcher, DispatcherGRPC, or
// DispatcherV3 depending on the protocol version the plugin negotiates, along with the
// *hcplugin.Client managing that plugin's subprocess. cwd sets the working directory for the
// plugin process; an empty string uses the current directory.
//
// The returned client is non-nil as soon as the subprocess has been launched (i.e. once
// hcplugin.NewClient is called), even if a later step in this function errors, so callers can
// unconditionally kill it to avoid leaking the subprocess.
func (p *Plugin) dispensePluginInterface(config config.IConfig, fs afero.Fs, version, cwd string, logger *log.Entry) (interface{}, *hcplugin.Client, error) {
	pluginDir, err := p.getPluginInstallPath(config, version)
	if err != nil {
		return nil, nil, err
	}
	pluginBinaryPath := filepath.Join(pluginDir, p.Binary)
	pluginBinaryPath += GetBinaryExtension()

	cmd := exec.Command(pluginBinaryPath)

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

	if !isLocalDevelopmentVersion(version) {
		sum, err := p.getChecksum(version)
		if err != nil {
			return nil, nil, err
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
		return nil, client, err
	}

	// Request the plugin's main interface
	raw, err := rpcClient.Dispense("main")
	if err != nil {
		logger.Debugf("Could not dispense plugin interface: %s", err)
		return nil, client, err
	}

	return raw, client, nil
}

// Run boots up the binary and then sends the command to it via RPC.
// cwd sets the working directory for the plugin process; an empty string uses the current directory.
// versionOverride, when non-empty, forces the plugin to run at that specific installed version,
// bypassing the automatic version resolution (including local.build.dev priority).
// apiBaseURL, dashboardBaseURL, and accessBaseURL are forwarded to the plugin via AdditionalInfo
// so it can target the same non-default environment as the CLI that launched it. They should be
// empty unless the user explicitly passed --api-base/--dashboard-base/--access-base; an empty
// value tells the plugin to fall back to its own default rather than the CLI's resolved default.
func (p *Plugin) Run(ctx context.Context, config *config.Config, fs afero.Fs, args []string, cwd string, versionOverride string, apiBaseURL, dashboardBaseURL, accessBaseURL string) error {
	logger := log.WithFields(log.Fields{
		"prefix": "plugins.plugin.Run",
	})

	var version string

	switch {
	case versionOverride != "":
		version = versionOverride
		if !p.IsVersionInstalled(config, fs, version) {
			installed := p.InstalledVersion(config, fs)
			hint := ""
			if installed != "" {
				hint = fmt.Sprintf("; installed version is %s", installed)
			}
			return errorcategory.Errorf(errorcategory.UserInput, "plugin %q version %q is not installed%s", p.Shortname, version, hint)
		}
	case PluginsPath != "":
		version = localDevelopmentVersion
	default:
		var err error
		version, err = p.lookUpInstalledVersion(config, fs)
		if err != nil {
			return err
		}

		// If the plugin binary is missing locally, resolve the freshest metadata
		// before reinstalling so stale cached local metadata does not pin us to an
		// older release.
		if version == "" {
			installAPIBaseURL := apiBaseURL
			if installAPIBaseURL == "" {
				installAPIBaseURL = stripe.DefaultAPIBaseURL
			}
			installDashboardBaseURL := dashboardBaseURL
			if installDashboardBaseURL == "" {
				installDashboardBaseURL = stripe.DashboardBaseURLForAPIBaseURL(installAPIBaseURL)
			}

			resolvedPlugin, err := resolvePluginForAutoInstall(ctx, config, fs, p.Shortname, installAPIBaseURL, installDashboardBaseURL)
			if err != nil {
				return err
			}

			p = resolvedPlugin.Plugin
			version = resolvedPlugin.Version
			if err := resolvedPlugin.Install(ctx, config, fs, installAPIBaseURL, installDashboardBaseURL); err != nil {
				return err
			}

			runPostInstallHook(ctx, config, fs, p, version, "", apiBaseURL, dashboardBaseURL, accessBaseURL)
		}
	}

	// Plugins read the config file themselves, so one too old to understand the v2
	// layout would start up and find no profiles at all. Fail with something the
	// user can act on instead.
	if err := p.refuseIfConfigTooNew(version); err != nil {
		return err
	}

	raw, _, err := p.dispensePluginInterface(config, fs, version, cwd, logger)
	if err != nil {
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
		if err = d.RunCommand(buildAdditionalInfo(logger, apiBaseURL, dashboardBaseURL, accessBaseURL), args); err != nil {
			return err
		}
	case DispatcherV3:
		logger.Debug("negotiated gRPC with plugin process (v3)")
		if err = d.RunCommand(buildAdditionalInfo(logger, apiBaseURL, dashboardBaseURL, accessBaseURL), args, NewCoreCLIHelper(ctx, config, fs, apiBaseURL, dashboardBaseURL, accessBaseURL)); err != nil {
			return err
		}
	default:
		return errorcategory.New(errorcategory.Internal, "dispensed an unknown plugin interface")
	}
	return nil
}

// runPostInstallHook calls the plugin's PostInstall hook for a version installed outside the
// explicit `plugin install`/`upgrade` commands (e.g. Run's auto-install when the binary is
// missing locally). This is best-effort: a failure here must never block the command the user
// actually ran.
//
// It intentionally does not call CleanupAllClients: this path is reachable from within another
// plugin's own RunCommand (via CoreCLIHelper.RunPeerPlugin -> Run -> auto-install), and a global
// cleanup there would kill that caller's still-running client out from under it. PostInstall
// kills only the client it dispensed for itself.
func runPostInstallHook(ctx context.Context, config *config.Config, fs afero.Fs, p *Plugin, version, previousVersion string, apiBaseURL, dashboardBaseURL, accessBaseURL string) {
	if err := p.PostInstall(ctx, config, fs, version, previousVersion, apiBaseURL, dashboardBaseURL, accessBaseURL); err != nil {
		log.WithFields(log.Fields{
			"prefix": "plugins.plugin.runPostInstallHook",
			"plugin": p.Shortname,
		}).Debugf("plugin PostInstall hook failed: %s", err)
	}
}

// PostInstall calls the plugin's PostInstall hook for the given version, if the plugin
// supports it. Plugins that don't implement DispatcherV3 are silently skipped. Callers
// should treat errors as best-effort and not block the install/upgrade on them.
func (p *Plugin) PostInstall(ctx context.Context, config *config.Config, fs afero.Fs, version, previousVersion string, apiBaseURL, dashboardBaseURL, accessBaseURL string) error {
	logger := log.WithFields(log.Fields{
		"prefix": "plugins.plugin.PostInstall",
	})

	raw, client, err := p.dispensePluginInterface(config, fs, version, "", logger)
	if client != nil {
		defer client.Kill()
	}
	if err != nil {
		return err
	}

	d, ok := raw.(DispatcherV3)
	if !ok {
		logger.Debug("plugin does not support PostInstall")
		return nil
	}

	return d.PostInstall(buildAdditionalInfo(logger, apiBaseURL, dashboardBaseURL, accessBaseURL), version, previousVersion, NewCoreCLIHelper(ctx, config, fs, apiBaseURL, dashboardBaseURL, accessBaseURL))
}

// PreUninstall calls the plugin's PreUninstall hook for the currently installed version, if
// the plugin supports it. Plugins that don't implement DispatcherV3 are silently skipped.
// Callers should treat errors as best-effort and not block the uninstall on them.
func (p *Plugin) PreUninstall(ctx context.Context, config *config.Config, fs afero.Fs, version string, apiBaseURL, dashboardBaseURL, accessBaseURL string) error {
	logger := log.WithFields(log.Fields{
		"prefix": "plugins.plugin.PreUninstall",
	})

	raw, client, err := p.dispensePluginInterface(config, fs, version, "", logger)
	if client != nil {
		defer client.Kill()
	}
	if err != nil {
		return err
	}

	d, ok := raw.(DispatcherV3)
	if !ok {
		logger.Debug("plugin does not support PreUninstall")
		return nil
	}

	return d.PreUninstall(buildAdditionalInfo(logger, apiBaseURL, dashboardBaseURL, accessBaseURL), version, NewCoreCLIHelper(ctx, config, fs, apiBaseURL, dashboardBaseURL, accessBaseURL))
}
