package coopcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"

	"github.com/stripe/stripe-cli/pkg/coop/skill"
)

// harnessAdapter describes how one coding harness is detected, taught the
// bundled stripe-coop skill, and launched. Everything harness-specific lives
// here; the skill content itself stays portable.
//
// Verified against official harness documentation, July 2026.
type harnessAdapter struct {
	// id is the stable identifier accepted by --agent.
	id          string
	displayName string
	// executables are the PATH candidates that select this adapter.
	executables []string
	// skillDir returns the harness's user-level skill directory for the
	// bundled stripe-coop skill, relative to the user's home directory.
	skillDir func(home string) string
	// activationLine is the first line of every launch prompt. It uses the
	// harness's documented explicit skill activation syntax.
	activationLine string
	// promptClause places the launch prompt on the command line. It must
	// reference the shell variable "$prompt" exactly once, and may include a
	// leading subcommand (e.g. `run -t "$prompt"` for Goose).
	promptClause string
	// approvalArgs returns the flags implementing the developer's permission
	// choice, and extraArgs returns unconditional flags.
	approvalArgs func(autoApprove bool) string
	extraArgs    string
	// envPrefix returns a `KEY=value ` script prefix for harnesses that are
	// configured through the environment rather than flags.
	envPrefix func(autoApprove bool) string
	// bypassLabel names the harness's unrestricted mode in the permission
	// prompt.
	bypassLabel string
	// interactiveLaunch reports whether launching with an initial prompt
	// leaves the developer in an interactive conversation. Harnesses without
	// it cannot run the conversational no-blueprint discovery flow.
	interactiveLaunch bool
	// notes describes capability limitations surfaced to the developer.
	notes string
}

// agentsSkillsDir is the cross-harness skills directory from the Agent Skills
// standard, read by Codex, OpenCode, Roo Code, OpenHands, and Goose.
func agentsSkillsDir(home string) string {
	return filepath.Join(home, ".agents", "skills")
}

