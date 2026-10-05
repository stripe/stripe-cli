package agentsetup

const (
	// Kiro requires the user to add the Stripe plugin manually via the website
	ClientKiro            = "kiro"
	KiroBinaryName        = "kiro-cli"
	KiroDisplayName       = "Kiro"
	KiroManualInstruction = "run /powers install stripe inside Kiro"
)

func NewKiroProvider(scanner Scanner) Provider {
	return NewManualProvider(scanner, ClientKiro, KiroBinaryName, KiroDisplayName, KiroManualInstruction)
}
