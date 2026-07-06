package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

type mockUnknownCmdTelemetryClient struct {
	mu     sync.Mutex
	events []struct {
		name  string
		value string
	}
}

func (m *mockUnknownCmdTelemetryClient) SendAPIRequestEvent(_ context.Context, _ string, _ bool) (*http.Response, error) {
	return nil, nil
}

func (m *mockUnknownCmdTelemetryClient) SendEvent(_ context.Context, eventName string, eventValue string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, struct {
		name  string
		value string
	}{eventName, eventValue})
}

func TestRecordUnknownCommand_NotInAgentEnv(t *testing.T) {
	t.Setenv("CLAUDECODE", "")
	t.Setenv("CURSOR_AGENT", "")
	t.Setenv("CODEX_SANDBOX", "")
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("CODEX_SANDBOX_NETWORK_DISABLED", "")
	t.Setenv("CODEX_CI", "")
	t.Setenv("CLINE_ACTIVE", "")
	t.Setenv("GEMINI_CLI", "")
	t.Setenv("OPENCODE", "")
	t.Setenv("OPENCLAW_SHELL", "")
	t.Setenv("ANTIGRAVITY_CLI_ALIAS", "")

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	ctx := context.Background()
	mock := &mockUnknownCmdTelemetryClient{}
	ctx = stripe.WithTelemetryClient(ctx, mock)

	recordUnknownCommand(ctx, "stripe foobar")

	batchPath := filepath.Join(tmpDir, "stripe", unknownCmdBatchFile)
	_, err := os.Stat(batchPath)
	assert.True(t, os.IsNotExist(err))
}

func TestRecordUnknownCommand_InAgentEnv_BatchesCommands(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	ctx := context.Background()
	mock := &mockUnknownCmdTelemetryClient{}
	ctx = stripe.WithTelemetryClient(ctx, mock)

	for i := 0; i < 9; i++ {
		recordUnknownCommand(ctx, "stripe foobar")
	}

	batchPath := filepath.Join(tmpDir, "stripe", unknownCmdBatchFile)
	data, err := os.ReadFile(batchPath)
	require.NoError(t, err)

	var entries []unknownCommandEntry
	require.NoError(t, json.Unmarshal(data, &entries))
	assert.Len(t, entries, 9)
	assert.Equal(t, "stripe foobar", entries[0].Command)
	assert.Equal(t, "claude_code", entries[0].Agent)

	mock.mu.Lock()
	assert.Len(t, mock.events, 0)
	mock.mu.Unlock()
}

func TestRecordUnknownCommand_InAgentEnv_SendsAtBatchSize(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	ctx := context.Background()
	mock := &mockUnknownCmdTelemetryClient{}
	ctx = stripe.WithTelemetryClient(ctx, mock)

	for i := 0; i < unknownCmdBatchSize; i++ {
		recordUnknownCommand(ctx, "stripe bazqux")
	}

	batchPath := filepath.Join(tmpDir, "stripe", unknownCmdBatchFile)
	_, err := os.Stat(batchPath)
	assert.True(t, os.IsNotExist(err))

	require.Eventually(t, func() bool {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		return len(mock.events) == 1
	}, 1*time.Second, 10*time.Millisecond)

	mock.mu.Lock()
	defer mock.mu.Unlock()
	assert.Equal(t, unknownCmdEventName, mock.events[0].name)

	var entries []unknownCommandEntry
	require.NoError(t, json.Unmarshal([]byte(mock.events[0].value), &entries))
	assert.Len(t, entries, 10)
	assert.Equal(t, "stripe bazqux", entries[0].Command)
	assert.Equal(t, "claude_code", entries[0].Agent)
}

func TestRecordUnknownCommand_NoTelemetryClient(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	ctx := context.Background()
	recordUnknownCommand(ctx, "stripe foobar")

	batchPath := filepath.Join(tmpDir, "stripe", unknownCmdBatchFile)
	_, err := os.Stat(batchPath)
	assert.True(t, os.IsNotExist(err))
}
