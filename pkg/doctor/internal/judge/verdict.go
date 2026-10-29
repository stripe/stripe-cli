// Package judge gives every scanned finding a generic, topic-agnostic
// verdict: "remove" rules get a generic REVIEW; "advise" rules get the
// rule's own ADVISE message. A registered remediation can override any
// finding's verdict with real judgment (see Result.FindingVerdicts).
package judge

import (
	"strings"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
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

// Evaluate judges every finding against the rule alone, with no
// topic-specific domain knowledge.
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
