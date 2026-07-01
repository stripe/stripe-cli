package agentsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const (
	ClientCodex       = "codex"
	CodexBinaryName   = "codex"
	CodexPluginName   = "stripe"
	CodexMarketplace  = "openai-curated"
	TargetCodexPlugin = "stripe@openai-curated"
	CodexDisplayName  = "Codex CLI"

	codexListTimeout = 10 * time.Second
)

// RunOutputFunc runs a command and returns its standard output. It exists so
// Codex detection (which shells out to `codex plugin list --json`) is testable.
type RunOutputFunc func(context.Context, string, ...string) ([]byte, error)

// CodexProvider detects and installs the Stripe plugin for Codex CLI.
//
// Codex has a real plugin CLI, so detection runs `codex plugin list --json` and
// installation runs `codex plugin add stripe@openai-curated`.
type CodexProvider struct {
	Scanner    Scanner
	RunCommand RunCommandFunc
	RunOutput  RunOutputFunc
}

// NewCodexProvider returns a Codex CLI setup provider.
func NewCodexProvider(scanner Scanner, runCommand RunCommandFunc) Provider {
	if runCommand == nil {
		runCommand = RunCommand
	}
	return CodexProvider{
		Scanner:    scanner,
		RunCommand: runCommand,
		RunOutput:  runCommandOutput,
	}
}

func (p CodexProvider) ID() string { return ClientCodex }

func (p CodexProvider) Detect() Status {
	scanner := p.Scanner.withDefaults()

	status := Status{
		Client:      ClientCodex,
		DisplayName: CodexDisplayName,
		Status:      StatusNotDetected,
	}

	binPath, err := scanner.LookPath(CodexBinaryName)
	if err != nil {
		return status
	}
	status.Detected = true
	status.ExecutablePath = binPath
	status.Status = StatusMissing

	ctx, cancel := context.WithTimeout(context.Background(), codexListTimeout)
	defer cancel()

	if version, ok := p.stripePluginVersion(ctx); ok {
		status.Plugin.Installed = true
		status.Plugin.ID = TargetCodexPlugin
		status.Plugin.Version = version
		status.Plugin.Scope = "user"
		status.Status = StatusInstalled
	}

	return status
}

// stripePluginVersion runs `codex plugin list --json` and reports the installed
// Stripe plugin version, if any. A command or parse failure is treated as "not
// installed" so a single flaky call does not fail the whole scan.
func (p CodexProvider) stripePluginVersion(ctx context.Context) (string, bool) {
	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}
	out, err := runOutput(ctx, CodexBinaryName, "plugin", "list", "--json")
	if err != nil {
		return "", false
	}
	return findCodexStripePlugin(out)
}

func (p CodexProvider) Plan(status Status, force bool) Plan {
	command := []string{CodexBinaryName, "plugin", "add", TargetCodexPlugin}

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

func (p CodexProvider) Apply(ctx context.Context, _ io.Writer, plan Plan) error {
	if plan.Action == ActionNone {
		return nil
	}
	if len(plan.Command) == 0 {
		return fmt.Errorf("missing command for %s action", plan.Action)
	}
	runCommand := p.RunCommand
	if runCommand == nil {
		runCommand = RunCommand
	}
	if err := runCommand(ctx, plan.Command[0], plan.Command[1:]...); err != nil {
		return err
	}

	// `codex plugin add` exits 0 even when it fails (e.g. the marketplace is not
	// configured), so the exit code cannot be trusted. Confirm the plugin is
	// actually installed before reporting success.
	if _, ok := p.stripePluginVersion(ctx); !ok {
		return fmt.Errorf("codex reported success but %s is not installed; run `%s` to see the underlying error",
			TargetCodexPlugin, strings.Join(plan.Command, " "))
	}
	return nil
}

// codexPluginList is the shape of `codex plugin list --json` output.
type codexPluginList struct {
	Installed []codexInstalledPlugin `json:"installed"`
}

// codexInstalledPlugin is an entry in `codex plugin list --json`'s "installed"
// array. Field names match the real Codex CLI output (verified against
// codex-cli 0.142.0), e.g. {"pluginId":"stripe@openai-curated","name":"stripe",
// "marketplaceName":"openai-curated","version":"..."}.
type codexInstalledPlugin struct {
	PluginID    string `json:"pluginId"`
	Name        string `json:"name"`
	Marketplace string `json:"marketplaceName"`
	Version     string `json:"version"`
}

// findCodexStripePlugin reports whether the Stripe plugin appears in the
// installed list and returns its version when available.
func findCodexStripePlugin(listJSON []byte) (string, bool) {
	var list codexPluginList
	if err := json.Unmarshal(listJSON, &list); err != nil {
		return "", false
	}

	for _, plugin := range list.Installed {
		if codexPluginIsStripe(plugin) {
			return plugin.Version, true
		}
	}
	return "", false
}

func codexPluginIsStripe(plugin codexInstalledPlugin) bool {
	if strings.EqualFold(plugin.PluginID, TargetCodexPlugin) {
		return true
	}
	return strings.EqualFold(plugin.Name, CodexPluginName) &&
		strings.EqualFold(plugin.Marketplace, CodexMarketplace)
}

func runCommandOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}
