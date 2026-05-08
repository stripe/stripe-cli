package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/stripe/stripe-cli/pkg/useragent"
)

const claudeCodePluginHint = `<claude-code-hint v="1" type="plugin" value="stripe@claude-plugins-official" />`

// emitClaudeCodePluginHint writes a plugin hint to stderr when running inside
// Claude Code. Claude Code strips the hint from command output and may prompt
// the user to install the Stripe plugin. See
// https://code.claude.com/docs/en/plugin-hints for the protocol.
func emitClaudeCodePluginHint() {
	writeClaudeCodePluginHint(os.Getenv, os.Stderr)
}

func writeClaudeCodePluginHint(getEnv func(string) string, w io.Writer) {
	if useragent.DetectAIAgent(getEnv) != "claude_code" {
		return
	}
	fmt.Fprintln(w, claudeCodePluginHint)
}
