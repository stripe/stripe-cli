package catalog

// apirule.go parses one rule object as returned by the live doctor_rules
// API (GET /v2/integ_hlth/doctor_rules or .../doctor_rules/{topic}) into
// the engine's Rule. There is no local copy of the rule itself — see the
// judge package (judge.FetchRule), which fetches it live and is the
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
//     API version) contributes no ParamMatch — doctor's engine doesn't
//     probe live account facts to re-derive that condition itself — but its
//     cutoff version (Rule.IntroducedIn) is still read.
//
// There is no "fixes" field here: what a remediation actually does (which
// params to touch, what guidance to surface) is no longer data on the
// rule at all — it's defined directly in that topic's own Go
// implementation. See pkg/doctor/remediation/paymentmethods for dpm's
// hardcoded equivalent. Action is therefore always "remove": every topic
// doctor knows today is a remove-this-parameter migration, judged per
// finding by intent alone (see the judge package) — a future detect-only
// topic would need this reconsidered, not derived from fixes data that no
// longer exists.
//
// "doctorTopic" is the only remediation-related field left: a bare
// string (e.g. "dpm") naming which pkg/doctor/remediation.Implementation
// to run — see catalog.Rule.RemediationTopic. There's no separate
// per-remediation "kind"/Options data; a topic's implementation is
// expected to know everything about its own remediation already.
//
// The API has no "signals" field (webhook events, etc.) and this package
// doesn't model one: a remediation that cares about code signals declares
// and scans for its own — see pkg/doctor/remediation/paymentmethods.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

type apiRule struct {
	Topic       string         `json:"topic"`
	Severity    string         `json:"severity"`
	Docs        []string       `json:"docs"`
	Message     string         `json:"message"`
	Action      string         `json:"action"`
	DoctorTopic string         `json:"doctorTopic"`
	Logic       []apiLogicNode `json:"logic"`
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
		return Rule{}, errorcategory.Errorf(errorcategory.API, "doctor: doctor_rule %q: expected exactly one top-level \"or\" condition group", e.Topic)
	}

	rule := Rule{
		ID:               e.Topic,
		Severity:         e.Severity,
		Action:           "remove",
		Message:          strings.TrimSpace(e.Message),
		ActionText:       strings.TrimSpace(e.Action),
		Docs:             strings.Join(e.Docs, ", "),
		RemediationTopic: strings.TrimSpace(e.DoctorTopic),
	}

	for _, node := range e.Logic[0].ConditionGroup.Items {
		if node.MatcherType != "entry" || node.Entry == nil {
			return Rule{}, errorcategory.Errorf(errorcategory.API, "doctor: doctor_rule %q: expected an \"entry\" node under the top-level group", e.Topic)
		}
		entry := node.Entry

		if len(entry.PostParams) != 1 || entry.PostParams[0].MatcherType != "entry" {
			// Not a param-usage match (e.g. the STRIPE_VERSION-gated
			// companion condition) — contributes no ParamMatch; only its
			// cutoff version is relevant.
			if v := versionCutoff(entry); v != "" {
				rule.IntroducedIn = v
			}
			continue
		}

		cond := entry.PostParams[0]
		if cond.Matcher == nil || cond.Matcher.Operator != "exist" {
			return Rule{}, errorcategory.Errorf(errorcategory.API, "doctor: doctor_rule %q: unsupported post_params matcher on %q", e.Topic, cond.Key)
		}

		rule.Match = append(rule.Match, ParamMatch{
			Param:      cond.Key,
			Operations: []string{entry.Method + " " + entry.Path.Value},
		})
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
