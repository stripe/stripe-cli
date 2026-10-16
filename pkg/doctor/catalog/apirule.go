package catalog

// apirule.go parses one rule object as returned by the live doctor_rules
// API (GET /v2/integ_hlth/doctor_rules or .../doctor_rules/{topic}) into
// the engine's Rule. There is no local copy of the rule itself — see the
// judge package (judge.FetchDPMRule), which fetches it live and is the
// only caller of ParseAPIRule.
//
// "logic" is shaped like a request matcher (method/path/headers/body
// params) because that is how the API describes a migration generically,
// but doctor's engine is a static source scanner, not a traffic inspector.
// Only the parts a static scan can act on are consumed:
//   - a logic entry whose postParams is a single "entry" (param key,
//     operator "exist") becomes one ParamMatch — the path/method are read
//     only to classify the match, since the exact path shape doesn't
//     matter to a presence scanner.
//   - a logic entry whose postParams is a "conditionGroup" (the dpm rule's
//     STRIPE_VERSION-gated item, describing when a bare request lacks BOTH
//     payment_method_types and automatic_payment_methods on a pre-cutoff
//     API version) contributes no ParamMatch at all: the engine already
//     derives that condition from the account's live API version (the
//     judge package, Rule.IntroducedIn), so this item is only read for its
//     cutoff version.
//
// Fixes are declared per logic entry via "doctor_fix_ids", not per rule.
// doctor's engine only cares whether a match-producing entry declares at
// least one fix id at all — that marks the parameter as a reviewed
// "remove" target with per-account judgment (see the judge package) rather
// than a bare "advise" detection. An entry with no doctor_fix_ids means the
// migration isn't fully modeled yet, and the whole rule falls back to
// "advise": better to detect-and-explain everything than to call a site
// safe nobody has reviewed. The specific fix kind ("removeParam" vs
// "upsertParam") only matters to a code-editing tool this package no
// longer has, so it's validated for API-response sanity and otherwise
// ignored.
//
// The API has no "signals" field yet (webhook events, etc.) — see
// dpm_signals.go for that stopgap.

import (
	"encoding/json"
	"fmt"
	"strings"
)

type apiRule struct {
	Topic    string         `json:"topic"`
	Severity string         `json:"severity"`
	Docs     []string       `json:"docs"`
	Message  string         `json:"message"`
	Action   string         `json:"action"`
	Fixes    []apiFix       `json:"fixes"`
	Logic    []apiLogicNode `json:"logic"`
}

type apiFix struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"` // "removeParam" | "upsertParam"
	Param string `json:"param"`
}

// apiLogicNode is a discriminated union: MatcherType "entry" means Entry is
// populated; "conditionGroup" means ConditionGroup is populated. Used both
// for the top-level logic list and recursively inside ConditionGroup.Items.
type apiLogicNode struct {
	MatcherType    string             `json:"matcher_type"` // "entry" | "conditionGroup"
	Entry          *apiEntry          `json:"entry"`
	ConditionGroup *apiConditionGroup `json:"condition_group"`
}

type apiConditionGroup struct {
	LogicalOperator string         `json:"logical_operator"`
	Items           []apiLogicNode `json:"items"`
}

type apiEntry struct {
	Method         string         `json:"method"`
	Path           apiMatcher     `json:"path"`
	RequestHeaders []apiParamNode `json:"request_headers"`
	PostParams     []apiParamNode `json:"post_params"`
	DoctorFixIDs   []string       `json:"doctor_fix_ids"`
}

