package cmd

import "github.com/stripe/stripe-cli/pkg/useragent"

// interactiveHuman reports whether the CLI is talking to a person at a terminal.
// Agents with a real TTY (e.g. Claude Code) and headless environments (CI,
// /dev/null stdin) both return false, so nothing blocks waiting for an answer
// nobody is there to give.
// getEnv and stdinIsTerminal are injected to allow testing.
func interactiveHuman(getEnv func(string) string, stdinIsTerminal bool) bool {
	return stdinIsTerminal && useragent.DetectAIAgent(getEnv) == ""
}

// shouldAutoLogin reports whether the CLI should attempt automatic browser-based login.
// Returns true only when stdin is an interactive terminal and no AI agent is detected.
func shouldAutoLogin(getEnv func(string) string, stdinIsTerminal bool) bool {
	return interactiveHuman(getEnv, stdinIsTerminal)
}
