package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteClaudeCodePluginHint_EmitsWhenSet(t *testing.T) {
	t.Setenv("AI_AGENT", "")
	t.Setenv("CLAUDECODE", "1")
	var buf bytes.Buffer

	writeClaudeCodePluginHint(&buf)

	assert.Equal(t, claudeCodePluginHint+"\n", buf.String())
}

func TestWriteClaudeCodePluginHint_SilentWhenUnset(t *testing.T) {
	t.Setenv("AI_AGENT", "")
	t.Setenv("CLAUDECODE", "")
	var buf bytes.Buffer

	writeClaudeCodePluginHint(&buf)

	assert.Empty(t, buf.String())
}