type apiMatcher struct {
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// apiParamNode is the same kind of discriminated union as apiLogicNode, but
// flattened onto one struct (as the API shapes post_params/request_headers
// entries): MatcherType "entry" uses Key/Matcher; "conditionGroup" uses
// Items/LogicalOperator. Nested conditionGroups are never read today — see
// the package doc comment — so Items/LogicalOperator are declared only so
// json.Unmarshal doesn't need them stripped, not because this package
// recurses into them.
type apiParamNode struct {
	MatcherType string      `json:"matcher_type"` // "entry" | "conditionGroup"
	Key         string      `json:"key"`
	Matcher     *apiMatcher `json:"matcher"`
}

// ParseAPIRule converts one rule object from the live doctor_rules API
// into a Rule.
func ParseAPIRule(raw []byte) (Rule, error) {
	var e apiRule
	if err := json.Unmarshal(raw, &e); err != nil {
		return Rule{}, fmt.Errorf("doctor: invalid doctor_rules response: %w", err)
	}
	if len(e.Logic) != 1 || e.Logic[0].MatcherType != "conditionGroup" || e.Logic[0].ConditionGroup == nil ||
		e.Logic[0].ConditionGroup.LogicalOperator != "or" {
		return Rule{}, fmt.Errorf("doctor: doctor_rule %q: expected exactly one top-level \"or\" condition group", e.Topic)
	}

	fixByID := make(map[string]apiFix, len(e.Fixes))
	for _, f := range e.Fixes {
		fixByID[f.ID] = f
	}

	rule := Rule{
		ID:       e.Topic,
		Severity: e.Severity,
		Message:  strings.TrimSpace(e.Message + " " + e.Action),
		Docs:     strings.Join(e.Docs, ", "),
	}

	// allFixable only stays true if every match-producing entry declares a
	// fix id — one bare entry is enough to drop the whole rule to "advise",
	// since the engine can't scope the "remove" judgment to a subset of
	// matches.
	allFixable := false
	for _, node := range e.Logic[0].ConditionGroup.Items {
		if node.MatcherType != "entry" || node.Entry == nil {
			return Rule{}, fmt.Errorf("doctor: doctor_rule %q: expected an \"entry\" node under the top-level group", e.Topic)
		}
		entry := node.Entry

		if len(entry.PostParams) != 1 || entry.PostParams[0].MatcherType != "entry" {
			// Not a param-usage match (e.g. the STRIPE_VERSION-gated
			// companion condition) — only its cutoff version is relevant.
			if v := versionCutoff(entry); v != "" {
				rule.IntroducedIn = v
			}
			continue
		}

		cond := entry.PostParams[0]
		if cond.Matcher == nil || cond.Matcher.Operator != "exist" {
			return Rule{}, fmt.Errorf("doctor: doctor_rule %q: unsupported post_params matcher on %q", e.Topic, cond.Key)
		}

		allFixable = true
		rule.Match = append(rule.Match, ParamMatch{
			Param:      cond.Key,
			Operations: []string{entry.Method + " " + entry.Path.Value},
		})

		if len(entry.DoctorFixIDs) == 0 {
			allFixable = false
		}
		for _, fixID := range entry.DoctorFixIDs {
			fix, ok := fixByID[fixID]
			if !ok {
				return Rule{}, fmt.Errorf("doctor: doctor_rule %q: unknown fix id %q", e.Topic, fixID)
			}
			switch fix.Kind {
			case "removeParam", "upsertParam":
				// Either kind marks the parameter as a reviewed "remove"
				// target; which kind it is only mattered to the code editor
				// this package no longer has.
			default:
				return Rule{}, fmt.Errorf("doctor: doctor_rule %q: unsupported fix kind %q", e.Topic, fix.Kind)
			}
		}
	}
	if allFixable {
		rule.Action = "remove"
	} else {
		rule.Action = "advise"
	}
	return rule, nil
}

// versionCutoff returns the API version an entry's STRIPE_VERSION header
// matcher names, or "" if it has none.
func versionCutoff(entry *apiEntry) string {
	for _, h := range entry.RequestHeaders {
		if h.Key == "STRIPE_VERSION" && h.Matcher != nil && h.Matcher.Operator == "less_than" {
			return h.Matcher.Value
		}
	}
	return ""
}
