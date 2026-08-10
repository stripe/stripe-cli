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
// Values are passed through as reported rather than mapped to a fixed set, since the
// set evolves outside this CLI. Known Claude Code values include "cli",
// "claude-desktop", "claude-vscode", "mcp", "sdk-ts", "sdk-py", "local-agent",
// "claude-code-github-action", and several "remote*" variants.
func DetectAgentHost(getEnv func(string) string) string {
	return sanitizeEnvValue(getEnv("CLAUDE_CODE_ENTRYPOINT"))
}

// DetectMacAppBundleID returns the bundle identifier of the macOS application that
// launched the process tree, or "" on other platforms and when unset.
//
// macOS sets __CFBundleIdentifier when an app bundle launches a process, and child
// processes inherit it. That inheritance means it identifies the GUI application at
// the root of the session rather than the direct parent: launching from a terminal
// emulator reports that emulator (e.g. "com.apple.Terminal"), and it is absent
// entirely under ssh, cron, and CI. It is therefore a signal that *some* Mac app
// started the session, not proof of who invoked the CLI.
func DetectMacAppBundleID(getEnv func(string) string) string {
	return sanitizeEnvValue(getEnv("__CFBundleIdentifier"))
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
// Private functions
//

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
