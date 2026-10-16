package signals

// Output types for this package's substring/manifest code-signal scanning —
// honest about their strength: these are signals, never verdicts.

// TriageResult is one evaluated TriageBranch with the file-level evidence
// that fired it.
type TriageResult struct {
	Detected  string           `json:"detected"`
	Recommend string           `json:"recommendation"`
	Evidence  []TriageEvidence `json:"evidence"`
}

type TriageEvidence struct {
	File  string `json:"file"`
	Token string `json:"token"`
}

// HandlerSignals reports whether the USER'S code mentions each webhook event
// type the pack declares as expected for the migrated integration.
type HandlerSignals struct {
	Events     []EventSignal `json:"events"`
	AllPresent bool          `json:"all_present"`
}

type EventSignal struct {
	Event   string   `json:"event"`
	Files   []string `json:"files,omitempty"`
	Present bool     `json:"present"`
}

// FrontendWarning flags a pack-declared legacy client-side token found in the
// scanned code, with the pack's explanation of why it matters.
type FrontendWarning struct {
	File   string `json:"file"`
	Signal string `json:"signal"`
	Note   string `json:"note,omitempty"`
}

// ManifestCheck reports a pack-declared package version floor against what
// the repo's package.json actually declares.
type ManifestCheck struct {
	Package string `json:"package"`
	Floor   string `json:"floor"`
	Found   string `json:"found,omitempty"` // "" = package not present
	File    string `json:"file,omitempty"`
	OK      bool   `json:"ok"` // true when absent or >= floor
}
