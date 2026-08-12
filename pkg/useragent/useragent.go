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

// detectAIAgentRaw returns the agent's self-reported identifier from the emerging
// AI_AGENT/AGENT convention, or "" when unset. It is deliberately unexported and never
// reported as-is: it is unvalidated free text from the environment, and only the
// version parsed out of it by DetectAgentVersion leaves the machine.
//
// AI_AGENT is used by Vercel's detect-agent and set by Claude Code (which
// deliberately does not overwrite another vendor's value). AGENT is used by Goose,
// Amp, and Bun, and is being added to Codex. AGENT is the more collision-prone of
// the two, so it is only consulted as a fallback.
func detectAIAgentRaw(getEnv func(string) string) string {
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
// that DetectAIAgent is curated rather than free text. It exists so that
// a question like "how many invocations came from a desktop app" is one stable
// predicate rather than a list of per-vendor spellings that has to be extended every
// time a vendor ships a new host or renames an existing one.
func AgentHostKind(host string) string {
	if host == "" {
		return ""
	}
	// Remote hosts are matched first and by prefix: "remote-desktop" is a remote
	// session, not the desktop app, so any substring match on "desktop" would
	// silently conflate the two.
	if host == "remote" || strings.HasPrefix(host, "remote-") {
		return "remote"
	}
	if kind, ok := agentHostKinds[host]; ok {
		return kind
	}
	return agentHostKindOther
}

// DetectAgentHostKind reports the category of the host that invoked the CLI. This is
// the single host signal reported in telemetry: the underlying host string is a
// per-vendor spelling that callers would have to enumerate, whereas the category is a
// bounded set, and the vendor is already carried by DetectAIAgent.
func DetectAgentHostKind(getEnv func(string) string) string {
	return AgentHostKind(DetectAgentHost(getEnv))
}

// DetectAgentVersion returns the agent's own version, parsed out of the identifier it
// reports through the AI_AGENT convention, or "" when absent or unparseable.
//
// Claude Code encodes it as "claude-code_2-1-222_agent" (dots written as dashes) and
// has also used a "claude-code/2.1.222" form. No other agent reports a version this
// way today, so in practice this is populated for Claude Code and empty elsewhere.
// The format is undocumented, so this validates what it extracts and reports nothing
// rather than guessing.
func DetectAgentVersion(getEnv func(string) string) string {
	identifier := detectAIAgentRaw(getEnv)
	if identifier == "" {
		return ""
	}

	// "claude-code/2.1.222" -- everything after the last slash.
	if idx := strings.LastIndex(identifier, "/"); idx >= 0 {
		return validVersion(strings.TrimSpace(identifier[idx+1:]))
	}

	// "claude-code_2-1-222_agent" -- the first underscore-delimited segment that reads
	// as a version once dashes are treated as dots.
	for _, segment := range strings.Split(identifier, "_") {
		if version := validVersion(strings.ReplaceAll(segment, "-", ".")); version != "" {
			return version
		}
	}

	return ""
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

// agentHostKindOther is reported when a host was detected but is not one we
// categorize. It keeps the field bounded while still making a new or renamed host
// visible, which reporting "" would not.
const agentHostKindOther = "other"

//
// Private variables
//

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

// agentHostKinds categorizes the hosts we know about. "remote*" hosts are handled by
// prefix in AgentHostKind rather than enumerated here.
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

// validVersion returns the candidate if it is a dot-separated numeric version, and ""
// otherwise. Deliberately strict: the identifier formats this parses are undocumented,
// and a wrong version is worse than a missing one.
func validVersion(candidate string) string {
	if candidate == "" || strings.HasPrefix(candidate, ".") || strings.HasSuffix(candidate, ".") {
		return ""
	}

	digits := false
	for _, r := range candidate {
		switch {
		case r >= '0' && r <= '9':
			digits = true
		case r == '.':
		default:
			return ""
		}
	}

	if !digits || strings.Contains(candidate, "..") {
		return ""
	}

	return candidate
}

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
