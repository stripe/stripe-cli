package agentsetup

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCursor_NotDetected(t *testing.T) {
	provider := CursorProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }},
	}

	status := provider.Detect()

	require.Equal(t, ClientCursor, status.Client)
	require.Equal(t, "Cursor", status.DisplayName)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
}

func TestCursor_DetectedAlwaysUnknown(t *testing.T) {
	provider := CursorProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/cursor", nil }},
	}

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, "/usr/local/bin/cursor", status.ExecutablePath)
	require.Equal(t, StatusUnknown, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestCursor_PlanManualWhenNotInstalled(t *testing.T) {
	provider := CursorProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/cursor", nil }},
	}

	status := provider.Detect()
	plan := provider.Plan(status, false)

	require.Equal(t, ActionManual, plan.Action)
	require.Contains(t, plan.Manual, "/add-plugin stripe")
}

func TestCursor_PlanNoneWhenNotDetected(t *testing.T) {
	provider := CursorProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }},
	}

	status := provider.Detect()
	plan := provider.Plan(status, false)

	require.Equal(t, ActionNone, plan.Action)
}

func TestCursor_PlanNoneWhenInstalled(t *testing.T) {
	provider := CursorProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/cursor", nil }},
	}

	status := provider.Detect()
	status.Plugin.Installed = true
	plan := provider.Plan(status, false)

	require.Equal(t, ActionNone, plan.Action)
}

func TestCursor_ErrorHintForTUIDisable(t *testing.T) {
	provider := CursorProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/cursor", nil }},
	}

	status := provider.Detect()

	require.Contains(t, status.Error, "/add-plugin stripe")
}
