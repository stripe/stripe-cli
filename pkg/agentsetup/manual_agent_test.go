package agentsetup

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManual_NotDetected(t *testing.T) {
	scanner := Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }}
	provider := NewCursorProvider(scanner)

	status := provider.Detect()

	require.Equal(t, ClientCursor, status.Client)
	require.Equal(t, "Cursor", status.DisplayName)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
}

func TestManual_DetectedAlwaysUnknown(t *testing.T) {
	provider := cursorTestProvider()

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, "/usr/local/bin/cursor", status.ExecutablePath)
	require.Equal(t, StatusUnknown, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestManual_PlanManualWhenNotInstalled(t *testing.T) {
	provider := cursorTestProvider()

	status := provider.Detect()
	plan := provider.Plan(status, false)

	require.Equal(t, ActionManual, plan.Action)
	require.Contains(t, plan.Manual, "/add-plugin stripe")
}

func TestManual_PlanNoneWhenNotDetected(t *testing.T) {
	scanner := Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }}
	provider := NewManualProvider(scanner, ClientCursor, CursorBinaryName, CursorDisplayName, CursorManualInstruction)

	status := provider.Detect()
	plan := provider.Plan(status, false)

	require.Equal(t, ActionNone, plan.Action)
}

func TestManual_PlanNoneWhenInstalled(t *testing.T) {
	provider := cursorTestProvider()

	status := provider.Detect()
	status.Plugin.Installed = true
	plan := provider.Plan(status, false)

	require.Equal(t, ActionNone, plan.Action)
}

func cursorTestProvider() Provider {
	scanner := Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/cursor", nil }}
	return NewCursorProvider(scanner)
}
