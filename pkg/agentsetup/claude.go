// Package agentsetup contains helpers for detecting and configuring AI coding
// agent integrations.
package agentsetup

import (
	"context"
	"fmt"
	"io"
	"strings"
)

const (
	ClientClaudeCode      = "claude-code"
	ClaudeBinaryName      = "claude"
	TargetClaudePlugin    = "stripe@claude-plugins-official"
	LocalClaudePlugin     = "stripe@stripe"
	ClaudeMarketplace     = "claude-plugins-official"
	ClaudePluginStatePath = ".claude/plugins/installed_plugins.json"
	ClaudeDisplayName     = "Claude Code"
)

func claudeClient() pluginClient {
	return pluginClient{
		id:           ClientClaudeCode,
		displayName:  ClaudeDisplayName,
		shortName:    "Claude",
		binaryName:   ClaudeBinaryName,
		targetPlugin: TargetClaudePlugin,
		localPlugin:  LocalClaudePlugin,
		pluginState:  ClaudePluginStatePath,
	}
}

// ClaudeProvider detects and configures the Stripe plugin for Claude Code.
type ClaudeProvider struct {
	Scanner    Scanner
	RunCommand RunCommandFunc
}

// NewClaudeProvider returns a Claude Code setup provider.
func NewClaudeProvider(scanner Scanner, runCommand RunCommandFunc) ClaudeProvider {
	if runCommand == nil {
		runCommand = RunCommand
	}
	return ClaudeProvider{
		Scanner:    scanner,
		RunCommand: runCommand,
	}
}

func (p ClaudeProvider) ID() string {
	return ClientClaudeCode
}

func (p ClaudeProvider) Detect() Status {
	return p.Scanner.ScanClaude()
}

func (p ClaudeProvider) Plan(status Status, force bool) Plan {
	name, args := ClaudeInstallCommand()
	command := append([]string{name}, args...)

	switch {
	case status.Status == StatusError:
		return Plan{Action: ActionNone}
	case !status.Detected:
		return Plan{Action: ActionNone}
	case status.Plugin.Installed && force:
		return Plan{Action: ActionReinstall, Command: command}
	case status.Plugin.Installed:
		return Plan{Action: ActionNone}
	default:
		return Plan{Action: ActionInstall, Command: command}
	}
}

// Apply installs the Stripe Claude Code plugin, refreshing the official plugin
// marketplace and retrying once if the first attempt fails.
func (p ClaudeProvider) Apply(ctx context.Context, out io.Writer, plan Plan) error {
	if plan.Action == ActionNone {
		return nil
	}
	if len(plan.Command) == 0 {
		return fmt.Errorf("missing command for %s action", plan.Action)
	}

	name, installArgs := plan.Command[0], plan.Command[1:]
	if err := p.RunCommand(ctx, name, installArgs...); err == nil {
		return nil
	}

	updateName, updateArgs := ClaudeMarketplaceUpdateCommand()
	updateCommand := append([]string{updateName}, updateArgs...)
	fmt.Fprintf(out, "Install failed. Updating Claude plugin marketplace and retrying: %s\n", strings.Join(updateCommand, " "))
	if updateErr := p.RunCommand(ctx, updateName, updateArgs...); updateErr != nil {
		return fmt.Errorf("running %q after install failed: %w", strings.Join(updateCommand, " "), updateErr)
	}
	if retryErr := p.RunCommand(ctx, name, installArgs...); retryErr != nil {
		return fmt.Errorf("running %q after marketplace update: %w", strings.Join(plan.Command, " "), retryErr)
	}
	return nil
}

// ScanClaude returns Claude Code installation and Stripe plugin status.
func (s Scanner) ScanClaude() Status {
	return s.scanPlugin(claudeClient())
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
