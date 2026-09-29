package agentsetup

import (
	"context"
	"io"
)

// ManualProvider detects a client's Stripe plugin that must be installed by
// the user from inside that client, rather than via a shell CLI installer.
//
// Some clients (e.g. Cursor, Kiro) have no CLI installer or JSON registry for
// their plugins. This provider points the user to a caller-supplied instruction rather
// than trying to install for them.
type ManualProvider struct {
	ProviderConfig
	ManualInstruction string
}

// NewManualProvider returns a setup provider for a client whose plugin must
// be installed manually; these clients have no CLI installer to run.
func NewManualProvider(scanner Scanner, client, binaryName, displayName, manualInstruction string) Provider {
	return ManualProvider{
		ProviderConfig: ProviderConfig{
			Scanner:     scanner,
			Client:      client,
			BinaryName:  binaryName,
			DisplayName: displayName,
		},
		ManualInstruction: manualInstruction,
	}
}

func (p ManualProvider) ID() string { return p.Client }

func (p ManualProvider) Detect() Status {
	status := detectAgentExecutable(
		p.ProviderConfig,
		StatusUnknown,
	)
	if !status.Detected {
		return status
	}
	// Signal to the TUI that this row is not actionable from the CLI.
	status.Error = p.ManualInstruction
	return status
}

func (p ManualProvider) Plan(status Status, _ bool) Plan {
	if status.Detected && !status.Plugin.Installed {
		return Plan{
			Action: ActionManual,
			Manual: p.ManualInstruction,
		}
	}
	return Plan{Action: ActionNone}
}

// Apply is a no-op: the plugin is installed from inside the client, so the
// setup flow surfaces the ActionManual instruction rather than calling Apply.
func (p ManualProvider) Apply(_ context.Context, _ io.Writer, _ Plan) error {
	return nil
}
