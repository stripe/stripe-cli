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
// Unlike Claude Code, Cursor has no `cursor plugin` CLI: the `cursor` binary is
// only the editor launcher. Enabled plugins are recorded in
// ~/Library/Application Support/Cursor/User/globalStorage/state.vscdb (macOS)
// under cursor.plugins.installedIds.* keys. When sqlite3 is available, this
// provider queries that registry; otherwise it falls back to prompting the user
// to run /add-plugin stripe inside Cursor.
//
// Cursor plugins are installed from inside Cursor via the `/add-plugin stripe`
// slash command; there is no shell CLI installer.
type CursorProvider struct {
	Scanner   Scanner
	RunOutput RunOutputFunc
}

// NewCursorProvider returns a Cursor setup provider. The RunCommandFunc argument
// is accepted for signature parity with the other providers but is unused,
// because Cursor has no CLI installer.
func NewCursorProvider(scanner Scanner, _ RunCommandFunc) Provider {
	return CursorProvider{
		Scanner:   scanner,
		RunOutput: runCommandOutput,
	}
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
	status.Status = StatusMissing

	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}

	pluginStatus, err := cursorStripePluginFromVscdb(s, runOutput)
	if err != nil {
		status.Error = cursorManualInstallHint
		return status
	}

	if pluginStatus.installed {
		status.Plugin.Installed = true
		status.Plugin.ID = TargetCursorPlugin
		status.Plugin.Scope = pluginStatus.scope
		status.Plugin.Project = pluginStatus.projectPath
		status.Status = StatusInstalled

		if home, homeErr := s.HomeDir(); homeErr == nil {
			status.Plugin.Version = cursorPluginVersionFromCache(s, home)
		}
		return status
	}

	// Signal to the TUI that this row is not actionable from the CLI.
	status.Error = cursorManualInstallHint
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
