package doctor

// reports.go — the machine contract. Every subcommand produces one of these
// structs; --json marshals it to stdout verbatim (logs go to stderr), and the
// text renderers in cmd.go present the same data to humans. Exit codes:
//
//	0  clean / verified
//	1  findings present / verification failed
//	2  operational error
//
// Agents should treat these schemas plus the exit codes as the interface;
// `stripe doctor guide` documents the step-by-step playbook.

// ScanReport is scan-stage data (kept for internal use and tests).
type ScanReport struct {
	Command  string    `json:"command"`
	Findings []Finding `json:"findings"`
	Stats    ScanStats `json:"stats"`
}

type ScanStats struct {
	FilesScanned int `json:"files_scanned"`
	FilesParsed  int `json:"files_parsed"`
	Skipped      int `json:"skipped_by_prefilter"`
}

// DoctorReport is the output of `stripe doctor`: scan + account facts +
// verdicts, optionally with live behavioral checks. Degraded is set (and
// Account empty) when account facts were unavailable — findings then carry
// verdict_class UNKNOWN.
type DoctorReport struct {
	Command   string          `json:"command"`
	Topic     string          `json:"topic"`
	Degraded  string          `json:"degraded,omitempty"`
	Account   AccountSummary  `json:"account,omitzero"`
	Findings  []DoctorFinding `json:"findings"`
	Summary   map[string]int  `json:"summary"` // verdict class -> count
	Stats     ScanStats       `json:"stats"`
	LiveDrill *DrillReport    `json:"live_drill,omitempty"`
	// Code signals (substring-level, honest about their strength):
	WebhookHandlers *HandlerSignals   `json:"webhook_handlers,omitempty"`
	FrontendSignals []FrontendWarning `json:"frontend_warnings,omitempty"`
	ManifestChecks  []ManifestCheck   `json:"manifest_checks,omitempty"`
	// Triage: which integration from-states were detected and which migration
	// guide applies to each — the "which of these three docs do I follow"
	// answer, with file-level evidence.
	Triage []TriageResult `json:"triage,omitempty"`
}

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

type AccountSummary struct {
	ID             string         `json:"id"`
	Name           string         `json:"name,omitempty"`
	EventVersions  map[string]int `json:"event_api_versions"`
	VersionsOK     bool           `json:"versions_at_or_after_cutoff"`
	VersionsMixed  bool           `json:"versions_mixed"`
	Cutoff         string         `json:"dpm_cutoff"`
	Configs        int            `json:"payment_method_configurations"`
	ActiveConfig   string         `json:"active_configuration,omitempty"`
	ActiveConfigID string         `json:"active_configuration_id,omitempty"`
	MethodsOn      int            `json:"methods_on"`
	MethodsOff     int            `json:"methods_off"`
	EnabledMethods []string       `json:"enabled_methods,omitempty"`
	// Unavailable: toggled ON but capability inactive — will not render.
	Unavailable    []string `json:"enabled_but_unavailable,omitempty"`
	NoRecentEvents bool     `json:"no_recent_events"`
	ConfiguredOK   bool     `json:"dashboard_configured"`
}

type DoctorFinding struct {
	Finding
	Intent  string `json:"intent"`
	Verdict string `json:"verdict"`
	Class   string `json:"verdict_class"` // CANDIDATE | SKIP | REVIEW | CAUTION | BLOCKED
}

// DrillReport is the webhook round-trip evidence (doctor --live).
type DrillReport struct {
	Command     string       `json:"command"`
	ListenReady bool         `json:"listen_ready"`
	Triggered   string       `json:"triggered_event,omitempty"`
	Events      []DrillEvent `json:"events_received"`
	Verified    bool         `json:"verified"`
	Note        string       `json:"note,omitempty"`
}

type DrillEvent struct {
	Type      string `json:"type"`
	Signature string `json:"signature"` // verified | invalid
}
