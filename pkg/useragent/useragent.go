// Package useragent builds User-Agent strings for Stripe API requests.
package useragent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/stripe/stripe-cli/pkg/version"
)

//
// Public functions
//

// GetEncodedStripeUserAgent returns the string to be used as the value for
// the `X-Stripe-Client-User-Agent` HTTP header.
func GetEncodedStripeUserAgent() string {
	return encodedStripeUserAgent
}

// GetEncodedUserAgent returns the string to be used as the value for
// the `User-Agent` HTTP header.
func GetEncodedUserAgent() string {
	return encodedUserAgent
}

// DetectInstallMethod detects how the CLI was installed.
// Returns one of: "npm_global", "npm_run", "npx", "homebrew", "scoop", "apt", or "unknown".
func DetectInstallMethod(
	getEnv func(string) string,
	getExecutable func() (string, error),
	statFile func(string) error,
) string {
	if method := getEnv("STRIPE_INSTALL_METHOD"); method != "" {
		return method
	}

	if exe, err := getExecutable(); err == nil {
		exeLower := strings.ToLower(filepath.ToSlash(exe))
		if strings.Contains(exeLower, "/cellar/") || strings.Contains(exeLower, "/homebrew/") {
			return "homebrew"
		}
		if strings.Contains(exeLower, "/scoop/apps/") {
			return "scoop"
		}
	}

	if err := statFile("/var/lib/dpkg/info/stripe.list"); err == nil {
		return "apt"
	}

	return "unknown"
}

// DetectInTmux detects whether the CLI was invoked from within a tmux session.
func DetectInTmux(getEnv func(string) string) bool {
	return getEnv("TMUX") != ""
}

// DetectInScreen detects whether the CLI was invoked from within a GNU Screen session.
func DetectInScreen(getEnv func(string) string) bool {
	return getEnv("STY") != ""
}

// DetectTerminalProgram detects the terminal program the CLI was invoked from, when available.
func DetectTerminalProgram(getEnv func(string) string) string {
	if terminal := getEnv("LC_TERMINAL"); terminal != "" {
		return terminal
	}
	if getEnv("WARP_CLIENT_VERSION") != "" {
		return "warp"
	}
	if getEnv("WT_SESSION") != "" {
		return "windows_terminal"
	}
	if getEnv("KITTY_WINDOW_ID") != "" {
		return "kitty"
	}
	if getEnv("ALACRITTY_WINDOW_ID") != "" || getEnv("ALACRITTY_LOG") != "" {
		return "alacritty"
	}
	if getEnv("WEZTERM_EXECUTABLE") != "" || getEnv("WEZTERM_PANE") != "" {
		return "wezterm"
	}
	if getEnv("GHOSTTY_RESOURCES_DIR") != "" {
		return "ghostty"
	}
	if program := getEnv("TERM_PROGRAM"); program != "" {
		return program
	}
	return ""
}

// DetectAIAgent detects if the CLI was invoked by a coding agent, based on well-known env vars.
// It accepts an environment getter function to allow testing without modifying the actual environment.
func DetectAIAgent(getEnv func(string) string) string {
	if getEnv("ANTIGRAVITY_CLI_ALIAS") != "" {
		return "antigravity"
	}
	if getEnv("CLAUDECODE") != "" {
		return "claude_code"
	}
	if getEnv("CLINE_ACTIVE") != "" {
		return "cline"
	}
	// Codex Desktop also sets the generic Codex env vars, so check its originator first.
	if getEnv("CODEX_INTERNAL_ORIGINATOR_OVERRIDE") == "Codex Desktop" {
		return "codex_desktop"
	}
	if getEnv("CODEX_SANDBOX") != "" || getEnv("CODEX_THREAD_ID") != "" || getEnv("CODEX_SANDBOX_NETWORK_DISABLED") != "" || getEnv("CODEX_CI") != "" {
		return "codex_cli"
	}
	if getEnv("CURSOR_AGENT") != "" {
		return "cursor"
	}
	if getEnv("GEMINI_CLI") != "" {
		return "gemini_cli"
	}
	if getEnv("OPENCODE") != "" {
		return "open_code"
	}
	if getEnv("OPENCLAW_SHELL") != "" {
		return "openclaw"
	}
	return ""
}

