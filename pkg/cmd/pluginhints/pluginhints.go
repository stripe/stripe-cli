// Package pluginhints provides placeholder Cobra commands for known plugins
// that are not yet installed, guiding users to install or request access.
package pluginhints

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/cmd/plugin/postinstall"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// AddHintCommands registers a hint command for each known plugin that is not
// present in installedPluginSet.
func AddHintCommands(rootCmd *cobra.Command, cfg *config.Config, installedPluginSet map[string]bool) {
	if !installedPluginSet["apps"] {
		rootCmd.AddCommand(
			newPluginHintCmd(cfg, "apps", "This plugin lets you build and manage Stripe Apps.").Command,
		)
	}
	if !installedPluginSet["generate"] {
		rootCmd.AddCommand(
			newPluginHintCmd(cfg, "generate", "This plugin creates skeleton files to get you started.", withPrivatePreview()).Command,
		)
	}
	if !installedPluginSet["projects"] {
		rootCmd.AddCommand(
			newPluginHintCmd(cfg, "projects", "This plugin scaffolds and manages Stripe integration projects.").Command,
		)
	}
	if !installedPluginSet["directory"] {
		directoryCmd := newPluginHintCmd(cfg, "directory", "Discover businesses on Stripe. Learn more: https://stripe.directory").Command
		directoryCmd.Aliases = []string{
			"search",
			"directry",
			"directary",
			"direcotry", //nolint:misspell // Intentional typo alias.
			"diretory",
		}
		rootCmd.AddCommand(
			directoryCmd,
		)
	}
	if !installedPluginSet["docs"] {
		rootCmd.AddCommand(
			newPluginHintCmd(cfg, "docs", "Browse Stripe documentation and API reference.").Command,
		)
	}
	if !installedPluginSet["tools"] {
		rootCmd.AddCommand(
			newPluginHintCmd(cfg, "tools", "Search, inspect, and execute Stripe operations not available in the public API.").Command,
		)
	}
}

// pluginHintCmd is a placeholder Cobra command registered when a known plugin
// is not installed. It either prompts the user to install the plugin (if
// available) or explains that their account doesn't have access yet.
type pluginHintCmd struct {
	*cobra.Command
	name           string
	description    string
	privatePreview bool

	lookupFn      func(ctx context.Context) error
	installFn     func(ctx context.Context) error
	loginFn       func(ctx context.Context) error
	accountIDFn   func() (string, error)
	openBrowserFn func(url string) error
	stdin         io.Reader
	stdout        io.Writer
}

type option func(*pluginHintCmd)

func withPrivatePreview() option {
	return func(p *pluginHintCmd) {
		p.privatePreview = true
	}
}

func newPluginHintCmd(cfg *config.Config, name, description string, opts ...option) *pluginHintCmd {
	fs := afero.NewOsFs()
	dashboardBaseURL := stripe.DashboardBaseURLForAPIBaseURL(stripe.DefaultAPIBaseURL)
	resolvePlugin := func(ctx context.Context) (*plugins.ResolvedPluginVersion, error) {
		// Reuse the main install resolution path so metadata-first lookup and
		// backward-compatible manifest fallback stay centralized in plugins.
		return plugins.ResolvePluginForInstall(ctx, cfg, fs, name, "", stripe.DefaultAPIBaseURL, dashboardBaseURL)
	}

	p := &pluginHintCmd{
		name:        name,
		description: description,
		lookupFn: func(ctx context.Context) error {
			_, err := resolvePlugin(ctx)
			return err
		},
		installFn: func(ctx context.Context) error {
			resolvedPlugin, err := resolvePlugin(ctx)
			if err != nil {
				return err
			}
			return resolvedPlugin.Install(ctx, cfg, fs, stripe.DefaultAPIBaseURL, dashboardBaseURL)
		},
		loginFn: func(ctx context.Context) error {
			return login.Login(ctx, dashboardBaseURL, cfg)
		},
		accountIDFn:   cfg.GetProfile().GetAccountID,
		openBrowserFn: open.Browser,
		stdin:         os.Stdin,
		stdout:        os.Stdout,
	}

	for _, opt := range opts {
		opt(p)
	}

	p.Command = &cobra.Command{
		Use:   name,
		Short: description,
		// Accept unknown flags/args so they aren't rejected before we can show the hint
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		RunE:               p.run,
	}

	return p
}

func (p *pluginHintCmd) run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := p.lookupFn(ctx); err == nil {
		return p.promptInstall(ctx)
	}

	if p.privatePreview {
		return p.suggestNotAvailable()
	}

	// If the user is not logged in, offer to kick off the login flow rather than
	// a manual install that will also fail unauthenticated.
	accountID, err := p.accountIDFn()
	if err != nil || accountID == "" {
		return p.promptLogin(ctx)
	}

	fmt.Fprintf(p.stdout, "The \"%s\" plugin is not currently available. Run 'stripe plugin install %s' to try installing it manually.\n", p.name, p.name)
	return nil
}

func (p *pluginHintCmd) promptInstall(ctx context.Context) error {
	fmt.Fprintf(p.stdout, "The \"%s\" plugin is required to run this command.\n", p.name)
	fmt.Fprintf(p.stdout, "\n")
	fmt.Fprintf(p.stdout, "%s\n", p.description)
	fmt.Fprintf(p.stdout, "You can run 'stripe plugin install %s' or press Enter to install", p.name)

	var input string
	fmt.Fscanln(p.stdin, &input)

	if input != "" {
		return fmt.Errorf("installation canceled")
	}

	if err := p.installFn(ctx); err != nil {
		return err
	}

	color := ansi.Color(p.stdout)
	fmt.Fprintln(p.stdout, color.Green("✔ installation complete."))
	postinstall.PrintTips(p.stdout, p.name)

	return nil
}

func (p *pluginHintCmd) promptLogin(ctx context.Context) error {
	fmt.Fprintf(p.stdout, "You must be logged in to access the \"%s\" plugin.\n", p.name)
	fmt.Fprintf(p.stdout, "\n")
	fmt.Fprintf(p.stdout, "Press Enter to run 'stripe login', or type anything to cancel")

	var input string
	fmt.Fscanln(p.stdin, &input)

	if input != "" {
		return fmt.Errorf("login canceled")
	}

	return p.loginFn(ctx)
}

func (p *pluginHintCmd) suggestNotAvailable() error {
	accountID, err := p.accountIDFn()

	if err != nil || accountID == "" {
		fmt.Fprintf(p.stdout, "The '%s' plugin is in private preview. Run 'stripe login' to verify your account has access.\n", p.name)
		os.Exit(1)
		return nil
	}

	fmt.Fprintf(p.stdout, "The logged-in account %s does not have access to the private preview 'generate' plugin. Log into a different account with 'stripe login', or contact Stripe support.\n", accountID)
	fmt.Fprintf(p.stdout, "\n")
	fmt.Fprintf(p.stdout, "%s\n", p.description)
	os.Exit(1)

	return nil
}
