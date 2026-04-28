package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteClaudeCodePluginHint_EmitsWhenSet(t *testing.T) {
	var buf bytes.Buffer
	getEnv := func(key string) string {
		if key == "CLAUDECODE" {
			return "1"
		}
		return ""
	}

	writeClaudeCodePluginHint(getEnv, &buf)

	assert.Equal(t, claudeCodePluginHint+"\n", buf.String())
}

func TestWriteClaudeCodePluginHint_SilentWhenUnset(t *testing.T) {
	var buf bytes.Buffer
	getEnv := func(string) string { return "" }

	writeClaudeCodePluginHint(getEnv, &buf)

	assert.Empty(t, buf.String())
}
