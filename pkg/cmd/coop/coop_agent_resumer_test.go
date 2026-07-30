package coopcmd

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitCoopAgentPaneTagsReturnedPane(t *testing.T) {
	originalRunTmux := runTmux
	originalRunTmuxOutput := runTmuxOutput
	var calls [][]string
	runTmuxOutput = func(args ...string) (string, error) {
		calls = append(calls, append([]string(nil), args...))
		return "%7\n", nil
	}
	runTmux = func(args ...string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalRunTmuxOutput
	})

	pane, err := splitCoopAgentPane("-h", "bash", "-c", "agent")

	require.NoError(t, err)
	assert.Equal(t, "%7", pane)
	require.Len(t, calls, 2)
	assert.Equal(t, []string{"split-window", "-P", "-F", "#{pane_id}", "-h", "bash", "-c", "agent"}, calls[0])
	assert.Equal(t, []string{"set-option", "-p", "-t", "%7", coopAgentPaneOption, "1"}, calls[1])
}

func TestTmuxAgentResumerSerializesConcurrentWakeUps(t *testing.T) {
	originalRunTmux := runTmux
	originalRunTmuxOutput := runTmuxOutput
	var callsMu sync.Mutex
	var calls [][]string
	runTmuxOutput = func(args ...string) (string, error) {
		return "%1\t\n%2\t1\n", nil
	}
	runTmux = func(args ...string) error {
		callsMu.Lock()
		defer callsMu.Unlock()
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalRunTmuxOutput
	})

	resumer := newTmuxAgentResumer("%1")
	resumer.keyDelay = 0
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for _, sessionID := range []string{"session_one", "session_two", "session_three"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- resumer.Notify(sessionID)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	callsMu.Lock()
	defer callsMu.Unlock()
	require.Len(t, calls, 6)
	for i := 0; i < len(calls); i += 2 {
		require.Len(t, calls[i], 5)
		assert.Equal(t, []string{"send-keys", "-t", "%2", "-l"}, calls[i][:4])
		assert.True(t, strings.Contains(calls[i][4], "stripe coop agent resume --session=session_"))
		assert.Equal(t, []string{"send-keys", "-t", "%2", "Enter"}, calls[i+1])
	}
}
