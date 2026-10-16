package catalog

// packs.go — the topic registry. dpm is the only topic today (the steel
// thread); other migrations previously lived here as detect-only "advise"
// packs and were removed to keep this tool scoped to one working path.
// They may return later — see git history for the removed rules.

// A Pack pairs a rule with its declared signals (expected webhook events,
// package version floors). Adding a migration means adding an entry to
// Packs — the verbs never change.
//
// Rule is empty for dpm: there is no local copy of that rule anymore, only
// a live fetch (judge.FetchDPMRule) from the doctor_rules API — see
// apirule.go. A future topic with its own static Rule would populate this
// field normally.
type Pack struct {
	Rule    Rule
	Signals *PackSignals
}

// Packs is the topic registry: every migration doctor knows about, keyed by
// the topic name used on the command line (`stripe doctor <topic> <dir>`).
var Packs = map[string]Pack{
	"dpm": {Signals: &DpmSignals},
}
