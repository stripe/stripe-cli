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

// DetectAgentHostKind reports where the agent ran: "desktop", "terminal", "ide",
// "remote", "sdk", "mcp", or "other", and "" when no agent reported a host.
//
// Only the category is exposed. The underlying values are per-vendor spellings that
// every consumer would otherwise have to enumerate, and the vendor is already carried
// by DetectAIAgent.
func DetectAgentHostKind(getEnv func(string) string) string {
	host := getEnv("CLAUDE_CODE_ENTRYPOINT")
	if host == "" {
		// Codex Desktop sets this alongside the generic Codex signals, and it is
		// currently the only inherited value separating Desktop from the Codex CLI.
		host = getEnv("CODEX_INTERNAL_ORIGINATOR_OVERRIDE")
	}
	if host == "" {
		return ""
	}

	// Vendors disagree on formatting, and Claude Code disagrees with itself: it ships
	// both "claude_in_slack" and "claude-in-slack", while Codex reports "Codex Desktop".
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.NewReplacer(" ", "-", "_", "-").Replace(host)

	// Checked first and by prefix: "remote-desktop" is a remote session, not the
	// desktop app, so matching "desktop" anywhere would silently inflate desktop counts.
	if host == "remote" || strings.HasPrefix(host, "remote-") {
		return "remote"
	}
	if kind, ok := agentHostKinds[host]; ok {
		return kind
	}
	// Reported rather than dropped, so a host we have not categorized stays
	// distinguishable from no host at all while the field stays bounded.
	return "other"
}

// DetectAgentVersion returns the agent's own version, parsed out of the identifier it
// reports through the AI_AGENT convention, or "" when absent or unparseable.
//
// Claude Code encodes it as "claude-code_2-1-227_agent" (dots written as dashes) and
// has also used "claude-code/2.1.227". No other agent reports a version this way today.
// Both formats are undocumented, so this validates what it extracts and reports nothing
// rather than guessing: a wrong version is worse than a missing one.
func DetectAgentVersion(getEnv func(string) string) string {
	identifier := strings.TrimSpace(getEnv("AI_AGENT"))
	if identifier == "" {
		// AGENT is the same convention under the name Goose, Amp and Bun use.
		identifier = strings.TrimSpace(getEnv("AGENT"))
	}

	if idx := strings.LastIndex(identifier, "/"); idx >= 0 {
		return validVersion(identifier[idx+1:])
	}
	for _, segment := range strings.Split(identifier, "_") {
		if version := validVersion(strings.ReplaceAll(segment, "-", ".")); version != "" {
			return version
		}
	}
	return ""
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

// maxVersionLength bounds a parsed version before it is reported, so an unexpected or
// hostile value cannot bloat a request.
const maxVersionLength = 32

//
// Private variables
//

// agentHostKinds categorizes the hosts we know about. "remote*" hosts are handled by
// prefix in DetectAgentHostKind rather than enumerated here.
var agentHostKinds = map[string]string{
	"claude-desktop": "desktop",
	"claude-vscode":  "ide",
	"cli":            "terminal",
	"codex-desktop":  "desktop",
	"mcp":            "mcp",
	"sdk-cli":        "sdk",
	"sdk-py":         "sdk",
	"sdk-ts":         "sdk",
}

//
// Private functions
//

// validVersion returns the candidate if it is a dot-separated numeric version, and ""
// otherwise. Deliberately strict: the identifier formats this parses are undocumented,
// so anything unexpected is dropped rather than guessed at.
func validVersion(candidate string) string {
	if candidate == "" || len(candidate) > maxVersionLength ||
		strings.HasPrefix(candidate, ".") || strings.HasSuffix(candidate, ".") ||
		strings.Contains(candidate, "..") {
		return ""
	}

	for _, r := range candidate {
		if (r < '0' || r > '9') && r != '.' {
			return ""
		}
	}

	return candidate
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