var harnessRegistry = []*harnessAdapter{
	{
		id:                "claude",
		displayName:       "Claude Code",
		executables:       []string{"claude"},
		skillDir:          func(home string) string { return filepath.Join(home, ".claude", "skills") },
		activationLine:    "Use the /stripe-coop skill.",
		promptClause:      `"$prompt"`,
		extraArgs:         "--agents " + shellQuote(claudeCoopAgents),
		approvalArgs:      flagWhenBypassed("--dangerously-skip-permissions"),
		bypassLabel:       "Bypass permissions — skip safety checks (isolated environments only)",
		interactiveLaunch: true,
	},
	{
		id:                "codex",
		displayName:       "Codex",
		executables:       []string{"codex"},
		skillDir:          agentsSkillsDir,
		activationLine:    "Use the $stripe-coop skill.",
		promptClause:      `"$prompt"`,
		approvalArgs:      flagWhenBypassed("--dangerously-bypass-approvals-and-sandbox"),
		bypassLabel:       "Bypass approvals and sandbox — skip safety checks (isolated environments only)",
		interactiveLaunch: true,
	},
	{
		id:                "opencode",
		displayName:       "OpenCode",
		executables:       []string{"opencode"},
		skillDir:          agentsSkillsDir,
		activationLine:    "Use the stripe-coop skill.",
		promptClause:      `--prompt "$prompt"`,
		approvalArgs:      flagWhenBypassed("--auto"),
		bypassLabel:       "Auto-approve permissions — skip approval prompts (isolated environments only)",
		interactiveLaunch: true,
	},
	{
		id:                "cline",
		displayName:       "Cline",
		executables:       []string{"cline"},
		skillDir:          func(home string) string { return filepath.Join(home, ".cline", "skills") },
		activationLine:    "Use the /stripe-coop skill.",
		promptClause:      `"$prompt"`,
		approvalArgs:      flagWhenBypassed("--yolo"),
		bypassLabel:       "Auto-approve all actions (--yolo) — skip safety checks (isolated environments only)",
		interactiveLaunch: false,
		notes:             "Cline runs the launch prompt as a one-shot task, so the conversational no-blueprint discovery flow is unavailable.",
	},
	{
		id:             "roo",
		displayName:    "Roo Code",
		executables:    []string{"roo"},
		skillDir:       agentsSkillsDir,
		activationLine: "Use the stripe-coop skill.",
		promptClause:   `"$prompt"`,
		// Roo's CLI auto-approves by default; normal mode opts back into
		// manual approval.
		approvalArgs: func(autoApprove bool) string {
			if autoApprove {
				return ""
			}
			return "--require-approval"
		},
		bypassLabel:       "Auto-approve all actions (Roo default) — skip safety checks (isolated environments only)",
		interactiveLaunch: true,
		notes:             "Roo Code was discontinued in May 2026; its skill support is frozen at the final release.",
	},
	{
		id:                "openhands",
		displayName:       "OpenHands",
		executables:       []string{"openhands"},
		skillDir:          agentsSkillsDir,
		activationLine:    "Use the stripe-coop skill.",
		promptClause:      `-t "$prompt"`,
		approvalArgs:      flagWhenBypassed("--always-approve"),
		bypassLabel:       "Always approve actions — skip confirmation prompts (isolated environments only)",
		interactiveLaunch: true,
	},
	{
		id:             "goose",
		displayName:    "Goose",
		executables:    []string{"goose"},
		skillDir:       agentsSkillsDir,
		activationLine: "Use the stripe-coop skill.",
		promptClause:   `run -t "$prompt"`,
		// Goose's permission model is the GOOSE_MODE environment variable
		// (its default is autonomous).
		envPrefix: func(autoApprove bool) string {
			if autoApprove {
				return "GOOSE_MODE=auto "
			}
			return "GOOSE_MODE=approve "
		},
		bypassLabel:       "Autonomous mode (GOOSE_MODE=auto) — skip approval prompts (isolated environments only)",
		interactiveLaunch: false,
		notes:             "goose run executes the launch prompt without an interactive conversation, so the no-blueprint discovery flow is unavailable.",
	},
}

func flagWhenBypassed(flag string) func(bool) string {
	return func(autoApprove bool) string {
		if autoApprove {
			return flag
		}
		return ""
	}
}

// genericAdapter wraps an unrecognized --agent command. It launches with a
// positional prompt and receives no skill install, so its prompt falls back to
// the full self-contained lifecycle instructions.
func genericAdapter(command string) *harnessAdapter {
	return &harnessAdapter{
		id:                command,
		displayName:       command,
		executables:       []string{command},
		activationLine:    "",
		promptClause:      `"$prompt"`,
		approvalArgs:      func(bool) string { return "" },
		interactiveLaunch: true,
		notes:             "Unknown agent: launched with a positional prompt and without the managed stripe-coop skill.",
	}
}

func harnessByID(id string) *harnessAdapter {
	for _, adapter := range harnessRegistry {
		if adapter.id == id {
			return adapter
		}
		for _, executable := range adapter.executables {
			if executable == id {
				return adapter
			}
		}
	}
	return nil
}

type agentInfo struct {
	adapter *harnessAdapter
	path    string
}

var (
	lookPath    = exec.LookPath
	userHomeDir = os.UserHomeDir
)

