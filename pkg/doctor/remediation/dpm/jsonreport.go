package dpm

// jsonreport.go builds this topic's own --json envelope
// (remediation.Result.Report): schema_version/check/status/summary/
// findings/dashboard/documentation, keyed by "check":
// "dynamic_payment_methods" — entirely independent of doctor's generic
// DoctorReport JSON shape. doctor's cmd.go treats Report opaquely: when
// a remediation sets one, it's emitted as the whole --json body instead
// of the generic schema.
//
// The "findings" array is built by calling remediation.FindingsJSON over
// result.Findings — the same Findings findings.go already built for the
// text action-plan renderer, so those facts are only written once.
// dashboardFinding (ID == "") is skipped: it's prose tied to
// Result.Status, not a discrete reportable fact — this schema's
// "dashboard" object below covers that ground instead.

import (
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

const schemaVersion = "1.0"
const checkID = "dynamic_payment_methods"

// Status strings for this schema — distinct from remediation.Status:
// every finding this topic can currently raise needs a human decision
// before anything is applied, so "review_required" covers nearly all of
// them; "blocked"/"ready" are for the Dashboard-configuration states
// that genuinely are blocking or clear.
const (
	jsonReviewRequired = "review_required"
	jsonBlocked        = "blocked"
	jsonReady          = "ready"
	jsonUnknown        = "unknown"
)

type jsonReport struct {
	SchemaVersion string                    `json:"schema_version"`
	Check         string                    `json:"check"`
	Status        string                    `json:"status"`
	Summary       string                    `json:"summary"`
	Findings      []remediation.FindingJSON `json:"findings"`
	Dashboard     jsonDashboard             `json:"dashboard"`
	Documentation []jsonDoc                 `json:"documentation"`
}

type jsonDashboard struct {
	EnabledPaymentMethods any    `json:"enabled_payment_methods"`
	RecommendedAction     string `json:"recommended_action"`
	URL                   string `json:"url"`
}

type jsonDoc struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// buildJSONReport assembles this topic's full --json contract, built
// around result.Findings — the same Findings findings.go already
// constructed for the text action-plan renderer.
func buildJSONReport(ctx remediation.Context, pmf pmConfigFacts, result remediation.Result) *jsonReport {
	var findings []remediation.FindingJSON
	for _, f := range result.Findings {
		if f.ID == "" {
			continue
		}
		findings = append(findings, remediation.FindingsJSON([]remediation.Finding{f}, result.Status)[0])
	}

	status := jsonReady
	if !pmf.ConfiguredOK {
		status = jsonBlocked
	} else if len(findings) > 0 {
		status = jsonReviewRequired
	}

	var enabledMethods any
	if pmf.ConfiguredOK {
		enabledMethods = pmf.EnabledMethods
	}

	return &jsonReport{
		SchemaVersion: schemaVersion,
		Check:         checkID,
		Status:        status,
		Summary:       jsonSummary(ctx, status),
		Findings:      findings,
		Dashboard: jsonDashboard{
			EnabledPaymentMethods: enabledMethods,
			RecommendedAction:     "Review which payment methods to enable in the Dashboard.",
			URL:                   dashboardConfigURL,
		},
		Documentation: []jsonDoc{{Title: title, URL: ctx.Rule.Docs}},
	}
}

func jsonSummary(ctx remediation.Context, status string) string {
	if len(ctx.Findings) == 0 {
		return "No explicit payment method lists were found in the scanned code."
	}
	if status == jsonBlocked {
		return "Explicit payment method lists were found, and no payment methods are currently enabled in the Dashboard."
	}
	return "Explicit payment method lists prevent Dashboard-enabled methods from appearing automatically in these checkout flows."
}
