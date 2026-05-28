// Package agent detects AI coding agents via well-known environment variables.
package agent

import "os"

// Detect reports whether the process was invoked by an AI coding agent.
// It checks the same env vars as stripe-cli's useragent.DetectAIAgent.
func Detect() bool {
	return DetectWith(os.Getenv)
}

// DetectWith is like Detect but accepts a custom env getter for testing.
func DetectWith(getEnv func(string) string) bool {
	for _, key := range envVars {
		if getEnv(key) != "" {
			return true
		}
	}
	return false
}

var envVars = []string{
	"ANTIGRAVITY_CLI_ALIAS",
	"CLAUDECODE",
	"CLINE_ACTIVE",
	"CODEX_SANDBOX",
	"CODEX_THREAD_ID",
	"CODEX_SANDBOX_NETWORK_DISABLED",
	"CODEX_CI",
	"CURSOR_AGENT",
	"GEMINI_CLI",
	"OPENCODE",
	"OPENCLAW_SHELL",
}
