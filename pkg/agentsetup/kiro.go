package agentsetup

const (
	// Kiro requires the user to add the Stripe plugin manually via the website
	ClientKiro            = "kiro"
	KiroBinaryName        = "kiro-cli"
	KiroDisplayName       = "Kiro"
	KiroManualInstruction = "add plugin manually using this link https://kiro.dev/launch/powers/add/?name=stripe"
)

func NewKiroProvider(scanner Scanner) Provider {
	return NewManualProvider(scanner, ClientKiro, KiroBinaryName, KiroDisplayName, KiroManualInstruction)
}
