// Package agentsetup contains helpers for detecting and configuring AI coding
// agent integrations.
package agentsetup

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const (
	ClientClaudeCode   = "claude-code"
	ClaudeBinaryName   = "claude"
	TargetClaudePlugin = "stripe@anthropic-plugin-directory"
	ClaudeMarketplace  = "anthropic-plugin-directory"
	ClaudeDisplayName  = "Claude Code"

	claudeListTimeout = 5 * time.Second
)

// ClaudeProvider detects and configures the Stripe plugin for Claude Code.
type ClaudeProvider struct {
	ProviderConfig
}

// NewClaudeProvider returns a Claude Code setup provider.
func NewClaudeProvider(scanner Scanner, runCommand RunCommandFunc) Provider {
	if runCommand == nil {
		runCommand = RunCommand
	}
	return ClaudeProvider{
		ProviderConfig: ProviderConfig{
			Scanner:     scanner,
			Client:      ClientClaudeCode,
			BinaryName:  ClaudeBinaryName,
			DisplayName: ClaudeDisplayName,
			RunCommand:  runCommand,
		},
	}
}

func (p ClaudeProvider) ID() string {
	return p.Client
}

func (p ClaudeProvider) Detect() Status {
	status := detectAgentExecutable(
		p.ProviderConfig,
		StatusMissing,
	)
	if !status.Detected {
		return status
	}

	ctx, cancel := context.WithTimeout(context.Background(), claudeListTimeout)
	defer cancel()

	pluginID, version, scope, ok, supportsPlugins := p.stripePluginStatus(ctx, status.ExecutablePath)
	if !supportsPlugins {
		status.Error = "upgrade Claude Code to enable plugin support"
		return status
	}
	if ok {
		status.Plugin.Installed = true
		status.Plugin.ID = pluginID
		status.Plugin.Version = version
		status.Plugin.Scope = scope
		status.Status = StatusInstalled
	}

	return status
}

func (p ClaudeProvider) Plan(status Status, force bool) Plan {
	installOrReinstallCommand := [][]string{{status.ExecutablePath, "plugin", "install", TargetClaudePlugin}}
	return getPlanByStatus(status, force, installOrReinstallCommand, installOrReinstallCommand)
}

// Apply installs the Stripe Claude Code plugin. On failure it silently refreshes
// the official marketplace and retries once.
func (p ClaudeProvider) Apply(ctx context.Context, _ io.Writer, plan Plan) error {
	if plan.Action == ActionNone {
		return nil
	}
	if len(plan.Commands) == 0 {
		return errorcategory.Errorf(errorcategory.Internal, "missing command for %s action", plan.Action)
	}
	command := plan.Commands[0]
	name, installArgs := command[0], command[1:]
	if _, err := p.RunCommand(ctx, name, installArgs...); err == nil {
		return nil
	}

	if _, updateErr := p.RunCommand(ctx, name, "plugin", "marketplace", "update", ClaudeMarketplace); updateErr != nil {
		return updateErr
	}
	_, err := p.RunCommand(ctx, name, installArgs...)
	return err
}

// stripePluginStatus runs `claude plugin list --json` and reports whether the
// Stripe plugin is installed. When the command fails (e.g. old Claude version
// without plugin support), supportsPlugins is false.
func (p ClaudeProvider) stripePluginStatus(ctx context.Context, executablePath string) (id, version, scope string, installed bool, supportsPlugins bool) {
	runCommand := p.RunCommand
	if runCommand == nil {
		runCommand = RunCommand
	}
	out, err := runCommand(ctx, executablePath, "plugin", "list", "--json")
	if err != nil {
		return "", "", "", false, false
	}
	pluginID, v, s, ok := findClaudeStripePlugin(out)
	return pluginID, v, s, ok, true
}

// claudeInstalledPlugin is an entry in `claude plugin list --json` output.
type claudeInstalledPlugin struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Scope   string `json:"scope"`
	Enabled bool   `json:"enabled"`
}

// findClaudeStripePlugin reports whether the Stripe plugin appears in the
// output of `claude plugin list --json`.
func findClaudeStripePlugin(listJSON []byte) (id, version, scope string, found bool) {
	var plugins []claudeInstalledPlugin
	if err := json.Unmarshal(listJSON, &plugins); err != nil {
		return "", "", "", false
	}

	for _, plugin := range plugins {
		if claudePluginIsStripe(plugin) {
			return plugin.ID, plugin.Version, plugin.Scope, true
		}
	}
	return "", "", "", false
}

func claudePluginIsStripe(plugin claudeInstalledPlugin) bool {
	return strings.EqualFold(plugin.ID, TargetClaudePlugin)
}
