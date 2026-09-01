// Package useragent builds User-Agent strings for Stripe API requests.
package useragent

import (
	"encoding/json"
	"fmt"
	"os"
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
//
// Agent-specific variables are checked first. When none match it falls back to the two host
// variables DetectAgentHost reads, which identify an agent surface and so imply the agent.
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

	// No agent-specific variable matched. The two host variables below are set by these
	// agents to name their own surface, so their presence is itself evidence of an agent.
	//
	// Some surfaces set a host without the matching agent variable. The four generic Codex
	// signals above are sandbox-, permission- and CI-dependent, so a Codex Desktop session
	// configured with full access can set none of them while still naming itself through
	// the originator override. Claude Code's hosted and SDK entrypoints likewise do not
	// always set CLAUDECODE. Without this fallback those sessions report a host and no
	// agent, which also means the CLI treats them as interactive and tries to open a
	// browser to log in.
	if getEnv("CLAUDE_CODE_ENTRYPOINT") != "" {
		return "claude_code"
	}
	if getEnv("CODEX_INTERNAL_ORIGINATOR_OVERRIDE") != "" {
		return "codex_cli"
	}

	return ""
}

// DetectAgentHost reports where the agent ran, as a bounded category and as the host
// value itself. Both are empty when no agent reported a host and none can be inferred.
//
// kind is what to group by: "desktop", "terminal", "ide", "remote", "sdk", "mcp", or
// "other" for a host we have not categorized. raw is the normalized host underneath it,
// reported for every host rather than only uncategorized ones, so that a value we map
// too coarsely stays recoverable. "claude-desktop" and "claude-desktop-3p" are both
// desktop, for instance, and without raw nothing downstream can tell them apart.
//
// Reporting raw also means an unmapped host can be identified from the data rather than
// from a vendor's source, which matters because mapping one otherwise costs a code
// change, a release, and users upgrading before it is even visible.
//
// One host is inferred rather than reported: Codex names every surface except the terminal,
// so a detected Codex agent with no host is a terminal one. See the empty-host branch below.
func DetectAgentHost(getEnv func(string) string) (kind string, raw string) {
	host := getEnv("CLAUDE_CODE_ENTRYPOINT")
	if host == "" {
		// Codex Desktop sets this alongside the generic Codex signals, and it is
		// currently the only inherited value separating Desktop from the Codex CLI.
		host = getEnv("CODEX_INTERNAL_ORIGINATOR_OVERRIDE")
	}

	host = normalizeAgentHost(host)
	if host == "" {
		// Codex reports no host from a terminal. The override read above is set only by
		// non-default surfaces -- Desktop, the SDKs, the VS Code extension -- so a plain
		// `codex` run leaves it unset, and absence is the only signal the terminal case
		// has. Inferring it here is what separates terminal Codex from a CLI too old to
		// report a host at all, which otherwise arrives identically empty.
		//
		// Gated on the detected agent rather than on the Codex variables directly, so an
		// inferred host cannot contradict the agent reported alongside it. This cannot
		// recurse: DetectAIAgent reads only environment variables.
		//
		// Claude Code needs no equivalent because it names its terminal case "cli".
		if DetectAIAgent(getEnv) == "codex_cli" {
			return "terminal", "codex-cli"
		}

		return "", ""
	}

	// Checked first and by prefix. "remote-desktop" is Claude Desktop driving a session
	// that executes remotely -- Claude Code labels it "Claude Desktop" as a surface --
	// but this field describes where the CLI itself ran, and that is the remote host,
	// not the user's machine. Matching "desktop" anywhere would put it in the wrong one.
	if host == "remote" || strings.HasPrefix(host, "remote-") {
		return "remote", host
	}
	if kind, ok := agentHostKinds[host]; ok {
		return kind, host
	}

	return "other", host
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

// maxVersionLength and maxHostLength bound values before they are reported, so an
// unexpected or hostile value cannot bloat a request.
const (
	maxVersionLength = 32
	maxHostLength    = 32
)

//
// Private variables
//

// agentHostKinds categorizes the hosts we know about. "remote*" hosts are handled by
// prefix in DetectAgentHost rather than enumerated here.
//
// The codex-* entries are the originators Codex itself enumerates in
// KNOWN_ORIGINATOR_TAG_VALUES (codex-rs/otel/src/metrics/tags.rs), normalized. Mapping the
// whole set rather than only the values we have observed costs nothing and keeps a surface
// out of "other" the first time it appears. Codex's "none" is deliberately absent: it means
// no originator, so reporting it as an uncategorized host is the honest answer.
var agentHostKinds = map[string]string{
	"claude-desktop":       "desktop",
	"claude-desktop-3p":    "desktop",
	"claude-vscode":        "ide",
	"cli":                  "terminal",
	"codex-app-server":     "sdk",
	"codex-app-server-sdk": "sdk",
	"codex-cli":            "terminal",
	"codex-cli-rs":         "terminal",
	"codex-desktop":        "desktop",
	"codex-exec":           "terminal",
	"codex-mcp-server":     "mcp",
	"codex-sdk-ts":         "sdk",
	"codex-tui":            "terminal",
	"codex-vscode":         "ide",
	"mcp":                  "mcp",
	"sdk-cli":              "sdk",
	"sdk-py":               "sdk",
	"sdk-ts":               "sdk",
}

//
// Private functions
//

// normalizeAgentHost lowercases a reported host, collapses spaces and underscores to
// dashes, drops anything outside printable ASCII, and bounds the length. Vendors
// disagree on formatting, and Claude Code disagrees with itself: it ships both
// "claude_in_slack" and "claude-in-slack", while Codex reports "Codex Desktop".
//
// Normalizing before reporting matters because the host is reported as well as
// categorized: it keeps one host from arriving under several spellings, including
// across platforms.
func normalizeAgentHost(host string) string {
	host = strings.Map(func(r rune) rune {
		switch {
		case r == ' ' || r == '_':
			return '-'
		case r < ' ' || r > '~':
			return -1
		default:
			return r
		}
	}, strings.ToLower(strings.TrimSpace(host)))

	if len(host) > maxHostLength {
		host = host[:maxHostLength]
	}

	return host
}

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
