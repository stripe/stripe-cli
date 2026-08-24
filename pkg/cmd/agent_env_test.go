package cmd

import "testing"

// clearAgentEnv unsets every variable useragent.DetectAIAgent reads, so a test asserting
// "no agent" is not answering with the environment of whichever agent is running it.
//
// This covers the host variables as well as the agent-specific ones. DetectAIAgent falls
// back to CLAUDE_CODE_ENTRYPOINT and CODEX_INTERNAL_ORIGINATOR_OVERRIDE when no
// agent-specific variable matched, because some surfaces set only a host. That makes an
// incomplete list here fail only when the suite runs inside an agent, which is the worst
// time to find out.
func clearAgentEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"ANTIGRAVITY_CLI_ALIAS",
		"CLAUDECODE",
		"CLAUDE_CODE_ENTRYPOINT",
		"CLINE_ACTIVE",
		"CODEX_CI",
		"CODEX_INTERNAL_ORIGINATOR_OVERRIDE",
		"CODEX_SANDBOX",
		"CODEX_SANDBOX_NETWORK_DISABLED",
		"CODEX_THREAD_ID",
		"CURSOR_AGENT",
		"GEMINI_CLI",
		"OPENCLAW_SHELL",
		"OPENCODE",
	} {
		t.Setenv(key, "")
	}
}
