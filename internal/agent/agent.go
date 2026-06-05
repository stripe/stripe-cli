// Package agent detects AI coding agents via well-known environment variables.
package agent

import "os"

// Agent identifies which AI coding agent (if any) invoked the process.
type Agent string

// NotDetected is the zero value, returned when no known agent is found.
const NotDetected Agent = ""

// Known AI coding agents.
const (
	Antigravity Agent = "antigravity"
	ClaudeCode  Agent = "claude_code"
	Cline       Agent = "cline"
	CodexCLI    Agent = "codex_cli"
	Cursor      Agent = "cursor"
	GeminiCLI   Agent = "gemini_cli"
	OpenCode    Agent = "open_code"
	Openclaw    Agent = "openclaw"
)

// agentDef pairs an Agent value with the env vars that indicate its presence.
type agentDef struct {
	agent   Agent
	envKeys []string
}

// agents is the canonical list; the first entry whose env key is set wins.
var agents = []agentDef{
	{Antigravity, []string{"ANTIGRAVITY_CLI_ALIAS"}},
	{ClaudeCode, []string{"CLAUDECODE"}},
	{Cline, []string{"CLINE_ACTIVE"}},
	{CodexCLI, []string{"CODEX_SANDBOX", "CODEX_THREAD_ID", "CODEX_SANDBOX_NETWORK_DISABLED", "CODEX_CI"}},
	{Cursor, []string{"CURSOR_AGENT"}},
	{GeminiCLI, []string{"GEMINI_CLI"}},
	{OpenCode, []string{"OPENCODE"}},
	{Openclaw, []string{"OPENCLAW_SHELL"}},
}

// Detect reports which AI coding agent invoked the process, or NotDetected.
// It checks the same env vars as stripe-cli's useragent.DetectAIAgent.
func Detect() Agent {
	return DetectWith(os.Getenv)
}

// DetectWith is like Detect but accepts a custom env getter for testing.
func DetectWith(getEnv func(string) string) Agent {
	for _, def := range agents {
		for _, key := range def.envKeys {
			if getEnv(key) != "" {
				return def.agent
			}
		}
	}
	return NotDetected
}
