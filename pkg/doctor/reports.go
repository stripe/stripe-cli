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

// FixReport is the output of `stripe doctor fix` (dry-run by default; --apply writes).
type FixReport struct {
	Command  string    `json:"command"`
	Topic    string    `json:"topic,omitempty"`
	Applied  bool      `json:"applied"`
	AllClean bool      `json:"all_reparse_clean"`
	Files    []FixFile `json:"files"`
	// Companion reports the version fork: whether removals were replaced with
	// the rule's companion parameter, and the account evidence that decided it.
	Companion *CompanionReport `json:"companion,omitempty"`
	// Skipped findings the gate excluded (dynamic values, deliberate
	// restrictions) — removed only with --all.
	Skipped []SkippedFinding `json:"skipped,omitempty"`
}

// CompanionReport is the account-forked replace decision. Mode "insert" means
// removals of the companioned parameter become replacements; "omit" means the
// account's traffic is entirely at/after the rule's cutoff so plain removal is
// behavior-preserving.
type CompanionReport struct {
	Param         string `json:"param"`
	Mode          string `json:"mode"` // insert | omit
	Reason        string `json:"reason"`
	OldestVersion string `json:"oldest_traffic_version,omitempty"`
	// Account is the Stripe account whose facts decided the fork ("" when
	// the decision was made without account access).
	Account string `json:"account,omitempty"`
	Inserts int    `json:"inserts"` // replacements actually made
	// PinnedSites lists exactly which findings received the
	// allow_redirects:"never" pin — the structured form of the notes.
	PinnedSites []SiteRef `json:"pinned_sites,omitempty"`
	// Notes surface site-level caveats: server-side-confirmation sites that
	// were pinned to allow_redirects:"never", or that gained --return-url.
	Notes []string `json:"notes,omitempty"`
}

type SiteRef struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

type SkippedFinding struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Intent string `json:"intent"`
	Reason string `json:"reason"`
}

type FixFile struct {
	Path         string    `json:"path"`
	Error        string    `json:"error,omitempty"` // write failure; file NOT written
	BytesRemoved int       `json:"bytes_removed"`
	BytesAdded   int       `json:"bytes_added,omitempty"` // companion insertions
	Edits        []FixEdit `json:"edits"`
	Reparse      string    `json:"reparse"` // clean | error
	Written      bool      `json:"written"`
}

type FixEdit struct {
	Start uint32 `json:"start_byte"`
	End   uint32 `json:"end_byte"`
	Label string `json:"label"`
	// Variant marks companion edits: plain | never-pin | return_url.
	Variant string `json:"variant,omitempty"`
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
