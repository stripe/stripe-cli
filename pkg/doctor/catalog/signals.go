package catalog

// Pack-declared code signals: expected webhook events, legacy frontend
// tokens, package version floors, and the which-migration triage. These are
// pure data, authored alongside a Rule in packs.go or catalog.go; the code
// that scans a repo for them lives in the signals package.

// PackSignals are a pack's declared code signals: which webhook events the
// migrated integration must handle, which frontend tokens indicate
// from-state code, and which package version floors apply.
type PackSignals struct {
	WebhookEvents  []string
	FrontendTokens []FrontendToken
	ManifestFloors []ManifestFloor
}

type FrontendToken struct {
	Token string
	Note  string
}

type ManifestFloor struct {
	Package string
	Min     string // "8.0.0"
}

// TriageBranch describes one detectable integration from-state and which
// migration guide applies to it. Branches are evaluated independently — a
// repo can legitimately be in several states at once (that is the point of
// triage).
type TriageBranch struct {
	Detected  string   // what this evidence means
	Recommend string   // which topic/guide to follow, and why
	Tokens    []string // code tokens whose presence constitutes evidence
}
