package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// InstallCmd is the struct used for configuring the plugin install command
type InstallCmd struct {
	cfg *config.Config
	Cmd *cobra.Command
	fs  afero.Fs

	archiveURL     string
	archivePath    string
	localPluginDir string

	apiBaseURL string
}

// NewInstallCmd creates a command for installing plugins
func NewInstallCmd(config *config.Config) *InstallCmd {
	ic := &InstallCmd{}
	ic.fs = afero.NewOsFs()
	ic.cfg = config

	ic.Cmd = &cobra.Command{
		Use:   "install",
		Args:  validators.MaximumNArgs(1),
		Short: "Install a Stripe CLI plugin",
		Long: `Install a Stripe CLI plugin. To download a specific version, run stripe install [plugin_name]@[version].
			By default, the most recent version will be installed.`,
		RunE: ic.runInstallCmd,
	}

	ic.Cmd.Flags().StringVar(&ic.archiveURL, "archive-url", "", "Install a plugin by an archive URL")
	ic.Cmd.Flags().StringVar(&ic.archivePath, "archive-path", "", "Install a plugin by a local archive path")
	ic.Cmd.Flags().StringVar(&ic.localPluginDir, "local", "", "Install a development version plugin from a local development folder")
	ic.Cmd.Flags().Bool("archive", false, "Install a plugin by archive data from stdout")

	// Hidden configuration flags, useful for dev/debugging
	ic.Cmd.Flags().StringVar(&ic.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	ic.Cmd.Flags().MarkHidden("api-base") // #nosec G104

	return ic
}

// parsePluginArg takes in the argument and returns the name of the plugin to download and the version
// Ex: parseArg('plugin') -> 'plugin', nil
// Ex: parseArg('plugin@2.0.2') -> 'plugin', '2.0.2'
func parseInstallArg(arg string) (string, string) {
	args := strings.Split(arg, "@")
	plugin := args[0]
	version := ""
	if len(args) > 1 {
		version = args[1]
	}

	return plugin, version
}

func (ic *InstallCmd) installPluginByName(cmd *cobra.Command, arg string) error {
	pluginName, version := parseInstallArg(arg)

	if pluginName == "-" && hasPipedData() {
		// -: reads data from stdout
		return plugins.ExtractStdoutArchive(cmd.Context(), ic.cfg)
	}

	plugin, err := plugins.LookUpPlugin(cmd.Context(), ic.cfg, ic.fs, pluginName)

	if err != nil {
		return err
	}

	if len(version) == 0 {
		version = plugin.LookUpLatestVersion()
	}

	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "cmd.installCmd.runInstallCmd",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	err = plugin.Install(ctx, ic.cfg, ic.fs, version, ic.apiBaseURL)

	return err
}

func (ic *InstallCmd) installPluginByArchive(cmd *cobra.Command) error {
	switch {
	case ic.archiveURL == "" && ic.archivePath == "":
		// no arhive URL or path was provided. try to read from piped stdin
		readFromArchive, err := cmd.Flags().GetBool("archive")
		if err != nil {
			return err
		}

		if readFromArchive {
			if !hasPipedData() {
				return fmt.Errorf("Please pipe into stdout: curl <url> | stripe plugin install --archive")
			}

			return plugins.ExtractStdoutArchive(cmd.Context(), ic.cfg)
		}

		return fmt.Errorf("To install a plugin from archive, please provide archive url/path or pipe archive data into stdout")
	case ic.archiveURL != "":
		return plugins.FetchAndExtractRemoteArchive(cmd.Context(), ic.cfg, ic.archiveURL)
	case ic.archivePath != "":
		return plugins.ExtractLocalArchive(cmd.Context(), ic.cfg, ic.archivePath)
	}

	return nil
}

func (ic *InstallCmd) installPluginFromLocalDir() error {
	os.Chdir(ic.localPluginDir)

	cmd := exec.Command("make", "install")
	stdout, err := cmd.Output()
	if err != nil {
		return err
	}

	// Print the output
	fmt.Println(string(stdout))
	return nil
}

func (ic *InstallCmd) runInstallCmd(cmd *cobra.Command, args []string) error {
	var err error
	color := ansi.Color(os.Stdout)

	// check if plugin manfest exists to be updated with the plugin to be installed
	configPath := ic.cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginManifestPath := filepath.Join(configPath, "plugins.toml")
	_, err = afero.ReadFile(ic.fs, pluginManifestPath)
	if os.IsNotExist(err) {
		// plugin manifest does not exist. will need to retrieve from web
		// api key is required to retrieve the plugin manifest
		_, err = ic.cfg.GetProfile().GetAPIKey(false)
		if err != nil {
			fmt.Println(color.Red("x could not install plugin. please run `stripe login` and try again"))
			return fmt.Errorf("installation process exited")
		}
	}

	if len(args) == 0 {
		if ic.localPluginDir != "" {
			err = ic.installPluginFromLocalDir()
			if err != nil {
				return err
			}
		} else {
			err = ic.installPluginByArchive(cmd)
			if err != nil {
				return err
			}
		}
	} else {
		// Refresh the plugin before proceeding
		err = plugins.RefreshPluginManifest(cmd.Context(), ic.cfg, ic.fs, ic.apiBaseURL)
		if err != nil {
			return err
		}

		err = ic.installPluginByName(cmd, args[0])
		if err != nil {
			return err
		}
	}

	if err == nil {
		fmt.Println(color.Green("✔ installation complete."))
	}

	return nil
}

func withSIGTERMCancel(ctx context.Context, onCancel func()) context.Context {
	// Create a context that will be canceled when Ctrl+C is pressed
	ctx, cancel := context.WithCancel(ctx)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-interruptCh
		onCancel()
		cancel()
	}()
	return ctx
}

func hasPipedData() bool {
	info, _ := os.Stdin.Stat()
	return info.Mode()&os.ModeCharDevice == 0
}
