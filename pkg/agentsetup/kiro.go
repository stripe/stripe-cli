package agentsetup

const (
	// Kiro requires the user to add the Stripe plugin manually via the Kiro IDE or website.
	ClientKiro            = "kiro"
	KiroBinaryName        = "kiro-cli"
	KiroDisplayName       = "Kiro"
	KiroManualInstruction = "add plugin manually via Kiro IDE or website"
)

func NewKiroProvider(scanner Scanner) Provider {
	return NewManualProvider(scanner, ClientKiro, KiroBinaryName, KiroDisplayName, KiroManualInstruction)
}
