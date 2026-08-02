package coopcmd

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

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
		return "%1\t\t0\n%2\t1\t0\n", nil
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
	require.Len(t, calls, 9)
	for i := 0; i < len(calls); i += 3 {
		assert.Equal(t, []string{"send-keys", "-t", "%2", "C-u"}, calls[i])
		require.Len(t, calls[i+1], 5)
		assert.Equal(t, []string{"send-keys", "-t", "%2", "-l"}, calls[i+1][:4])
		assert.True(t, strings.Contains(calls[i+1][4], "stripe coop agent resume --session=session_"))
		assert.Equal(t, []string{"send-keys", "-t", "%2", "Enter"}, calls[i+2])
	}
}

func TestTmuxAgentResumerSkipsInjectionWhileAgentIsParked(t *testing.T) {
	originalRunTmux := runTmux
	originalRunTmuxOutput := runTmuxOutput
	var calls [][]string
	record := func(args ...string) {
		calls = append(calls, append([]string(nil), args...))
	}
	runTmuxOutput = func(args ...string) (string, error) {
		record(args...)
		return "%2\t1\t0\n", nil
	}
	runTmux = func(args ...string) error {
		record(args...)
		return nil
	}
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalRunTmuxOutput
	})

	resumer := newTmuxAgentResumer("%1")
	resumer.keyDelay = 0
	resumer.heartbeatAge = func(sessionID string) (time.Duration, error) {
		assert.Equal(t, "session_parked", sessionID)
		return 500 * time.Millisecond, nil
	}

	require.NoError(t, resumer.Notify("session_parked"))
	assert.Empty(t, calls, "a parked agent reads the decision from the session file; nothing should be injected")
}

func TestTmuxAgentResumerInjectsWhenHeartbeatMissing(t *testing.T) {
	originalRunTmux := runTmux
	originalRunTmuxOutput := runTmuxOutput
	var calls [][]string
	runTmuxOutput = func(args ...string) (string, error) {
		calls = append(calls, append([]string(nil), args...))
		return "%2\t1\t0\n", nil
	}
	runTmux = func(args ...string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalRunTmuxOutput
	})

	resumer := newTmuxAgentResumer("%1")
	resumer.keyDelay = 0
	resumer.heartbeatAge = func(string) (time.Duration, error) {
		return 0, errors.New("no heartbeat")
	}

	require.NoError(t, resumer.Notify("session_idle"))
	require.Len(t, calls, 4) // list-panes, C-u, prompt, Enter
	assert.Equal(t, []string{"send-keys", "-t", "%2", "C-u"}, calls[1])
}

func TestTmuxAgentResumerRefusesDeadAgentPane(t *testing.T) {
	originalRunTmux := runTmux
	originalRunTmuxOutput := runTmuxOutput
	var sends [][]string
	runTmuxOutput = func(args ...string) (string, error) {
		return "%2\t1\t1\n", nil
	}
	runTmux = func(args ...string) error {
		sends = append(sends, append([]string(nil), args...))
		return nil
	}
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalRunTmuxOutput
	})

	resumer := newTmuxAgentResumer("%1")
	err := resumer.Notify("session_dead")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exited")
	assert.Empty(t, sends, "no keystrokes may be sent into a dead pane")
}
