package agentsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const (
	ClientOpenclaw      = "openclaw"
	OpenclawBinaryName  = "openclaw"
	OpenclawPluginName  = "stripe"
	OpenclawDisplayName = "Openclaw"

	// openclawRepoDir is relative to $HOME. repos is a new folder we create within
	// .openclaw that none of .openclaw's functionality is reliant on.
	openclawRepoDir     = ".openclaw/repos/stripe"
	openclawListTimeout = 5 * time.Second
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
		fmt.Println("Error running openclaw plugins list:", err) // TESTING
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
		fmt.Println(plugin) // TESTING
		fmt.Println(openclawPluginIsStripe(plugin)) // TESTING
		if openclawPluginIsStripe(plugin) {
			return plugin, true
		}
	}
	return openclawInstalledPlugin{}, false
}

func openclawPluginIsStripe(plugin openclawInstalledPlugin) bool {
	// We have the user install our plugin directly from our GitHub repo, so we want to make sure
	// this doesn't get confused with a native Openclaw plugin
	return strings.EqualFold(plugin.Name, OpenclawPluginName)
}

func (p OpenclawProvider) repoPath() (string, error) {
	home, err := p.Scanner.withDefaults().HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, openclawRepoDir), nil
}

// Plan for Openclaw, installing and updating the plugin requires either cloning or pulling the repo beforehand,
// however installing the plugin after uses the same command `openclaw plugins install`.
func (p OpenclawProvider) Plan(status Status, force bool) Plan {
	repoPath, err := p.repoPath()
	if err != nil {
		return Plan{
			Action: ActionManual,
			Manual: fmt.Sprintf("Could not resolve home directory: %s", err),
		}
	}

	fmt.Println("repoPath:", repoPath) //TESTING
	p.Detect() //TESTING

	installCommand := []string{p.BinaryName, "plugins", "install", "--force", "--accept-capabilities", repoPath}

	makeRepoDirectoryCommand := []string{"mkdir", "-p", repoPath}
	gitInstallCommand := []string{gitBinaryName, "clone", "--branch", "plugins/agent-plugin", "--depth", "1", "https://github.com/stripe/ai.git", repoPath}
	installCommands := [][]string{makeRepoDirectoryCommand, gitInstallCommand, installCommand}

	gitPullCommand := []string{gitBinaryName, "pull", "-C", repoPath}
	reinstallCommands := [][]string{gitPullCommand, installCommand}

	return getPlanByStatus(status, force, installCommands, reinstallCommands)
}

// Apply runs each command in the plan in order. The install step is already
// the last entry in plan.Commands (see Plan), so there is nothing left to run
// after this loop.
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

	return nil
}
