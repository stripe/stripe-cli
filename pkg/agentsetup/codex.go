package agentsetup

import (
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const (
	ClientCodex         = "codex"
	CodexBinaryName     = "codex"
	CodexPluginName     = "stripe"
	CodexMarketplace    = "openai-curated"
	CodexAPIMarketplace = "openai-api-curated"
	TargetCodexPlugin   = "stripe@openai-curated"
	CodexDisplayName    = "Codex CLI"

	codexListTimeout = 5 * time.Second
)

// RunOutputFunc runs a command and returns its standard output. It exists so
// Codex detection (which shells out to `codex plugin list --json`) is testable.
type RunOutputFunc func(context.Context, string, ...string) ([]byte, error)

// CodexProvider detects and installs the Stripe plugin for Codex CLI.
//
// Detection runs `codex plugin list --json`. Installation uses the curated
// marketplace available to the user's Codex authentication method.
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

	pluginID, version, ok, supportsPlugins := p.stripePluginStatus(ctx)
	if !supportsPlugins {
		status.Error = "upgrade Codex to enable plugin support"
		return status
	}
	if ok {
		status.Plugin.Installed = true
		status.Plugin.ID = pluginID
		status.Plugin.Version = version
		status.Plugin.Scope = "user"
		status.Status = StatusInstalled
	}

	return status
}

// stripePluginStatus runs `codex plugin list --json` and reports whether (1)
// the command is supported (supportsPlugins), and if so (2) whether the Stripe
// plugin is installed, its ID, and its version. When the command fails (e.g. old Codex
// version without plugin support), supportsPlugins is false.
func (p CodexProvider) stripePluginStatus(ctx context.Context) (id, version string, installed, supportsPlugins bool) {
	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}
	out, err := runOutput(ctx, CodexBinaryName, "plugin", "list", "--json")
	if err != nil {
		return "", "", false, false
	}
	id, v, ok := findCodexStripePlugin(out)
	return id, v, ok, true
}

func (p CodexProvider) Plan(status Status, force bool) Plan {
	if status.Status == StatusError || !status.Detected || (status.Plugin.Installed && !force) {
		return Plan{Action: ActionNone}
	}

	pluginID := status.Plugin.ID
	if pluginID == "" {
		pluginID = p.targetPlugin()
	}
	command := []string{CodexBinaryName, "plugin", "add", pluginID}
	if status.Plugin.Installed {
		return Plan{Action: ActionReinstall, Command: command}
	}
	return Plan{Action: ActionInstall, Command: command}
}

// targetPlugin selects an available official marketplace. Older Codex versions
// may not support marketplace discovery, so retain the original default when
// discovery fails or returns no recognized marketplace.
func (p CodexProvider) targetPlugin() string {
	ctx, cancel := context.WithTimeout(context.Background(), codexListTimeout)
	defer cancel()

	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}
	out, err := runOutput(ctx, CodexBinaryName, "plugin", "marketplace", "list", "--json")
	if err != nil {
		return TargetCodexPlugin
	}
	var list codexMarketplaceList
	if err := json.Unmarshal(out, &list); err != nil {
		return TargetCodexPlugin
	}

	for _, name := range []string{CodexMarketplace, CodexAPIMarketplace} {
		for _, marketplace := range list.Marketplaces {
			if strings.EqualFold(marketplace.Name, name) {
				return CodexPluginName + "@" + name
			}
		}
	}
	return TargetCodexPlugin
}

func (p CodexProvider) Apply(ctx context.Context, _ io.Writer, plan Plan) error {
	if plan.Action == ActionNone {
		return nil
	}
	if len(plan.Command) == 0 {
		return errorcategory.Errorf(errorcategory.Internal, "missing command for %s action", plan.Action)
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
	if _, _, installed, _ := p.stripePluginStatus(ctx); !installed {
		return errorcategory.Errorf(errorcategory.Internal, "codex reported success but %s is not installed; run `%s` to see the underlying error",
			plan.Command[len(plan.Command)-1], strings.Join(plan.Command, " "))
	}
	return nil
}

// codexPluginList is the shape of `codex plugin list --json` output.
type codexPluginList struct {
	Installed []codexInstalledPlugin `json:"installed"`
}

// codexMarketplaceList is the shape of `codex plugin marketplace list --json`.
type codexMarketplaceList struct {
	Marketplaces []struct {
		Name string `json:"name"`
	} `json:"marketplaces"`
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
// installed list and returns its ID and version when available.
func findCodexStripePlugin(listJSON []byte) (id, version string, found bool) {
	var list codexPluginList
	if err := json.Unmarshal(listJSON, &list); err != nil {
		return "", "", false
	}

	for _, plugin := range list.Installed {
		if id := codexStripePluginID(plugin); id != "" {
			return id, plugin.Version, true
		}
	}
	return "", "", false
}

func codexStripePluginID(plugin codexInstalledPlugin) string {
	for _, marketplace := range []string{CodexMarketplace, CodexAPIMarketplace} {
		id := CodexPluginName + "@" + marketplace
		if strings.EqualFold(plugin.PluginID, id) ||
			(strings.EqualFold(plugin.Name, CodexPluginName) && strings.EqualFold(plugin.Marketplace, marketplace)) {
			return id
		}
	}
	return ""
}

func runCommandOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}
