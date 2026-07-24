package coopcmd

import (
	"fmt"
	"path/filepath"
	"strings"
)

// harness describes a terminal AI coding agent that co-op can launch in the
// agent pane.
//
// Co-op drives the agent through a single generated launcher script of the form
// `exec <path> [autoApproveFlag] [promptFlag] "$prompt"`, so every harness here
// must accept an initial prompt that leaves an *interactive* session running.
// Harnesses whose prompt flag forces a headless one-shot run (the agent answers
// and exits) are not usable: co-op's review loop depends on the agent process
// staying alive across `stripe coop agent await-review`.
type harness struct {
	// id is the stable identifier accepted by --agent.
	id string
	// displayName labels the harness in the picker and permission prompt.
	displayName string
	// binary is the executable looked up on PATH.
	binary string
	// args are fixed arguments placed before any flags, for harnesses whose
	// seeded-interactive mode lives behind a subcommand.
	args []string
	// promptFlag carries the initial prompt for harnesses that do not accept it
	// as a positional argument. Empty means the prompt is positional.
	//
	// Note that the flag spelling matters: on several harnesses a bare
	// -p/--prompt means "run headless and exit" while a different flag seeds an
	// interactive session. Only the interactive spelling belongs here.
	promptFlag string
	// autoApproveFlag skips all permission prompts.
	autoApproveFlag string
	// autoApproveEnv is a KEY=value pair exported before exec, for harnesses
	// that control approvals through the environment instead of a flag.
	autoApproveEnv string
	// noPermissionGate marks a harness that never prompts before acting. Co-op
	// warns instead of simply omitting the permission-mode question, so the
	// missing prompt is not read as "the safe default applied".
	noPermissionGate bool
}

// permissionNotice returns a warning to print before launching, or "" when the
// harness gates tool use normally.
//
// This covers only harnesses known to have no gate. A custom --agent binary
// also has no autoApproveFlag, but that reflects co-op not knowing its
// permission model rather than knowing it has none.
func (h harness) permissionNotice() string {
	if !h.noPermissionGate {
		return ""
	}
	return fmt.Sprintf("%s has no permission prompts: it reads, writes, and runs commands "+
		"without asking. Co-op cannot gate it, so there is no permission mode to choose.",
		h.displayName)
}

// offersAutoApprove reports whether co-op can skip this harness's permission
// prompts. When false, co-op does not ask the developer to choose a permission
// mode — either the harness has no such control, or it has no prompts at all.
func (h harness) offersAutoApprove() bool {
	return h.autoApproveFlag != "" || h.autoApproveEnv != ""
}

// supportedHarnesses is the canonical registry, in picker order. Claude Code and
// Codex lead because they were the original two supported agents; the rest
// follow alphabetically.
var supportedHarnesses = []harness{
	{
		id:              "claude",
		displayName:     "Claude Code",
		binary:          "claude",
		autoApproveFlag: "--dangerously-skip-permissions",
	},
	{
		id:              "codex",
		displayName:     "Codex",
		binary:          "codex",
		autoApproveFlag: "--dangerously-bypass-approvals-and-sandbox",
	},
	{
		id:              "cursor",
		displayName:     "Cursor CLI",
		binary:          "cursor-agent",
		autoApproveFlag: "--force",
	},
	{
		id:          "gemini",
		displayName: "Gemini CLI",
		binary:      "gemini",
		// -i/--prompt-interactive seeds the session and stays interactive;
		// -p/--prompt would run headless and exit.
		promptFlag:      "-i",
		autoApproveFlag: "--approval-mode=yolo",
	},
	{
		id:          "goose",
		displayName: "Goose",
		binary:      "goose",
		// `goose session` cannot be seeded; only `goose run` accepts input, and
		// -s/--interactive is what keeps it interactive afterward.
		args:       []string{"run", "-s"},
		promptFlag: "-t",
		// Goose has no auto-approve flag; GOOSE_MODE selects the approval
		// policy. Co-op sets it only when the developer opts into auto-approve,
		// so declining leaves any configured mode untouched.
		autoApproveEnv: "GOOSE_MODE=auto",
	},
	{
		id:          "opencode",
		displayName: "opencode",
		binary:      "opencode",
		// opencode inverts the usual convention: --prompt seeds the TUI, while
		// `opencode run <message>` is the headless one-shot form.
		promptFlag:      "--prompt",
		autoApproveFlag: "--auto",
	},
	{
		id:          "pi",
		displayName: "Pi",
		binary:      "pi",
		// Pi ships no permission system and no sandbox: it runs with the
		// privileges of the launching process and never prompts, so there is
		// nothing for co-op's auto-approve choice to skip.
		noPermissionGate: true,
	},
}

// customHarness describes an agent supplied via --agent that is not in the
// registry. Co-op passes the prompt positionally and adds no flags, since it
// cannot know the binary's permission model.
func customHarness(name string) harness {
	return harness{id: name, displayName: name, binary: name}
}

// harnessByID returns the registry entry with the given id.
func harnessByID(id string) (harness, bool) {
	for _, h := range supportedHarnesses {
		if h.id == id {
			return h, true
		}
	}
	return harness{}, false
}

// harnessFor resolves a --agent value to a registry entry. It matches the value
// against harness ids and binary names, then falls back to the resolved
// executable's base name so that an absolute path (or a shell alias resolving to
// one) still picks up the right flags.
//
// Matching is deliberately exact rather than substring-based: a path such as
// /home/claude-vm/bin/myagent must not be mistaken for Claude Code.
func harnessFor(name, path string) (harness, bool) {
	candidates := []string{name, executableName(name), executableName(path)}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		for _, h := range supportedHarnesses {
			if h.id == candidate || h.binary == candidate {
				return h, true
			}
		}
	}
	return harness{}, false
}

// windowsExecExtensions are stripped when deriving a binary name, so a Windows
// lookup resolving to cursor-agent.exe still matches the registry.
var windowsExecExtensions = []string{".exe", ".bat", ".cmd", ".com"}

// executableName returns the base name of path with any Windows executable
// extension removed.
//
// Both separators are handled explicitly rather than via filepath.Base, which
// only treats "\" as a separator when GOOS is windows — exec.LookPath returns
// backslash paths there, and this must resolve identically wherever it runs.
// Only executable extensions are stripped: a wrapper script named claude.sh is
// a different program from Claude Code and must not inherit its flags.
func executableName(path string) string {
	base := path
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	for _, ext := range windowsExecExtensions {
		if strings.EqualFold(filepath.Ext(base), ext) {
			return base[:len(base)-len(ext)]
		}
	}
	return base
}

// supportedHarnessList renders the registry for error messages.
func supportedHarnessList() string {
	ids := make([]string, 0, len(supportedHarnesses))
	for _, h := range supportedHarnesses {
		ids = append(ids, h.id)
	}
	return strings.Join(ids, ", ")
}

// noAgentFoundError explains that no supported harness is installed.
func noAgentFoundError() error {
	return fmt.Errorf(`no AI agent found in PATH.
  Co-op supports: %s
  Install Claude Code: https://docs.anthropic.com/en/docs/claude-code
  Or specify a custom agent: --agent=<command>`, supportedHarnessList())
}