// DetectAIAgentRaw returns the agent's self-reported identifier from the emerging
// AI_AGENT/AGENT convention, or "" when unset. Unlike DetectAIAgent this is not a
// curated value: any vendor adopting the convention shows up here without a CLI
// release, at the cost of being unvalidated free text. Prefer DetectAIAgent for
// grouping and treat this as supplementary.
//
// AI_AGENT is used by Vercel's detect-agent and set by Claude Code (which
// deliberately does not overwrite another vendor's value). AGENT is used by Goose,
// Amp, and Bun, and is being added to Codex. AGENT is the more collision-prone of
// the two, so it is only consulted as a fallback.
func DetectAIAgentRaw(getEnv func(string) string) string {
	if agent := sanitizeEnvValue(getEnv("AI_AGENT")); agent != "" {
		return agent
	}
	return sanitizeEnvValue(getEnv("AGENT"))
}

// DetectAgentHost returns the application hosting the coding agent, when the agent
// reports one. This distinguishes a desktop or IDE-hosted agent from the same agent
// running in a terminal, which DetectAIAgent alone cannot do.
//
// Values are not mapped to a fixed set, since the set evolves outside this CLI. Known
// Claude Code values include "cli", "claude-desktop", "claude-vscode", "mcp",
// "sdk-ts", "sdk-py", "local-agent", "claude-code-github-action", and several
// "remote*" variants. Codex Desktop reports "codex-desktop".
func DetectAgentHost(getEnv func(string) string) string {
	if host := getEnv("CLAUDE_CODE_ENTRYPOINT"); host != "" {
		return normalizeAgentHost(host)
	}
	// Codex Desktop sets this alongside the generic Codex signals; it is currently the
	// only inherited value that separates Desktop from the Codex CLI.
	if host := getEnv("CODEX_INTERNAL_ORIGINATOR_OVERRIDE"); host != "" {
		return normalizeAgentHost(host)
	}
	return detectMacAppHost(getEnv)
}

// AgentHostKind maps a host returned by DetectAgentHost to a coarse category, or ""
// when the host is unset or not one we recognize.
//
// This is the curated counterpart to the free-form agent_host value, in the same way
// that DetectAIAgent is the curated counterpart to DetectAIAgentRaw. It exists so that
// a question like "how many invocations came from a desktop app" is one stable
// predicate rather than a list of per-vendor spellings that has to be extended every
// time a vendor ships a new host or renames an existing one.
func AgentHostKind(host string) string {
	// Remote hosts are matched first and by prefix: "remote-desktop" is a remote
	// session, not the desktop app, so any substring match on "desktop" would
	// silently conflate the two.
	if host == "remote" || strings.HasPrefix(host, "remote-") {
		return "remote"
	}
	return agentHostKinds[host]
}

// detectMacAppHost identifies the host from the macOS bundle identifier of the app
// that launched the process tree, for the case where a desktop app invokes the CLI
// without its coding agent in between -- an MCP server that shells out, for example --
// and so sets none of the agent environment variables.
//
// macOS sets __CFBundleIdentifier when an app bundle launches a process, and child
// processes inherit it, so it names the GUI app at the root of the session rather than
// the direct parent. Only bundle IDs we explicitly recognize are mapped; every other
// value, including terminal emulators, is ignored rather than reported. That keeps this
// a narrow positive check instead of a general "which app launched us" field, whose
// value would be unbounded and mostly terminal emulators.
func detectMacAppHost(getEnv func(string) string) string {
	// A terminal emulator between the app and the CLI means the session is a terminal
	// one, whatever app happens to sit at the root of the tree.
	if DetectTerminalProgram(getEnv) != "" {
		return ""
	}
	return macAppHosts[sanitizeEnvValue(getEnv("__CFBundleIdentifier"))]
}

