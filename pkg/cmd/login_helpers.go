package cmd

import "github.com/stripe/stripe-cli/pkg/useragent"

// shouldAutoLogin reports whether the CLI should attempt automatic browser-based login.
// Returns true only when stdin is an interactive terminal and no AI agent is detected.
// Agents with a real TTY (e.g. Claude Code) and headless environments (CI, /dev/null stdin)
// both return false, ensuring neither blocks waiting for a browser flow.
// getEnv and stdinIsTerminal are injected to allow testing.
func shouldAutoLogin(getEnv func(string) string, stdinIsTerminal bool) bool {
	return stdinIsTerminal && useragent.DetectAIAgent(getEnv) == ""
}
