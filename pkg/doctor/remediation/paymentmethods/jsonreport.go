package paymentmethods

// jsonreport.go builds this topic's own --json contract
// (remediation.Result.Report) — a structured, versioned report keyed by
// "check": "dynamic_payment_methods", entirely independent of doctor's
// generic DoctorReport JSON shape. doctor's cmd.go treats Report
// opaquely: when a remediation sets one, it's emitted as the whole
// --json body instead of the generic schema. Every field name and
// finding id below is this topic's own vocabulary, not doctor's.

import (
	"path/filepath"

	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
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
	jsonActionRequired = "action_required"
	jsonBlocked        = "blocked"
	jsonReady          = "ready"
	jsonUnknown        = "unknown"
)

type jsonReport struct {
	SchemaVersion string        `json:"schema_version"`
	Check         string        `json:"check"`
	Status        string        `json:"status"`
	Summary       string        `json:"summary"`
	Findings      []jsonFinding `json:"findings"`
	Dashboard     jsonDashboard `json:"dashboard"`
	Documentation []jsonDoc     `json:"documentation"`
}

type jsonFinding struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	Evidence          any    `json:"evidence"`
	Implication       string `json:"implication"`
	RecommendedAction any    `json:"recommended_action"`
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

type suggestedParameter struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type pmListAction struct {
	Description              string              `json:"description"`
	SuggestedParameter       *suggestedParameter `json:"suggested_parameter,omitempty"`
	RequiresVerification     []string            `json:"requires_verification,omitempty"`
	SafeToApplyWithoutReview bool                `json:"safe_to_apply_without_review"`
}

// ---------- finding: runtime_payment_method_routing ----------

type routingEvidence struct {
	Observation string `json:"observation"`
	Locations   any    `json:"locations"`
}

type simpleAction struct {
	Description              string `json:"description"`
	SafeToApplyWithoutReview bool   `json:"safe_to_apply_without_review"`
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

// buildJSONReport assembles this topic's full --json contract from the
// same evidence assess/findingVerdicts already gathered. hasDynamic
// mirrors assess's own classification of ctx.Findings — recomputed here
// rather than threaded through, since it's a one-pass, side-effect-free
// scan over data Run already has in hand.
func buildJSONReport(ctx remediation.Context, vf versionFacts, pmf pmConfigFacts, wh *codesignals.HandlerSignals) *jsonReport {
	hasDynamic := false
	for _, f := range ctx.Findings {
		if classifyIntent(f.Value) == "dynamic" {
			hasDynamic = true
		}
	}

	var findings []jsonFinding
	if len(ctx.Findings) > 0 {
		findings = append(findings, pmListsFinding(ctx, vf))
	}
	if hasDynamic {
		findings = append(findings, routingFinding())
	}
	if wh != nil {
		findings = append(findings, webhooksFinding(wh))
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

func pmListsFinding(ctx remediation.Context, vf versionFacts) jsonFinding {
	locations := make([]pmListLocation, 0, len(ctx.Findings))
	for _, f := range ctx.Findings {
		locations = append(locations, pmListLocation{
			Path:             relLocationPath(ctx.Dir, f),
			Line:             f.Line,
			Parameter:        f.Param,
			SourceExpression: f.Value,
		})
	}

	gate := findingsVersionGate(vf)
	obs := apiVersionObservation{
		Scope:                         "recent_account_traffic",
		VersionsForTheseCodeLocations: "not_verified",
	}
	implication := "Dashboard-enabled payment methods will not appear in these flows until the explicit list is removed."
	if gate.Class != "" && !vf.VersionsMixed && !vf.NoEvents {
		obs.AllObservedVersionsBefore = dpmCutoff
		implication = "Removing payment_method_types alone does not enable automatic payment methods on the observed API versions."
	}

	return jsonFinding{
		ID:          "explicit_payment_method_lists",
		Status:      jsonReviewRequired,
		Evidence:    pmListEvidence{Locations: locations, APIVersionObservation: obs},
		Implication: implication,
		RecommendedAction: pmListAction{
			Description: "Replace explicit payment method lists with automatic payment methods where appropriate.",
			SuggestedParameter: &suggestedParameter{
				Name:  "automatic_payment_methods[enabled]",
				Value: true,
			},
			RequiresVerification: []string{
				"Confirm the API endpoint and API version used by each flow.",
				"Confirm the automatic payment methods parameter is supported by that endpoint and version.",
				"Confirm explicit restrictions are not required by the flow.",
			},
			SafeToApplyWithoutReview: false,
		},
	}
}

func routingFinding() jsonFinding {
	return jsonFinding{
		ID:          "runtime_payment_method_routing",
		Status:      jsonReviewRequired,
		Evidence:    routingEvidence{Observation: "Some payment_method_types values are computed at runtime.", Locations: nil},
		Implication: "Switching to Dashboard-managed payment methods may change which methods are offered in individual flows.",
		RecommendedAction: simpleAction{
			Description:              "Inspect runtime routing and preserve any required payment method restrictions before enabling additional methods.",
			SafeToApplyWithoutReview: false,
		},
	}
}

func webhooksFinding(wh *codesignals.HandlerSignals) jsonFinding {
	handlers := make([]webhookHandler, 0, len(wh.Events))
	for _, e := range wh.Events {
		detection := "not_found"
		if e.Present {
			detection = "found"
		}
		m := webhookMeanings[e.Event]
		handlers = append(handlers, webhookHandler{
			Event:            e.Event,
			Detection:        detection,
			Meaning:          m.Meaning,
			ExpectedBehavior: m.ExpectedBehavior,
		})
	}
	return jsonFinding{
		ID:          "delayed_payment_webhooks",
		Status:      jsonReviewRequired,
		Evidence:    webhooksEvidence{Scope: "scanned_code", Handlers: handlers},
		Implication: "The scan could not verify that delayed payment outcomes are handled.",
		RecommendedAction: simpleAction{
			Description:              "Look for equivalent handling outside the scanned code before adding handlers. If delayed methods are supported, implement any missing behavior.",
			SafeToApplyWithoutReview: false,
		},
	}
}

// relLocationPath is relLocation's path half, for this schema's
// path/line split (rather than one combined "path:line" string).
func relLocationPath(dir string, f catalog.Finding) string {
	rel, err := filepath.Rel(dir, f.File)
	if err != nil {
		return f.File
	}
	return rel
}
