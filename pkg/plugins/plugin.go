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

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
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

// Plugin contains the plugin properties
type Plugin struct {
	Shortname        string
	Binary           string
	Releases         []Release `toml:"Release"`
	MagicCookieValue string
}

// PluginList contains a list of plugins
type PluginList struct {
	Plugins []Plugin `toml:"Plugin"`
}

// Release is the type that holds release data for a specific build of a plugin
type Release struct {
	Arch    string
	OS      string
	Version string
	Sum     string
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
	}

	return handshakeConfig, pluginSetMap
}

// getPluginInstallPath computes the absolute path of a specific plugin version's installation dir
func (p *Plugin) getPluginInstallPath(config config.IConfig, version string) string {
	pluginsDir := getPluginsDir(config)
	pluginPath := filepath.Join(pluginsDir, p.Shortname, version)

	return pluginPath
}

// cleanUpPluginPath empties the plugin folder except for the version specified
func (p *Plugin) cleanUpPluginPath(config config.IConfig, fs afero.Fs, versionToKeep string) error {
	logger := log.WithFields(log.Fields{
		"prefix": "plugins.plugin.cleanUpPluginPath",
	})
	logger.Debug("cleaning up other plugin versions...")

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
			logger.Debugf("skipping directory: %s", path)
			return nil
		case info.IsDir() && path == versionPathToKeep:
			logger.Debugf("skipping directory: %s", path)
			return filepath.SkipDir
		default:
			logger.Debugf("removing old plugin: %s", path)
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

// Install installs the plugin of the given version
func (p *Plugin) Install(ctx context.Context, config config.IConfig, fs afero.Fs, version string, baseURL string) error {
	spinner := ansi.StartNewSpinner(ansi.Faint(fmt.Sprintf("installing '%s' v%s...", p.Shortname, version)), os.Stdout)

	apiKey, err := config.GetProfile().GetAPIKey(false)

	if err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s': missing API key", p.Shortname)), os.Stdout)
		return err
	}

	pluginData, err := requests.GetPluginData(ctx, baseURL, stripe.APIVersion, apiKey, config.GetProfile())

	if err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s': unauthorized", p.Shortname)), os.Stdout)
		return err
	}

	pluginDownloadURL := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pluginData.PluginBaseURL, p.Shortname, version, runtime.GOOS, runtime.GOARCH, p.Binary)

	// Pull down bin, verify, and save to disk
	err = p.downloadAndSavePlugin(config, pluginDownloadURL, fs, version)

	if err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("could not install plugin '%s'", p.Shortname)), os.Stdout)
		return err
	}

	// Once the plugin is successfully downloaded, clean up other versions
	p.cleanUpPluginPath(config, fs, version)

	ansi.StopSpinner(spinner, ansi.Faint(""), os.Stdout)

	return nil
}

func (p *Plugin) downloadAndSavePlugin(config config.IConfig, pluginDownloadURL string, fs afero.Fs, version string) error {
	logger := log.WithFields(log.Fields{
		"prefix": "plugins.plugin.Install",
	})

	pluginDir := p.getPluginInstallPath(config, version)
	pluginFilePath := filepath.Join(pluginDir, p.Binary)

	logger.Debugf("installing %s to %s...", p.Shortname, pluginFilePath)

	body, err := FetchRemoteResource(pluginDownloadURL)

	if err != nil {
		return err
	}

	reader := bytes.NewReader(body)

	err = p.verifyChecksum(reader, version)

	if err != nil {
		logger.Debug("checksum mismatch")
		return err
	}

	err = fs.MkdirAll(pluginDir, 0755)

	if err != nil {
		logger.Debugf("could not create plugin directory: %s", pluginDir)
		return err
	}

	err = afero.WriteFile(fs, pluginFilePath, body, 0755)

	if err != nil {
		logger.Debug("could not save plugin to disk")
		return err
	}

	return nil
}

// verifyChecksum is to be used during installation only
// hcplugins takes care of the boot time verification for us
func (p *Plugin) verifyChecksum(binary io.Reader, version string) error {
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

// Run boots up the binary and then sends the command to it via RPC
func (p *Plugin) Run(ctx context.Context, config *config.Config, fs afero.Fs, args []string) error {
	var version string

	if PluginsPath != "" {
		version = "master"
	} else {
		// first perform a naive glob of the plugins/name dir for an existing version
		localPluginDir := filepath.Join(getPluginsDir(config), p.Shortname, "*.*.*")
		existingLocalPlugin, err := filepath.Glob(localPluginDir)
		if err != nil {
			return err
		}

		// if plugin is not installed locally, then we should install it first
		if len(existingLocalPlugin) == 0 {
			version = p.LookUpLatestVersion()
			err := p.Install(ctx, config, fs, version, stripe.DefaultAPIBaseURL)
			if err != nil {
				return err
			}
		} else {
			version = filepath.Base(existingLocalPlugin[0])
		}
	}

	pluginDir := p.getPluginInstallPath(config, version)
	pluginBinaryPath := filepath.Join(pluginDir, p.Binary)

	cmd := exec.Command(pluginBinaryPath)

	handshakeConfig, pluginSetMap := p.getPluginInterface()

	pluginLogger := hclog.New(&hclog.LoggerOptions{
		Name:  fmt.Sprintf("[plugin:%s]", p.Shortname),
		Level: hclog.LevelFromString("INFO"),
	})

	clientConfig := &hcplugin.ClientConfig{
		HandshakeConfig:  handshakeConfig,
		VersionedPlugins: pluginSetMap,
		Cmd:              cmd,
		SyncStdout:       os.Stdout,
		SyncStderr:       os.Stderr,
		Logger:           pluginLogger,
	}

	sum, err := p.getChecksum(version)
	if err != nil {
		return err
	}

	clientConfig.SecureConfig = &hcplugin.SecureConfig{
		Checksum: sum,
		Hash:     sha256.New(),
	}

	// start by launching the plugin process / binary
	client := hcplugin.NewClient(clientConfig)

	// Connect via RPC to the plugin
	rpcClient, err := client.Client()
	if err != nil {
		return err
	}

	defer client.Kill()

	// Request the plugin's main interface
	raw, err := rpcClient.Dispense("main")
	if err != nil {
		return err
	}

	// get the native golang interface for the plugin so that we can call it directly
	dispatcher := raw.(Dispatcher)

	// run the command that the user specified via args
	_, err = dispatcher.RunCommand(args)

	if err != nil {
		return err
	}

	return nil
}
