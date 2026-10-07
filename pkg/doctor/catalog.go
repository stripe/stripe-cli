package doctor

// catalog.go — packs expressed in the shape a future `stripe doctor` API
// would serve them in (topic/id/severity/checkFrom/checkUntil/docs/message/
// action, a discriminated "check", and sibling "signals"), parsed from
// embedded JSON today and converted into the engine's Rule/PackSignals.
// Swapping the embed for an HTTP fetch is the only change this format
// needs later.
//
// The catalog declares fixability per match, not per rule: each match's
// "fix" says what happens to that resource's sites on `fix --apply` —
// "removeParam" for a bare deletion (dpm's Checkout Sessions, Payment
// Links, Subscriptions, and Invoices matches: removing the param reverts
// to Dashboard-managed payment methods, nothing needs to replace it) or
// "replaceParam" for a deletion plus a companion insertion (dpm's
// payment_intents/setup_intents matches: automatic_payment_methods takes
// its place). The engine, however, only has one rule-level Action/Companion
// pair (fix.go, companion.go), and no way to scope removal to a subset of
// matches — so converting requires every match in the check to declare a
// fix kind before the rule is treated as "remove"-fixable at all. A match
// with no "fix" means the migration isn't fully modeled yet, and the whole
// rule falls back to "advise": better to detect-and-explain everything than
// to silently delete sites nobody has reviewed for safety.

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed catalog/dpm.json
var dpmCatalogJSON []byte

type catalogEntry struct {
	Topic      string          `json:"topic"`
	ID         string          `json:"id"`
	Severity   string          `json:"severity"`
	CheckFrom  string          `json:"checkFrom"`
	CheckUntil *string         `json:"checkUntil"` // reserved: version-ranged packs, not yet consumed
	Docs       string          `json:"docs"`
	Message    string          `json:"message"`
	Action     string          `json:"action"`
	Check      catalogCheck    `json:"check"`
	Signals    *catalogSignals `json:"signals"`
}

type catalogCheck struct {
	Kind  string         `json:"kind"`
	Match []catalogMatch `json:"match"`
}

type catalogMatch struct {
	Param      string      `json:"param"`
	Operations []string    `json:"operations"`
	Fix        *catalogFix `json:"fix"`
}

type catalogFix struct {
	Kind string         `json:"kind"`
	Add  *catalogFixAdd `json:"add"`
}

type catalogFixAdd struct {
	Param string `json:"param"`
}

type catalogSignals struct {
	WebhookEvents  []string               `json:"webhookEvents"`
	FrontendTokens []catalogFrontendToken `json:"frontendTokens"`
	ManifestFloors []catalogManifestFloor `json:"manifestFloors"`
}

type catalogFrontendToken struct {
	Token string `json:"token"`
	Note  string `json:"note"`
}

type catalogManifestFloor struct {
	Ecosystem string `json:"ecosystem"`
	Package   string `json:"package"`
	Min       string `json:"min"`
}

// loadCatalogPack parses one catalog entry and converts it into a Rule plus
// its PackSignals. Only "paramUsage" is implemented — the other kinds this
// format anticipates (runtimeFloor, apiVersionFloor, paramRename, ...) have
// no packs yet.
func loadCatalogPack(raw []byte) (Rule, PackSignals) {
	var e catalogEntry
	if err := json.Unmarshal(raw, &e); err != nil {
		panic(fmt.Sprintf("doctor: invalid catalog entry: %v", err))
	}
	if e.Check.Kind != "paramUsage" {
		panic(fmt.Sprintf("doctor: catalog entry %q: unsupported check kind %q", e.ID, e.Check.Kind))
	}

	rule := Rule{
		ID:           e.ID,
		Severity:     e.Severity,
		IntroducedIn: e.CheckFrom,
		Message:      strings.TrimSpace(e.Message + " " + e.Action),
		Docs:         e.Docs,
	}
	// allFixable only stays true if every match declares a fix kind — one
	// bare match is enough to drop the whole rule to "advise", since the
	// engine can't scope removal to a subset of matches.
	allFixable := len(e.Check.Match) > 0
	for _, m := range e.Check.Match {
		rule.Match = append(rule.Match, ParamMatch{Param: m.Param, Operations: m.Operations})
		if m.Fix == nil {
			allFixable = false
			continue
		}
		switch m.Fix.Kind {
		case "removeParam":
			// Bare deletion; no companion.
		case "replaceParam":
			rule.Companion = &Companion{
				Param:     m.Fix.Add.Param,
				ForParam:  m.Param,
				Resources: dedupResources(m.Operations),
			}
		default:
			panic(fmt.Sprintf("doctor: catalog entry %q: unsupported fix kind %q", e.ID, m.Fix.Kind))
		}
	}
	if allFixable {
		rule.Action = "remove"
	} else {
		rule.Action = "advise"
	}

	var sig PackSignals
	if e.Signals != nil {
		sig.WebhookEvents = e.Signals.WebhookEvents
		for _, t := range e.Signals.FrontendTokens {
			sig.FrontendTokens = append(sig.FrontendTokens, FrontendToken(t))
		}
		for _, f := range e.Signals.ManifestFloors {
			sig.ManifestFloors = append(sig.ManifestFloors, ManifestFloor{Package: f.Package, Min: f.Min})
		}
	}
	return rule, sig
}

// dedupResources extracts each operation's resource (the first non-templated
// path segment after /v1/) in first-seen order. This is deliberately
// distinct from scan.go's resourceFromOperation, which takes the LAST
// segment to build its SDK-token prefilter set and would misread
// ".../{intent}/confirm" as resource "confirm" instead of "payment_intents".
func dedupResources(operations []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, op := range operations {
		r := firstResourceSegment(op)
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out
}

func firstResourceSegment(op string) string {
	parts := strings.Fields(op)
	if len(parts) != 2 {
		return ""
	}
	segs := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
	for _, s := range segs[1:] {
		if !strings.HasPrefix(s, "{") {
			return s
		}
	}
	return ""
}

var dpmRule, dpmSignals = loadCatalogPack(dpmCatalogJSON)
