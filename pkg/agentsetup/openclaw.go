package agentsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const (
	ClientOpenclaw      = "openclaw"
	OpenclawBinaryName  = "openclaw"
	OpenclawPluginName  = "stripe"
	OpenclawDisplayName = "OpenClaw"

	openclawRepoPlaceholder = "<temporary-directory>/repo"
	openclawListTimeout     = 5 * time.Second
)

// OpenclawProvider detects and installs the Stripe plugin for Openclaw.
type OpenclawProvider struct {
	ProviderConfig
	mkdirTemp func(string, string) (string, error)
	removeAll func(string) error
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
		status.Error = "upgrade OpenClaw to enable plugin support"
		return status
	}

	// If home directory cannot be resolved and plugin is not found, return an error status. This is a prerequisite for Openclaw plugin installation.
	_, err := p.Scanner.withDefaults().HomeDir()
	if err != nil && !ok {
		status.Error = "Could not resolve home directory for OpenClaw plugin installation"
		status.Status = StatusError
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
	runCommand := p.RunCommand
	if runCommand == nil {
		runCommand = RunCommand
	}
	out, err := runCommand(ctx, p.BinaryName, "plugins", "list", "--json")
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

type openclawPluginsListOutput struct {
	Plugins []openclawInstalledPlugin `json:"plugins"`
}

// findOpenclawStripePlugin reports whether the Stripe plugin appears in the
// output of `openclaw plugins list --json`.
func findOpenclawStripePlugin(listJSON []byte) (openclawInstalledPlugin, bool) {
	var output openclawPluginsListOutput
	if err := json.Unmarshal(listJSON, &output); err != nil {
		return openclawInstalledPlugin{}, false
	}

	for _, plugin := range output.Plugins {
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

// Plan describes cloning and installing the plugin without allocating a workspace.
func (p OpenclawProvider) Plan(status Status, force bool) Plan {
	installCommand := []string{p.BinaryName, "plugins", "install", "--force", "--accept-capabilities", openclawRepoPlaceholder}
	gitInstallCommand := []string{gitBinaryName, "clone", "--branch", "plugins/agent-plugin", "--depth", "1", "https://github.com/stripe/ai.git", openclawRepoPlaceholder}
	commands := [][]string{gitInstallCommand, installCommand}

	return getPlanByStatus(status, force, commands, commands)
}

// Apply clones and installs in an isolated workspace, then attempts cleanup
// regardless of the installation result.
func (p OpenclawProvider) Apply(ctx context.Context, out io.Writer, plan Plan) error {
	if plan.Action == ActionNone {
		return nil
	}
	if len(plan.Commands) == 0 {
		return errorcategory.Errorf(errorcategory.Internal, "missing command for %s action", plan.Action)
	}
	if out == nil {
		out = io.Discard
	}
	if p.mkdirTemp == nil {
		p.mkdirTemp = os.MkdirTemp
	}
	if p.removeAll == nil {
		p.removeAll = os.RemoveAll
	}
	workspace, err := p.mkdirTemp("", "stripe-openclaw-*")
	if err != nil {
		return errorcategory.Errorf(errorcategory.Internal, "creating OpenClaw plugin workspace: %w", err)
	}
	defer func() {
		if err := p.removeAll(workspace); err != nil {
			fmt.Fprintf(out, "Warning: OpenClaw plugin cleanup failed; remove %q manually: %v\n", workspace, err)
			if tc := stripe.GetTelemetryClient(ctx); tc != nil {
				go tc.SendEvent(ctx, "Agent Setup: Plugin Cleanup", "openclaw:error")
			}
		}
	}()

	repoPath := filepath.Join(workspace, "repo")
	for _, command := range plan.Commands {
		resolved := append([]string(nil), command...)
		for i, arg := range resolved {
			if arg == openclawRepoPlaceholder {
				resolved[i] = repoPath
			}
		}
		if _, err := p.RunCommand(ctx, resolved[0], resolved[1:]...); err != nil {
			fmt.Fprintf(out, "OpenClaw plugin %s step failed: %v\n", resolved[0], err)
			return err
		}
	}

	return nil
}
