package catalog

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

type apiRule struct {
	Topic    string         `json:"topic"`
	Severity string         `json:"severity"`
	Docs     []string       `json:"docs"`
	Message  string         `json:"message"`
	Action   string         `json:"action"`
	Logic    []apiLogicNode `json:"logic"`
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

// apiParamNode flattens the same entry/conditionGroup union onto one
// struct, matching how the API shapes post_params/request_headers.
type apiParamNode struct {
	MatcherType string      `json:"matcher_type"` // "entry" | "conditionGroup"
	Key         string      `json:"key"`
	Matcher     *apiMatcher `json:"matcher"`
}

// ParseAPIRule parses one doctor_rules API rule into a Rule. Only parts
// a static scanner can act on are kept: a postParams "exist" check
// becomes a ParamMatch; a STRIPE_VERSION conditionGroup contributes no
// ParamMatch but still sets Rule.IntroducedIn.
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
		RemediationTopic: strings.TrimSpace(e.Topic),
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
