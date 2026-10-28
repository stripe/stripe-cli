package paymentmethods

// jsonreport.go builds this topic's own --json contract
// (remediation.Result.Report) — a structured, versioned report keyed by
// "check": "dynamic_payment_methods", entirely independent of doctor's
// generic DoctorReport JSON shape. doctor's cmd.go treats Report
// opaquely: when a remediation sets one, it's emitted as the whole
// --json body instead of the generic schema. Every field name and
// finding id below is this topic's own vocabulary, not doctor's.
//
// The "findings" array itself is built by calling remediation.FindingsJSON
// over the same result.Findings dynamic.go already built for the text
// action-plan renderer — see that function's doc comment for why that
// matters: the structured facts (locations, webhook handlers, etc.)
// only get written once. The one exception is
// "runtime_payment_method_routing": dynamic.go's corresponding Finding
// (dashboardFinding) is deliberately prose tied to Result.Status, not a
// discrete reportable fact, so its own --json entry is still
// hand-authored here and spliced into the same slot.

import (
	"github.com/stripe/stripe-cli/pkg/doctor/codesignals"
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

// ---------- finding: explicit_payment_method_lists ----------

type pmListLocation struct {
	Path             string `json:"path"`
	Line             int    `json:"line"`
	Parameter        string `json:"parameter"`
	SourceExpression string `json:"source_expression"`
}

type apiVersionObservation struct {
	Scope                         string `json:"scope"`
	AllObservedVersionsBefore     string `json:"all_observed_versions_before,omitempty"`
	VersionsForTheseCodeLocations string `json:"versions_for_these_code_locations"`
}

type pmListEvidence struct {
	Locations             []pmListLocation      `json:"locations"`
	APIVersionObservation apiVersionObservation `json:"api_version_observation"`
}

// ---------- finding: runtime_payment_method_routing ----------
//
// Hand-authored (see package doc comment) rather than built from a
// Finding — dynamic.go's dashboardFinding is prose tied to Result.Status,
// not a discrete fact this schema's generic conversion fits.

type routingEvidence struct {
	Observation string `json:"observation"`
	Locations   any    `json:"locations"`
}

type simpleAction struct {
	Description              string `json:"description"`
	SafeToApplyWithoutReview bool   `json:"safe_to_apply_without_review"`
}

func routingFindingJSON() remediation.FindingJSON {
	return remediation.FindingJSON{
		ID:          "runtime_payment_method_routing",
		Status:      remediation.JSONStatus(remediation.StatusReview),
		Evidence:    routingEvidence{Observation: "Some payment_method_types values are computed at runtime.", Locations: nil},
		Implication: "Switching to Dashboard-managed payment methods may change which methods are offered in individual flows.",
		RecommendedAction: simpleAction{
			Description:              "Inspect runtime routing and preserve any required payment method restrictions before enabling additional methods.",
			SafeToApplyWithoutReview: false,
		},
	}
}

// ---------- finding: delayed_payment_webhooks ----------

type webhookHandler struct {
	Event            string `json:"event"`
	Detection        string `json:"detection"`
	Meaning          string `json:"meaning"`
	ExpectedBehavior string `json:"expected_behavior"`
}

type webhooksEvidence struct {
	Scope    string           `json:"scope"`
	Handlers []webhookHandler `json:"handlers"`
}

// webhookMeanings splits webhookSignals' combined prose (used by the
// text action-plan renderer) into the separate meaning/expected_behavior
// fields this JSON schema wants — distinct wording for a distinct
// audience, not derived from one another.
var webhookMeanings = map[string]struct{ Meaning, ExpectedBehavior string }{
	"checkout.session.completed": {
		Meaning:          "Checkout completed; a delayed payment may still be pending.",
		ExpectedBehavior: "Check payment status before fulfilling the order.",
	},
	"checkout.session.async_payment_succeeded": {
		Meaning:          "A delayed payment succeeded.",
		ExpectedBehavior: "Fulfill the order if it has not already been fulfilled.",
	},
	"checkout.session.async_payment_failed": {
		Meaning:          "A delayed payment failed.",
		ExpectedBehavior: "Update the order and notify the customer as appropriate.",
	},
}

// buildJSONReport assembles this topic's full --json contract, built
// around result.Findings — the same Findings dynamic.go already
// constructed for the text action-plan renderer. hasDynamic mirrors
// assess's own classification of ctx.Findings — recomputed here rather
// than threaded through, since it's a one-pass, side-effect-free scan
// over data Run already has in hand.
func buildJSONReport(ctx remediation.Context, vf versionFacts, pmf pmConfigFacts, wh *codesignals.HandlerSignals, result remediation.Result) *jsonReport {
	hasDynamic := false
	for _, f := range ctx.Findings {
		if classifyIntent(f.Value) == "dynamic" {
			hasDynamic = true
		}
	}

	// dashboardFinding (ID == "") sits in result.Findings right where
	// "runtime_payment_method_routing" belongs in this schema's findings
	// array — splice it in there instead of appending it, so the order
	// matches regardless of how many other findings surround it.
	var findings []remediation.FindingJSON
	for _, f := range result.Findings {
		if f.ID != "" {
			findings = append(findings, remediation.FindingsJSON([]remediation.Finding{f}, result.Status)[0])
			continue
		}
		if hasDynamic {
			findings = append(findings, routingFindingJSON())
		}
	}

	status := jsonReady
	if !pmf.ConfiguredOK {
		status = jsonBlocked
	} else if len(findings) > 0 {
		status = jsonReviewRequired
	}

	var enabledMethods any
	if pmf.ConfiguredOK && !hasDynamic {
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
			RecommendedAction:     "After verifying routing and payment handling, review which payment methods to enable.",
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
