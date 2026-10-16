package catalog

// catalog.go — packs expressed in the shape a future `stripe doctor` API
// would serve them in: topic/severity/docs/message/action, a set of named
// "fixes", a "logic" tree describing which requests each fix applies to, and
// sibling "signals". Parsed from embedded JSON today and converted into the
// engine's Rule/PackSignals; swapping the embed for an HTTP fetch is the only
// change this format needs later.
//
// "logic" is shaped like a request matcher (method/path/headers/body params)
// because that is how the future API describes a migration generically, but
// doctor's engine is a static source scanner, not a traffic inspector. Only
// the parts a static scan can act on are consumed:
//   - a "logic" item whose postParams is a single "entry" (param key, operator
//     "exist") becomes one ParamMatch — the Operations prefix (method + path)
//     is read only to classify the match, since the exact path shape doesn't
//     matter to a presence scanner.
//   - a "logic" item whose postParams is a "conditionGroup" (the dpm rule's
//     STRIPE_VERSION-gated item, describing when a bare request lacks BOTH
//     payment_method_types and automatic_payment_methods on a pre-cutoff API
//     version) contributes no ParamMatch at all: the engine already derives
//     that condition from the account's live API version (the judge package,
//     Rule.IntroducedIn), so this item is only read for its cutoff version.
//
// Fixes are declared per logic item via "doctorFixIds", not per rule, in the
// upstream catalog format. doctor's engine only cares whether a match-
// producing item declares at least one fix id at all — that marks the
// parameter as a reviewed "remove" target with per-account judgment (see the
// judge package) rather than a bare "advise" detection. An item with no
// doctorFixIds means the migration isn't fully modeled yet, and the whole
// rule falls back to "advise": better to detect-and-explain everything than
// to call a site safe nobody has reviewed. The specific fix kind
// ("removeParam" vs "upsertParam") only matters to a code-editing tool this
// package no longer has, so it's validated for catalog sanity and otherwise
// ignored.

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed dpm.json
var dpmCatalogJSON []byte

type catalogEntry struct {
	Topic    string              `json:"topic"`
	Severity string              `json:"severity"`
	Docs     []string            `json:"docs"`
	Message  string              `json:"message"`
	Action   string              `json:"action"`
	Fixes    []catalogFix        `json:"fixes"`
	Logic    []catalogLogicGroup `json:"logic"`
	Signals  *catalogSignals     `json:"signals"`
}

type catalogFix struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"` // "removeParam" | "upsertParam"
	Param string `json:"param"`
}

type catalogLogicGroup struct {
	LogicalOperator string             `json:"logicalOperator"`
	Items           []catalogLogicItem `json:"items"`
}

type catalogLogicItem struct {
	Path           catalogMatcher           `json:"path"`
	Method         string                   `json:"method"`
	RequestHeaders []catalogHeaderCondition `json:"requestHeaders"`
	PostParams     []catalogParamCondition  `json:"postParams"`
	DoctorFixIDs   []string                 `json:"doctorFixIds"`
}

type catalogMatcher struct {
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type catalogHeaderCondition struct {
	Key     string          `json:"key"`
	Matcher *catalogMatcher `json:"matcher"`
}

type catalogParamCondition struct {
	MatcherType string          `json:"matcherType"` // "entry" | "conditionGroup"
	Key         string          `json:"key"`
	Matcher     *catalogMatcher `json:"matcher"`
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
// its PackSignals. Only a single top-level "or" logic group is implemented,
// since that is the only shape any pack needs today.
func loadCatalogPack(raw []byte) (Rule, PackSignals) {
	var e catalogEntry
	if err := json.Unmarshal(raw, &e); err != nil {
		panic(fmt.Sprintf("doctor: invalid catalog entry: %v", err))
	}
	if len(e.Logic) != 1 || e.Logic[0].LogicalOperator != "or" {
		panic(fmt.Sprintf("doctor: catalog entry %q: expected exactly one top-level \"or\" logic group", e.Topic))
	}

	fixByID := make(map[string]catalogFix, len(e.Fixes))
	for _, f := range e.Fixes {
		fixByID[f.ID] = f
	}

	rule := Rule{
		ID:       e.Topic,
		Severity: e.Severity,
		Message:  strings.TrimSpace(e.Message + " " + e.Action),
		Docs:     strings.Join(e.Docs, ", "),
	}

	// allFixable only stays true if every match-producing item declares a fix
	// id — one bare item is enough to drop the whole rule to "advise", since
	// the engine can't scope the "remove" judgment to a subset of matches.
	allFixable := false
	for _, item := range e.Logic[0].Items {
		if len(item.PostParams) != 1 || item.PostParams[0].MatcherType != "entry" {
			// Not a param-usage match (e.g. the STRIPE_VERSION-gated
			// companion condition) — only its cutoff version is relevant.
			if v := versionCutoff(item); v != "" {
				rule.IntroducedIn = v
			}
			continue
		}

		cond := item.PostParams[0]
		if cond.Matcher == nil || cond.Matcher.Operator != "exist" {
			panic(fmt.Sprintf("doctor: catalog entry %q: unsupported postParams matcher on %q", e.Topic, cond.Key))
		}

		allFixable = true
		rule.Match = append(rule.Match, ParamMatch{
			Param:      cond.Key,
			Operations: []string{item.Method + " " + item.Path.Value},
		})

		if len(item.DoctorFixIDs) == 0 {
			allFixable = false
		}
		for _, fixID := range item.DoctorFixIDs {
			fix, ok := fixByID[fixID]
			if !ok {
				panic(fmt.Sprintf("doctor: catalog entry %q: unknown fix id %q", e.Topic, fixID))
			}
			switch fix.Kind {
			case "removeParam", "upsertParam":
				// Either kind marks the parameter as a reviewed "remove"
				// target; which kind it is only mattered to the code editor
				// this package no longer has.
			default:
				panic(fmt.Sprintf("doctor: catalog entry %q: unsupported fix kind %q", e.Topic, fix.Kind))
			}
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

// versionCutoff returns the API version an item's STRIPE_VERSION header
// matcher names, or "" if it has none.
func versionCutoff(item catalogLogicItem) string {
	for _, h := range item.RequestHeaders {
		if h.Key == "STRIPE_VERSION" && h.Matcher != nil && h.Matcher.Operator == "less_than" {
			return h.Matcher.Value
		}
	}
	return ""
}

// DpmRule and DpmSignals are converted from the embedded dpm.json entry.
var DpmRule, DpmSignals = loadCatalogPack(dpmCatalogJSON)
