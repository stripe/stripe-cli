package judge

// Verdicts: a finding's intent alone -> a judgment. doctor no longer
// probes live account facts (API versions, Dashboard config) to reach a
// verdict, and no remediation-level guidance is surfaced here either —
// see pkg/doctor/remediation/paymentmethods, which owns that itself.

import (
	"regexp"
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
	Intent  string `json:"intent"`
	Verdict string `json:"verdict"`
	Class   string `json:"verdict_class"` // CANDIDATE | SKIP | REVIEW | ADVISE
}

var identRe = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

// ClassifyIntent inspects a finding's value expression and buckets it:
//
//	default-shaped — static list of only card/link: the classic pre-DPM
//	                 hardcode and the migration's actual target
//	deliberate     — static single non-card method (['oxxo']): per-method
//	                 integration, almost certainly intentional
//	static         — static multi-method list beyond card/link

// dynamic        — computed at runtime (identifiers, conditionals)
func ClassifyIntent(value string) string {
	v := strings.TrimSpace(value)
	quoted := regexp.MustCompile(`["']([a-z0-9_]+)["']`).FindAllStringSubmatch(v, -1)
	stripped := regexp.MustCompile(`["'][a-z0-9_]*["']|stripe\.StringSlice|\[\]string|Arrays\.asList|List\.of|[\[\]{},()\s]|new|List|string`).ReplaceAllString(v, "")
	if identRe.MatchString(stripped) || strings.ContainsAny(v, "?:") {
		return "dynamic"
	}
	if len(quoted) == 0 {
		return "dynamic"
	}
	onlyCardLink := true
	for _, q := range quoted {
		if q[1] != "card" && q[1] != "link" {
			onlyCardLink = false
		}
	}
	if onlyCardLink {
		return "default-shaped"
	}
	if len(quoted) == 1 {
		return "deliberate"
	}
	return "static"
}

// verdict turns a finding's intent alone into a judgment.
func verdict(intent string) string {
	switch intent {
	case "default-shaped":
		return "CANDIDATE: hardcoded to card/link only — removing this lets the Dashboard manage payment methods instead (confirm the frontend uses Payment Element)"
	case "deliberate":
		return "SKIP: restricted to a single payment method, which looks intentional — consider excluded_payment_method_types instead"
	case "dynamic":
		return "REVIEW: this value is computed at runtime — understand the routing logic before changing it"
	default:
		return "REVIEW: hardcoded list of multiple payment methods — confirm whether the restriction is still wanted"
	}
}

// Evaluate judges every finding against the rule alone — no account access,
// no credentials, no network. The rule decides the judgment style: a
// "remove" rule gets a CANDIDATE/SKIP/REVIEW verdict per finding, driven by
// the finding's intent; an "advise" rule gets a single ADVISE verdict per
// finding, carrying the rule's own remediation message.
func Evaluate(rule catalog.Rule, findings []catalog.Finding) *Report {
	r := &Report{Summary: map[string]int{}}
	for _, f := range findings {
		intent := ClassifyIntent(f.Value)
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
			v = verdict(intent)
		}
		class := v
		if i := strings.Index(v, ":"); i > 0 {
			class = v[:i]
		}
		r.Findings = append(r.Findings, Finding{Finding: f, Intent: intent, Verdict: v, Class: class})
		r.Summary[class]++
	}
	return r
}