//
// Private types
//

// stripeClientUserAgent contains information about the current runtime which
// is serialized and sent in the `X-Stripe-Client-User-Agent` as additional
// debugging information.
type stripeClientUserAgent struct {
	Name      string `json:"name"`
	OS        string `json:"os"`
	Publisher string `json:"publisher"`
	Uname     string `json:"uname"`
	Version   string `json:"version"`
}

//
// Private variables
//

var encodedStripeUserAgent string
var encodedUserAgent string

//
// Private constants
//

// maxEnvValueLength bounds free-form environment values before they are reported,
// so an unexpected or hostile value cannot bloat a request.
const maxEnvValueLength = 64

//
// Private variables
//

// agentHostKinds categorizes the hosts we know about. Hosts absent from this map are
// still reported verbatim as agent_host; only the coarse category is omitted, so an
// unrecognized host is a gap in grouping rather than lost data. "remote*" hosts are
// handled by prefix in AgentHostKind rather than enumerated here.
// macAppHosts maps the bundle identifiers of desktop apps that can invoke the CLI to
// the host they correspond to. Deliberately minimal: an unlisted bundle ID reports no
// host at all, so adding an entry is a decision rather than a default.
//
// com.openai.codex is what ChatGPT.app itself ships as, verified against an installed
// 26.730.61639 build -- OpenAI uses one bundle identifier for the desktop app, so this
// cannot separate ChatGPT from Codex, only identify the OpenAI desktop app.
var macAppHosts = map[string]string{
	"com.anthropic.claudefordesktop": "claude-desktop",
	"com.openai.codex":               "codex-desktop",
}

var agentHostKinds = map[string]string{
	"claude-code-github-action": "ci",
	"claude-desktop":            "desktop",
	"claude-in-slack":           "chat",
	"claude-in-teams":           "chat",
	"claude-vscode":             "ide",
	"cli":                       "terminal",
	"codex-desktop":             "desktop",
	"mcp":                       "mcp",
	"sdk-cli":                   "sdk",
	"sdk-py":                    "sdk",
	"sdk-ts":                    "sdk",
}

//
// Private functions
//

// normalizeAgentHost lowercases a reported host and collapses spaces and underscores
// to dashes, so that values from different vendors land in one comparable column.
// Claude Code reports "claude-desktop" while Codex Desktop reports "Codex Desktop",
// and Claude Code is itself inconsistent: it ships both "claude_in_slack" and
// "claude-in-slack", alongside dashed values like "claude-vscode".
func normalizeAgentHost(host string) string {
	lowered := strings.ToLower(sanitizeEnvValue(host))
	return strings.NewReplacer(" ", "-", "_", "-").Replace(lowered)
}

// sanitizeEnvValue trims an environment value, drops anything outside printable
// ASCII, and truncates the result to maxEnvValueLength.
func sanitizeEnvValue(value string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < ' ' || r > '~' {
			return -1
		}
		return r
	}, strings.TrimSpace(value))

	if len(cleaned) > maxEnvValueLength {
		cleaned = cleaned[:maxEnvValueLength]
	}

	return cleaned
}

func init() {
	initUserAgent()
}

func initUserAgent() {
	encodedUserAgent = "Stripe/v1 stripe-cli/" + version.Version
	if agent := DetectAIAgent(os.Getenv); agent != "" {
		encodedUserAgent += fmt.Sprintf(" AIAgent/%s", agent)
	}

	stripeUserAgent := &stripeClientUserAgent{
		Name:      "stripe-cli",
		Version:   version.Version,
		Publisher: "stripe",
		OS:        runtime.GOOS,
		Uname:     getUname(),
	}
	marshaled, err := json.Marshal(stripeUserAgent)
	// Encoding this struct should never be a problem, so we're okay to panic
	// in case it is for some reason.
	if err != nil {
		panic(err)
	}

	encodedStripeUserAgent = string(marshaled)
}
