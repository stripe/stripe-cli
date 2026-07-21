package agentsetup

import (
	"context"
	"io"
)

const (
	ClientCursor      = "cursor"
	CursorBinaryName  = "cursor"
	CursorDisplayName = "Cursor"
)

// CursorProvider detects the Stripe plugin for Cursor.
//
// Unlike Claude Code, Cursor has no `cursor plugin` CLI and no JSON registry:
// the `cursor` binary is only the editor launcher, and installed plugins are
// recorded as a directory tree under ~/.cursor/plugins/cache/<marketplace>/
// <plugin>/<hash>/ (a `.cache-complete` marker plus a .cursor-plugin/plugin.json
// metadata file). Detection therefore walks the filesystem.
//
// Cursor plugins are installed from inside Cursor via the `/add-plugin stripe`
// slash command; there is no shell CLI installer. This provider therefore
// detects the plugin from disk and, when it is missing, points the user at the
// in-app command rather than shelling out.
type CursorProvider struct {
	Scanner Scanner
}

// NewCursorProvider returns a Cursor setup provider. The RunCommandFunc argument
// is accepted for signature parity with the other providers but is unused,
// because Cursor has no CLI installer.
func NewCursorProvider(scanner Scanner, _ RunCommandFunc) Provider {
	return CursorProvider{Scanner: scanner}
}

func (p CursorProvider) ID() string { return ClientCursor }

func (p CursorProvider) Detect() Status {
	s := p.Scanner.withDefaults()

	status := Status{
		Client:      ClientCursor,
		DisplayName: CursorDisplayName,
		Status:      StatusNotDetected,
	}

	binPath, err := s.LookPath(CursorBinaryName)
	if err != nil {
		return status
	}
	status.Detected = true
	status.ExecutablePath = binPath
	status.Status = StatusUnknown
	// Signal to the TUI that this row is not actionable from the CLI.
	status.Error = "run /add-plugin stripe inside Cursor agent"
	return status
}

func (p CursorProvider) Plan(status Status, _ bool) Plan {
	if status.Detected && !status.Plugin.Installed {
		return Plan{
			Action: ActionManual,
			Manual: "run /add-plugin stripe inside Cursor",
		}
	}
	return Plan{Action: ActionNone}
}

// Apply is a no-op for Cursor: the plugin is installed from inside Cursor, so the
// setup flow surfaces the ActionManual instruction rather than calling Apply.
func (p CursorProvider) Apply(_ context.Context, _ io.Writer, _ Plan) error {
	return nil
}
