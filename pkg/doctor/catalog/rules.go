package catalog

// rules.go — the topic registry. dpm is the only topic today (the steel
// thread); other migrations previously lived here as detect-only "advise"
// topics and were removed to keep this tool scoped to one working path.
// They may return later — see git history for the removed rules.
//
// Adding a migration means adding an entry to Rules — the verbs never
// change. Code signals (expected webhook events, etc.) aren't declared
// here: a remediation implementation that cares about them declares and
// scans for its own — see pkg/doctor/remediation/paymentmethods for the
// dpm example — so a product team owns that decision the same way it owns
// the rest of its remediation's logic.
//
// dpm's Rule comes from the embedded dpm.json (see dpm.go) for now, not a
// live fetch — judge.FetchDPMRule still exists for when that's reinstated.

// Rules is the topic registry: every migration doctor knows about, keyed
// by the topic name used on the command line (`stripe doctor <topic> <dir>`).
var Rules = map[string]Rule{
	"dpm": DpmRule,
}
