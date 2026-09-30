package agentsetup

const (
	// Cursor requires the user to run a slash command from inside the agent to install the Stripe plugin.
	ClientCursor            = "cursor"
	CursorBinaryName        = "cursor"
	CursorDisplayName       = "Cursor"
	CursorManualInstruction = "run /add-plugin stripe inside Cursor"
)

func NewCursorProvider(scanner Scanner) Provider {
	return NewManualProvider(scanner, ClientCursor, CursorBinaryName, CursorDisplayName, CursorManualInstruction)
}
