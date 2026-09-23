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
	TargetClaudePlugin = "stripe@claude-plugins-official"
	ClaudeMarketplace  = "claude-plugins-official"
	ClaudeDisplayName  = "Claude Code"

	claudeListTimeout = 5 * time.Second
)

// ClaudeProvider detects and configures the Stripe plugin for Claude Code.
type ClaudeProvider struct {
	RunCommand RunCommandFunc
	RunOutput  RunOutputFunc
	config     ProviderConfig
}

// NewClaudeProvider returns a Claude Code setup provider.
func NewClaudeProvider(scanner Scanner, runCommand RunCommandFunc) ClaudeProvider {
	if runCommand == nil {
		runCommand = RunCommand
	}
	config := ProviderConfig{
		scanner:     scanner,
		client:      ClientClaudeCode,
		binaryName:  ClaudeBinaryName,
		displayName: ClaudeDisplayName,
	}
	return ClaudeProvider{
		RunCommand: runCommand,
		RunOutput:  runCommandOutput,
		config:     config,
	}
}

func (p ClaudeProvider) ID() string {
	return p.config.client
}

func (p ClaudeProvider) Detect() Status {
	status, detected := detectAgentExecutable(
		p.config,
		StatusMissing,
	)
	if !detected {
		return status
	}

	ctx, cancel := context.WithTimeout(context.Background(), claudeListTimeout)
	defer cancel()

	pluginID, version, scope, ok, supportsPlugins := p.stripePluginStatus(ctx)
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
	name, args := ClaudeInstallCommand()
	installOrReinstallcommand := append([]string{name}, args...)
	return determinePlan(status, force, installOrReinstallcommand, installOrReinstallcommand)
}

// Apply installs the Stripe Claude Code plugin. On failure it silently refreshes
// the official marketplace and retries once.
func (p ClaudeProvider) Apply(ctx context.Context, _ io.Writer, plan Plan) error {
	if plan.Action == ActionNone {
		return nil
	}
	if len(plan.Command) == 0 {
		return errorcategory.Errorf(errorcategory.Internal, "missing command for %s action", plan.Action)
	}

	name, installArgs := plan.Command[0], plan.Command[1:]
	if err := p.RunCommand(ctx, name, installArgs...); err == nil {
		return nil
	}

	updateName, updateArgs := ClaudeMarketplaceUpdateCommand()
	if updateErr := p.RunCommand(ctx, updateName, updateArgs...); updateErr != nil {
		return updateErr
	}
	return p.RunCommand(ctx, name, installArgs...)
}

// stripePluginStatus runs `claude plugin list --json` and reports whether the
// Stripe plugin is installed. When the command fails (e.g. old Claude version
// without plugin support), supportsPlugins is false.
func (p ClaudeProvider) stripePluginStatus(ctx context.Context) (id, version, scope string, installed bool, supportsPlugins bool) {
	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}
	out, err := runOutput(ctx, p.config.binaryName, "plugin", "list", "--json")
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

// ClaudeInstallCommand returns the command used to install the Stripe Claude
// Code plugin.
func ClaudeInstallCommand() (string, []string) {
	return ClaudeBinaryName, []string{"plugin", "install", TargetClaudePlugin}
}

// ClaudeMarketplaceUpdateCommand returns the command used to refresh Claude's
// official plugin marketplace metadata.
func ClaudeMarketplaceUpdateCommand() (string, []string) {
	return ClaudeBinaryName, []string{"plugin", "marketplace", "update", ClaudeMarketplace}
}
