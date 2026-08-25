// Package pluginhints provides placeholder Cobra commands for known plugins
// that are not yet installed, guiding users to install or request access.
package pluginhints

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/cmd/plugin/postinstall"
	"github.com/stripe/stripe-cli/pkg/cmdutil"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// AutoInstallOptOutEnvVar disables auto-installing plugins on first use when set
// to a truthy value. Environments that don't want the CLI fetching binaries
// on demand (locked-down CI, air-gapped hosts) set this and install explicitly
// with `stripe plugin install <name>`.
const AutoInstallOptOutEnvVar = "STRIPE_CLI_NO_PLUGIN_AUTO_INSTALL"

// PluginRunner dispatches to an installed plugin binary. cmd is the command that
// triggered the run (used for its context), name is the plugin shortname, and
// args are the arguments to forward to the plugin process.
type PluginRunner func(cmd *cobra.Command, name string, args []string) error

// AddHintCommands registers a hint command for each known plugin that is not
// present in installedPluginSet. runPlugin dispatches to a plugin binary that was
// installed during this invocation; plugins configured with withAutoInstall use it
// to run the user's original command once the install finishes. A nil runPlugin
// degrades auto-install to install-then-print-next-steps.
func AddHintCommands(rootCmd *cobra.Command, cfg *config.Config, installedPluginSet map[string]bool, runPlugin PluginRunner) {
	if !installedPluginSet["apps"] {
		rootCmd.AddCommand(
			newPluginHintCmd(cfg, "apps", "This plugin lets you build and manage Stripe Apps.").Command,
		)
		rootCmd.Annotations["apps"] = "available_plugin"
	}
	if !installedPluginSet["generate"] {
		rootCmd.AddCommand(
			newPluginHintCmd(cfg, "generate", "This plugin creates skeleton files to get you started.", withPrivatePreview()).Command,
		)
		rootCmd.Annotations["generate"] = "available_plugin"
	}
	if !installedPluginSet["projects"] {
		rootCmd.AddCommand(
			newPluginHintCmd(cfg, "projects", "This plugin scaffolds and manages Stripe integration projects.").Command,
		)
		rootCmd.Annotations["projects"] = "available_plugin"
	}
	if !installedPluginSet["directory"] {
		directoryCmd := newPluginHintCmd(
			cfg,
			"directory",
			"Allow your agent to search and provision tools and services. Learn more: https://stripe.directory",
			withAutoInstall(runPlugin),
		).Command
		// These aliases only make the command discoverable through a near miss; they
		// do not auto-install. See invokedByName.
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
		rootCmd.Annotations["directory"] = "available_plugin"
	}
	if !installedPluginSet["tools"] {
		rootCmd.AddCommand(
			newPluginHintCmd(cfg, "tools", "Search, inspect, and execute Stripe operations not available in the public API.").Command,
		)
		rootCmd.Annotations["tools"] = "available_plugin"
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
	accessBaseURL  string

	// autoInstall opts this plugin into installing on first use without prompting
	// and then running the command the user originally typed. The metadata endpoint
	// still has the final say per machine; see autoInstallEnabled.
	autoInstall bool

	lookupFn      func(ctx context.Context) (*plugins.ResolvedPluginVersion, error)
	installFn     func(ctx context.Context) error
	loginFn       func(ctx context.Context) error
	runPluginFn   func(cmd *cobra.Command, args []string) error
	accountIDFn   func() (string, error)
	openBrowserFn func(url string) error
	argvFn        func() []string
	lookupEnvFn   func(key string) string
	stdin         io.Reader
	stdout        io.Writer
	// stderr carries progress and next steps for the non-interactive auto-install
	// path, so a first run leaves stdout holding only the plugin's own output and
	// stays parseable by whatever asked for it.
	stderr io.Writer
}

type option func(*pluginHintCmd)

func withPrivatePreview() option {
	return func(p *pluginHintCmd) {
		p.privatePreview = true
	}
}

// withAutoInstall installs the plugin on first use and hands off to runPlugin so
// the user's original command runs in the same invocation. A nil runPlugin still
// skips the confirmation prompt, but can only print next steps afterwards.
func withAutoInstall(runPlugin PluginRunner) option {
	return func(p *pluginHintCmd) {
		p.autoInstall = true
		if runPlugin == nil {
			return
		}
		p.runPluginFn = func(cmd *cobra.Command, args []string) error {
			return runPlugin(cmd, p.name, args)
		}
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
		name:          name,
		description:   description,
		accessBaseURL: login.DefaultAccessBaseURL,
		lookupFn:      resolvePlugin,
		installFn: func(ctx context.Context) error {
			resolvedPlugin, err := resolvePlugin(ctx)
			if err != nil {
				return err
			}
			return resolvedPlugin.Install(ctx, cfg, fs, stripe.DefaultAPIBaseURL, dashboardBaseURL)
		},
		accountIDFn:   cfg.GetProfile().GetAccountID,
		openBrowserFn: open.Browser,
		argvFn:        func() []string { return os.Args },
		lookupEnvFn:   os.Getenv,
		stdin:         os.Stdin,
		stdout:        os.Stdout,
		stderr:        os.Stderr,
	}
	p.loginFn = func(ctx context.Context) error {
		return login.Login(ctx, dashboardBaseURL, p.accessBaseURL, cfg)
	}

	for _, opt := range opts {
		opt(p)
	}

	p.initCommand()

	return p
}

// initCommand wires up the Cobra command. It is separate from newPluginHintCmd so
// tests can build a pluginHintCmd with mocked side effects and still exercise the
// real flag and help handling.
func (p *pluginHintCmd) initCommand() {
	p.Command = &cobra.Command{
		Use:   p.name,
		Short: p.description,
		// Accept unknown flags/args so they aren't rejected before we can show the hint
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		RunE:               p.run,
	}
	p.Command.Flags().StringVar(&p.accessBaseURL, "access-base", login.DefaultAccessBaseURL, "Sets the access base URL")
	_ = p.Command.Flags().MarkHidden("access-base")

	if p.autoInstall {
		p.setAutoInstallHelpFunc()
	}
}

func (p *pluginHintCmd) run(cmd *cobra.Command, args []string) error {
	if err := login.ValidateAccessBaseURL(p.accessBaseURL); err != nil {
		return err
	}

	ctx := commandContext(cmd)

	// Honor the opt-out before resolving plugin metadata. Lookup can fail in the
	// locked-down and air-gapped environments this setting is intended for, and
	// falling through after that failure could otherwise trigger an interactive
	// login prompt and consume stdin.
	if p.autoInstall && p.autoInstallOptedOut() {
		return p.refuseAutoInstall()
	}

	if resolved, err := p.lookupFn(ctx); err == nil {
		switch {
		case p.autoInstallEnabled(resolved) && p.invokedByName(cmd):
			return p.autoInstallAndRun(ctx, cmd, p.pluginArgs())
		default:
			return p.promptInstall(ctx)
		}
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

// setAutoInstallHelpFunc makes `stripe <plugin> --help` install the plugin and then
// let the plugin document itself, since asking what a command does is a likely first
// interaction and this placeholder's one-line summary barely answers it.
func (p *pluginHintCmd) setAutoInstallHelpFunc() {
	// Capture Cobra's renderer before overriding it, so the fallback below prints the
	// placeholder help instead of recursing back into this hook.
	placeholderHelp := p.HelpFunc()

	p.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// Without a runner there is nothing to hand the help request to, so installing
		// would not produce better help than the placeholder already gives. Reaching
		// help through an alias is likewise not a clear enough signal to fetch a
		// binary, so let the placeholder answer that too.
		if p.runPluginFn == nil || !p.invokedByName(cmd) {
			placeholderHelp(cmd, args)
			return
		}

		handedOff, err := p.autoInstallHelp(cmd, args)
		if handedOff {
			return
		}

		placeholderHelp(cmd, args)

		if err != nil {
			// Cobra's help hook cannot report an error, and printing nothing would be
			// worse than the placeholder, so say why the plugin's own help is missing
			// and give the one command that fixes it.
			fmt.Fprintf(p.stdout, "\nThe %q plugin's own help is unavailable: %v\n", p.name, err)
			fmt.Fprintf(p.stdout, "Run 'stripe plugin install %s' to install it and see its full help.\n", p.name)
		}
	})
}

// autoInstallHelp installs the plugin and forwards the help request to it. It
// reports whether the plugin answered the help request; when it did not, a
// non-nil error means the caller should explain why, and a nil error means the
// placeholder help is the whole answer and nothing needs explaining.
func (p *pluginHintCmd) autoInstallHelp(cmd *cobra.Command, cobraArgs []string) (bool, error) {
	// The caller prints the install command, so this only has to say why the plugin
	// was not fetched.
	if p.autoInstallOptedOut() {
		return false, errorcategory.Errorf(errorcategory.UserInput, "%s disables installing it automatically", AutoInstallOptOutEnvVar)
	}

	ctx := commandContext(cmd)

	resolved, err := p.lookupFn(ctx)
	if err != nil {
		return false, err
	}

	// A machine the auto-install rollout has not reached gets the placeholder help
	// with no commentary, exactly like every other not-yet-installed plugin. The
	// rollout is not the user's business, so there is nothing to explain.
	if !p.autoInstallEnabled(resolved) {
		return false, nil
	}

	if err := p.autoInstallAndRun(ctx, cmd, p.helpArgs(cobraArgs)); err != nil {
		return false, err
	}

	return true, nil
}

// autoInstallAndRun installs the plugin without prompting and then runs the
// command the user typed, so first use of the plugin costs a one-time download
// rather than a canceled command the user has to retype.
func (p *pluginHintCmd) autoInstallAndRun(ctx context.Context, cmd *cobra.Command, pluginArgs []string) error {
	// This is progress, not output the caller asked for, so it goes to stderr. A
	// first run then leaves stdout holding only what the plugin itself printed,
	// which keeps `--format json` and friends parseable.
	fmt.Fprintln(p.stderr, ansi.Faint(fmt.Sprintf("Installing the %q plugin (one-time setup)...", p.name)))

	if err := p.installFn(ctx); err != nil {
		return err
	}

	// Print the next steps before handing off, so this one-time guidance stays
	// above the requested command's output instead of trailing it, where it would
	// look like part of the result.
	color := ansi.Color(p.stderr)
	fmt.Fprintln(p.stderr, color.Green("✔ installation complete."))
	postinstall.PrintTips(p.stderr, p.name)

	// Without a runner there is nothing to hand off to, so stop here rather than
	// exiting on a silent success.
	if p.runPluginFn == nil {
		return nil
	}

	fmt.Fprintln(p.stderr)

	return p.runPluginFn(cmd, pluginArgs)
}

// refuseAutoInstall reports that the plugin is missing without fetching it, so an
// environment that opted out fails loudly instead of blocking on a prompt that
// may have no one to answer it. The whole explanation lives in the error because
// the host already prints that to stderr and exits non-zero.
func (p *pluginHintCmd) refuseAutoInstall() error {
	return errorcategory.Errorf(
		errorcategory.UserInput,
		"the %q plugin is required to run this command, but %s disables installing it automatically; run 'stripe plugin install %s'",
		p.name, AutoInstallOptOutEnvVar, p.name,
	)
}

// autoInstallEnabled reports whether this invocation may install the plugin
// without asking. Both halves have to agree: the CLI has to opt the plugin in,
// and the metadata endpoint has to say the auto-install rollout has reached this
// machine. Anything else — an older server, a machine outside the rollout, a
// resolution that fell back to cached metadata — keeps today's prompt.
func (p *pluginHintCmd) autoInstallEnabled(resolved *plugins.ResolvedPluginVersion) bool {
	return p.autoInstall && resolved != nil && resolved.AutoInstall
}

func (p *pluginHintCmd) autoInstallOptedOut() bool {
	optedOut, err := strconv.ParseBool(p.lookupEnvFn(AutoInstallOptOutEnvVar))
	return err == nil && optedOut
}

// pluginArgs recovers the arguments intended for the plugin from the raw process
// arguments. Cobra has already consumed the flags and the plugin name it
// recognizes, so read them back from argv instead:
// "stripe [host_flags...] directory [plugin_args...]" => "[plugin_args...]".
//
// Only the auto-install path forwards arguments, and invokedByName gates that on
// the plugin's own name appearing in argv, so slicing after p.name is enough.
func (p *pluginHintCmd) pluginArgs() []string {
	return cmdutil.ArgsAfter(p.argvFn(), p.name)
}

// invokedByName reports whether the user reached this command by the plugin's real
// name rather than through one of its aliases. Installing a binary without asking
// is only reasonable when there is no doubt about what was meant, and an alias —
// a typo alias especially — leaves that doubt, so aliases fall back to the same
// prompt every other not-yet-installed plugin uses.
func (p *pluginHintCmd) invokedByName(cmd *cobra.Command) bool {
	if cmd != nil && cmd.CalledAs() != "" {
		return cmd.CalledAs() == p.name
	}

	// Cobra records no CalledAs on the target of `stripe help <cmd>`, so read the
	// name the user typed back from argv.
	return slices.Contains(p.argvFn(), p.name)
}

// helpArgs builds the arguments that make the plugin print the help the user asked
// for. Cobra passes the raw arguments through when help was requested with a flag,
// but passes none when it came from the `help` subcommand — and in that case argv
// holds no help flag for the plugin to act on, so one has to be added.
func (p *pluginHintCmd) helpArgs(cobraArgs []string) []string {
	args := p.pluginArgs()

	if len(cobraArgs) == 0 {
		// "stripe help directory [plugin_subcommands...]" => "[plugin_subcommands...] --help"
		return append(args, "--help")
	}

	// "stripe directory [plugin_subcommands...] --help" => "[plugin_subcommands...] --help"
	return args
}

// commandContext returns the command's context, falling back to a background one
// when Cobra never attached one — as happens for the target of `stripe help <cmd>`.
func commandContext(cmd *cobra.Command) context.Context {
	if cmd == nil || cmd.Context() == nil {
		return context.Background()
	}

	return cmd.Context()
}

func (p *pluginHintCmd) promptInstall(ctx context.Context) error {
	fmt.Fprintf(p.stdout, "The \"%s\" plugin is required to run this command.\n", p.name)
	fmt.Fprintf(p.stdout, "\n")
	fmt.Fprintf(p.stdout, "%s\n", p.description)
	fmt.Fprintf(p.stdout, "You can run 'stripe plugin install %s' or press Enter to install", p.name)

	var input string
	fmt.Fscanln(p.stdin, &input)

	if input != "" {
		return errorcategory.Errorf(errorcategory.UserInput, "installation canceled")
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
		return errorcategory.Errorf(errorcategory.UserInput, "login canceled")
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

	fmt.Fprintf(p.stdout, "The logged-in account %s does not have access to the private preview 'generate' plugin. Log in to a different account with 'stripe login', or contact Stripe support.\n", accountID)
	fmt.Fprintf(p.stdout, "\n")
	fmt.Fprintf(p.stdout, "%s\n", p.description)
	os.Exit(1)

	return nil
}
