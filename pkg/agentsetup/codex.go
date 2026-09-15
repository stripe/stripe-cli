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
	ClientCodex      = "codex"
	CodexBinaryName  = "codex"
	CodexPluginName  = "stripe"
	CodexMarketplace = "openai-curated"
	CodexDisplayName = "Codex CLI"

	codexListTimeout = 5 * time.Second
)

// The first ID is the default when marketplace discovery is unavailable.
var codexPluginIDs = [...]string{
	"stripe@openai-curated",
	"stripe@openai-api-curated",
}

// RunOutputFunc runs a command and returns its standard output. It exists so
// Codex detection (which shells out to `codex plugin list --json`) is testable.
type RunOutputFunc func(context.Context, string, ...string) ([]byte, error)

// CodexProvider detects and installs the Stripe plugin for Codex CLI.
//
// Codex has a real plugin CLI, so detection runs `codex plugin list --json` and
// installation uses the available official marketplace.
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*codexListTimeout)
	defer cancel()

	pluginID, version, ok, supportsPlugins := p.stripePluginStatus(ctx)
	if !supportsPlugins {
		status.Error = "upgrade Codex to enable plugin support"
		return status
	}
	status.Plugin.ID = pluginID
	if ok {
		status.Plugin.Installed = true
		status.Plugin.Version = version
		status.Plugin.Scope = "user"
		status.Status = StatusInstalled
	}

	return status
}

// stripePluginStatus runs `codex plugin list --json` and reports whether (1)
// the command is supported (supportsPlugins), and if so (2) whether the Stripe
// plugin is installed and its ID and version. If missing, its ID selects the
// marketplace to install from. When the command fails (e.g. old Codex without plugin support),
// supportsPlugins is false.
func (p CodexProvider) stripePluginStatus(ctx context.Context) (id, version string, installed, supportsPlugins bool) {
	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}
	out, err := runOutput(ctx, CodexBinaryName, "plugin", "list", "--json")
	if err != nil {
		return "", "", false, false
	}
	pluginID, version, ok := findCodexStripePlugin(out)
	if !ok {
		// API-key logins use a different marketplace. Keep the original default
		// if marketplace discovery is unavailable in this Codex version.
		pluginID = codexPluginIDs[0]
		out, err = runOutput(ctx, CodexBinaryName, "plugin", "marketplace", "list", "--json")
		var list struct {
			Marketplaces []struct{ Name string } `json:"marketplaces"`
		}
		if err == nil && json.Unmarshal(out, &list) == nil {
			for _, id := range codexPluginIDs {
				for _, marketplace := range list.Marketplaces {
					if CodexPluginName+"@"+marketplace.Name == id {
						pluginID = id
					}
				}
			}
		}
	}
	return pluginID, version, ok, true
}

func (p CodexProvider) Plan(status Status, force bool) Plan {
	command := []string{CodexBinaryName, "plugin", "add", codexPluginIDs[0]}
	if status.Plugin.ID != "" {
		command[3] = status.Plugin.ID
	}

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
		for _, id := range codexPluginIDs {
			if strings.EqualFold(plugin.PluginID, id) || strings.EqualFold(plugin.Name+"@"+plugin.Marketplace, id) {
				return id, plugin.Version, true
			}
		}
	}
	return "", "", false
}

func runCommandOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}
