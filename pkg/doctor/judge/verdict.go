// Package judge produces doctor's topic-agnostic judgment for one rule's
// findings. A "remove" rule's findings get a generic REVIEW verdict here
// — judge has no domain knowledge of what any particular parameter's
// values mean, only that the rule wants it removed. A registered
// remediation (see pkg/doctor/remediation) can override any specific
// finding's verdict with real topic-specific judgment — see
// applyFindingVerdicts in cmd.go. An "advise" rule's findings get the
// rule's own ADVISE message, which is already topic-agnostic (it's just
// whatever the API says).
package judge

import (
	"strings"

	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
)

// Report is one topic's judgment: every finding plus a tally by class.
type Report struct {
	Findings []Finding
	Summary  map[string]int // verdict class -> count
}

// Finding is one scanned Finding plus the judgment reached about it.
type Finding struct {
	catalog.Finding
	Verdict string `json:"verdict"`
	Class   string `json:"verdict_class"` // REVIEW | ADVISE, or whatever class a remediation override names
}

// Evaluate judges every finding against the rule alone — no account
// access, no credentials, no network, and no topic-specific domain
// knowledge. A "remove" rule gets a generic REVIEW verdict per finding;
// an "advise" rule gets a single ADVISE verdict per finding, carrying
// the rule's own remediation message. Callers (cmd.go) layer a
// registered remediation's own per-finding judgment on top of this via
// Result.FindingVerdicts, when one exists.
func Evaluate(rule catalog.Rule, findings []catalog.Finding) *Report {
	r := &Report{Summary: map[string]int{}}
	for _, f := range findings {
		var v string
		if rule.Action == "advise" {
			v = "ADVISE: " + rule.Message
			if rule.ActionText != "" {
				v += " " + rule.ActionText
			}
			if rule.IntroducedIn != "" {
				v += " (API " + rule.IntroducedIn + ")"
			}
		} else {
			v = "REVIEW: confirm whether removing " + f.Param + " here is safe"
		}
		class := v
		if i := strings.Index(v, ":"); i > 0 {
			class = v[:i]
		}
		r.Findings = append(r.Findings, Finding{Finding: f, Verdict: v, Class: class})
		r.Summary[class]++
	}
	return r
}