func (rc *coopRunCmd) detectAgent() (*agentInfo, error) {
	if rc.agent != "" {
		path, err := lookPath(rc.agent)
		if err != nil {
			return nil, fmt.Errorf("agent %q not found in PATH", rc.agent)
		}
		adapter := harnessByID(rc.agent)
		if adapter == nil {
			adapter = harnessByID(filepath.Base(rc.agent))
		}
		if adapter == nil {
			adapter = genericAdapter(rc.agent)
		}
		return &agentInfo{adapter: adapter, path: path}, nil
	}

	var found []*agentInfo
	for _, adapter := range harnessRegistry {
		if path, err := lookPathAny(adapter.executables); err == nil {
			found = append(found, &agentInfo{adapter: adapter, path: path})
		}
	}

	switch len(found) {
	case 0:
		return nil, fmt.Errorf("no AI agent found in PATH.\n  Install Claude Code: https://docs.anthropic.com/en/docs/claude-code\n  Or specify a custom agent: --agent=<command>")
	case 1:
		return found[0], nil
	}

	options := make([]huh.Option[string], 0, len(found))
	for _, agent := range found {
		options = append(options, huh.NewOption(agent.adapter.displayName, agent.adapter.id))
	}
	var choice string
	if err := selectString("Multiple agents detected. Which would you like to use?", options, &choice); err != nil {
		return nil, err
	}
	for _, agent := range found {
		if agent.adapter.id == choice {
			return agent, nil
		}
	}
	return found[0], nil
}

func lookPathAny(candidates []string) (string, error) {
	var err error
	for _, candidate := range candidates {
		var path string
		if path, err = lookPath(candidate); err == nil {
			return path, nil
		}
	}
	if err == nil {
		err = exec.ErrNotFound
	}
	return "", err
}

func (rc *coopRunCmd) promptAutoApprove(agent *agentInfo) (bool, error) {
	if agent.adapter.bypassLabel == "" {
		return false, nil
	}
	var choice string
	err := selectString(fmt.Sprintf("Permission mode for %s:", agent.adapter.displayName),
		[]huh.Option[string]{
			huh.NewOption("Normal — agent asks before running commands", "normal"),
			huh.NewOption(agent.adapter.bypassLabel, "bypass"),
		},
		&choice,
	)
	if err != nil {
		return false, err
	}
	return choice == "bypass", nil
}

// buildAgentCmd writes a self-deleting launcher script that reads the prompt
// file and exec's the harness with its adapter-specific arguments.
func (rc *coopRunCmd) buildAgentCmd(agent *agentInfo, promptPath string, autoApprove bool) (string, error) {
	launcherPath := promptPath + ".sh"
	adapter := agent.adapter

	args := make([]string, 0, 3)
	if adapter.extraArgs != "" {
		args = append(args, adapter.extraArgs)
	}
	if adapter.approvalArgs != nil {
		if approval := adapter.approvalArgs(autoApprove); approval != "" {
			args = append(args, approval)
		}
	}
	args = append(args, adapter.promptClause)

	envPrefix := ""
	if adapter.envPrefix != nil {
		envPrefix = adapter.envPrefix(autoApprove)
	}

	script := fmt.Sprintf("#!/bin/bash\nprompt=$(cat %s)\nrm -f %s %s\nexec %s%s %s\n",
		shellQuote(promptPath), shellQuote(promptPath), shellQuote(launcherPath),
		envPrefix, shellQuote(agent.path), strings.Join(args, " "))

	if err := os.WriteFile(launcherPath, []byte(script), 0700); err != nil {
		return "", fmt.Errorf("creating agent launcher: %w", err)
	}
	return launcherPath, nil
}

// harnessSkillTarget returns the full stripe-coop install path for a harness,
// or "" when the harness has no managed skill location.
func harnessSkillTarget(adapter *harnessAdapter) (string, error) {
	if adapter.skillDir == nil {
		return "", nil
	}
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(adapter.skillDir(home), skill.Name), nil
}

// ensureHarnessCoopSkill installs or refreshes the bundled stripe-coop skill
// for the selected harness. It reports whether launch prompts may rely on the
// skill being present.
func ensureHarnessCoopSkill(adapter *harnessAdapter) (bool, error) {
	target, err := harnessSkillTarget(adapter)
	if err != nil || target == "" {
		return false, err
	}
	if _, err := skill.Install(target); err != nil {
		return false, err
	}
	return true, nil
}
