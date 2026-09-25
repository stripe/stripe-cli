package agentsetup


import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"
	"fmt"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const (
	ClientOpenclaw      = "openclaw"
	OpenclawBinaryName  = "openclaw"
	OpenclawPluginName  = "stripe"
	OpenclawDisplayName = "Openclaw"

	openclawClonedRepoPath = "~/.openclaw/repos/stripe" // repos is a new folder we create within .openclaw that none of .openclaw's functionality is reliant on
	openclawListTimeout    = 5 * time.Second
)

// OpenclawProvider detects and installs the Stripe plugin for Openclaw.
type OpenclawProvider struct {
	ProviderConfig
}

// NewOpenclawProvider returns a Openclaw setup provider.
func NewOpenclawProvider(scanner Scanner, runCommand RunCommandFunc) Provider {
	if runCommand == nil {
		runCommand = RunCommand
	}
	return OpenclawProvider{
		ProviderConfig: ProviderConfig{
			Scanner:     scanner,
			Client:      ClientOpenclaw,
			BinaryName:  OpenclawBinaryName,
			DisplayName: OpenclawDisplayName,
			RunCommand:  runCommand,
			RunOutput:   runCommandOutput,
		},
	}
}

func (p OpenclawProvider) ID() string { return p.Client }

func (p OpenclawProvider) Detect() Status {
	status := detectAgentExecutable(
		p.ProviderConfig,
		StatusMissing,
	)
	if !status.Detected {
		return status
	}

	ctx, cancel := context.WithTimeout(context.Background(), openclawListTimeout)
	defer cancel()

	plugin, ok, supportsPlugins := p.stripePluginStatus(ctx)
	if !supportsPlugins {
		status.Error = "upgrade Openclaw Build to enable plugin support"
		return status
	}
	if ok {
		status.Plugin.Installed = true
		status.Plugin.ID = plugin.Name
		status.Plugin.Version = plugin.Version
		status.Plugin.StatePath = plugin.Source
		status.Plugin.Scope = "user"
		status.Status = StatusInstalled
	}

	return status
}

// stripePluginStatus runs `openclaw plugins list --json` and reports whether the
// Stripe plugin is installed. When the command fails (e.g. an old Openclaw
// version without plugin support), supportsPlugins is false.
func (p OpenclawProvider) stripePluginStatus(ctx context.Context) (plugin openclawInstalledPlugin, installed bool, supportsPlugins bool) {
	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}
	out, err := runOutput(ctx, p.BinaryName, "plugins", "list", "--json")
	if err != nil {
		return openclawInstalledPlugin{}, false, false
	}
	plugin, ok := findOpenclawStripePlugin(out)
	return plugin, ok, true
}

type openclawInstalledPlugin struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
	Format  string `json:"format"` // Whether a plugin is a built-in Openclaw plugin or a third-party "bundled plugin"
}

// findOpenclawStripePlugin reports whether the Stripe plugin appears in the
// output of `openclaw plugins list --json`.
func findOpenclawStripePlugin(listJSON []byte) (openclawInstalledPlugin, bool) {
	var plugins []openclawInstalledPlugin
	if err := json.Unmarshal(listJSON, &plugins); err != nil {
		return openclawInstalledPlugin{}, false
	}

	for _, plugin := range plugins {
		fmt.Println(plugin)
		fmt.Println(openclawPluginIsStripe(plugin))
		if openclawPluginIsStripe(plugin) {
			return plugin, true
		}
	}
	return openclawInstalledPlugin{}, false
}

func openclawPluginIsStripe(plugin openclawInstalledPlugin) bool {
	// We have the user install our plugin directly from our GitHub repo, so we want to make sure
	// this doesn't get confused with a native Openclaw plugin
	return strings.EqualFold(plugin.Name, OpenclawPluginName) && strings.EqualFold(plugin.Format, "bundle")
}

// Plan for Openclaw, installing and updating the plugin requires either cloning or pulling the repo beforehand,
// however installing the plugin after uses the same command `openclaw plugins install`.
// So, Plan should instead return the appropriate git command to run first depending on the Action
func (p OpenclawProvider) Plan(status Status, force bool) Plan {
	makeRepoDirectoryCommand := []string{"mkdir", "-p", openclawClonedRepoPath}
	gitInstallCommand := []string{gitBinaryName, "clone", "--branch", "plugins/agent-plugin", "--depth", "1", "https://github.com/stripe/ai.git", openclawClonedRepoPath}
	preInstallCommands := [][]string{makeRepoDirectoryCommand, gitInstallCommand}

	gitReinstallCommands := [][]string{{gitBinaryName, "pull", "-C", openclawClonedRepoPath}}

	return getPlanByStatus(status, force, preInstallCommands, gitReinstallCommands)
}

func (p OpenclawProvider) Apply(ctx context.Context, _ io.Writer, plan Plan) error {
	if plan.Action == ActionNone {
		return nil
	}
	if len(plan.Commands) == 0 {
		return errorcategory.Errorf(errorcategory.Internal, "missing command for %s action", plan.Action)
	}

	for _, command := range plan.Commands {
		if err := p.RunCommand(ctx, command[0], command[1:]...); err != nil {
			return err
		}
	}

	installName, installArgs := p.installCommand()
	return p.RunCommand(ctx, installName, installArgs...)
}

// installCommand returns the command to install the Stripe plugin for Openclaw using a local directory.
func (p OpenclawProvider) installCommand() (string, []string) {
	return p.BinaryName, []string{"plugins", "install", openclawClonedRepoPath}
}
