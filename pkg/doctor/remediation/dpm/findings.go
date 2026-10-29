package dpm

import (
	"fmt"
	"path/filepath"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/codesignals"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

// relPath formats a finding's file path relative to the scanned
// directory, for display. Falls back to the absolute path if it isn't
// under dir.
func relPath(dir string, f catalog.Finding) string {
	rel, err := filepath.Rel(dir, f.File)
	if err != nil {
		return f.File
	}
	return rel
}

// ---------- finding: explicit_payment_method_lists ----------

type pmListLocation struct {
	Path             string `json:"path"`
	Line             int    `json:"line"`
	Parameter        string `json:"parameter"`
	SourceExpression string `json:"source_expression"`
	// RuntimeComputed is classifyIntent's "dynamic" bucket, flagged per
	// location.
	RuntimeComputed bool `json:"runtime_computed"`
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

// hardcodedListsFinding builds the action item for the scanned code
// locations: a shared reason/instruction when the account's API version
// dictates the same fix for every finding, else each finding judged on
// its own code shape.
func hardcodedListsFinding(ctx remediation.Context, vf versionFacts) remediation.Finding {
	f := remediation.Finding{ID: "explicit_payment_method_lists", Heading: "Replace explicit lists with automatic payment methods"}

	fix := findingsVersionFix(vf)
	if fix.Class == "" {
		f.Body = []string{"Each flagged location below needs its own review before removing payment_method_types."}
		f.Implication = "Dashboard-enabled payment methods will not appear in these flows until the explicit list is removed."
	} else {
		f.Body = []string{fix.Reason, fix.Instruction}
		f.Implication = "Removing payment_method_types alone does not enable automatic payment methods on the observed API versions."
		f.ActionDescription = "Replace explicit payment method lists with automatic payment methods where appropriate."
		f.SuggestedParameter = &remediation.Param{Name: "automatic_payment_methods[enabled]", Value: true}
		f.RequiresVerification = []string{
			"Confirm the API endpoint and API version used by each flow.",
			"Confirm the automatic payment methods parameter is supported by that endpoint and version.",
			"Confirm explicit restrictions are not required by the flow.",
		}
	}

	hasDynamic := false
	locations := make([]pmListLocation, 0, len(ctx.Findings))
	for _, finding := range ctx.Findings {
		intent := classifyIntent(finding.Value)
		if intent == "dynamic" {
			hasDynamic = true
		}
		loc := pmListLocation{
			Path:             relPath(ctx.Dir, finding),
			Line:             finding.Line,
			Parameter:        finding.Param,
			SourceExpression: finding.Value,
			RuntimeComputed:  intent == "dynamic",
		}
		locations = append(locations, loc)
		var body string
		switch {
		case fix.Class == "":
			body = intentVerdict(intent)
		case intent == "dynamic":
			body = "Current list: " + loc.SourceExpression + " — this value is computed at runtime; its routing logic needs its own review."
		default:
			body = "Current list: " + loc.SourceExpression
		}
		f.Items = append(f.Items, remediation.Item{
			Heading: fmt.Sprintf("%s:%d", loc.Path, loc.Line),
			Body:    body,
			Fields:  loc,
		})
	}
	if hasDynamic {
		f.Body = append(f.Body, "Some payment method lists are built at runtime. Check whether that logic restricts payment methods for particular checkout flows. Preserve any required restrictions when switching to dynamic payment methods.")
	}
	obs := apiVersionObservation{Scope: "recent_account_traffic", VersionsForTheseCodeLocations: "not_verified"}
	if fix.Class != "" && !vf.VersionsMixed && !vf.NoEvents {
		obs.AllObservedVersionsBefore = dpmCutoff
	}
	f.Evidence = pmListEvidence{Locations: locations, APIVersionObservation: obs}
	return f
}

// ---------- finding: the standing Dashboard-config reminder ----------

// dashboardFinding is the action-plan item for whether the Dashboard's
// payment-method configuration matches what the code hardcodes. Left
// with no ID: it's prose tied to Result.Status, not an independently
// reportable fact (the --json contract covers this via its own
// top-level "dashboard" object instead).
func dashboardFinding(msg string) remediation.Finding {
	return remediation.Finding{
		Heading: "Verify Dashboard payment methods match your code",
		Body:    []string{msg, dashboardReviewLead + "\n" + dashboardConfigURL},
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

// webhookMeanings gives each event's meaning and expected handling,
// separately from webhookSignals' combined prose used by the text
// renderer.
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

// webhooksFinding is the action-plan item for webhook-handler coverage.
// Returns a zero-value Finding when wh is nil (no webhook signals
// scanned).
func webhooksFinding(wh *codesignals.HandlerSignals) remediation.Finding {
	if wh == nil {
		return remediation.Finding{}
	}
	f := remediation.Finding{
		ID:                "delayed_payment_webhooks",
		Heading:           "Verify handling for delayed payments",
		Implication:       "The scan could not verify that delayed payment outcomes are handled.",
		ActionDescription: "Look for equivalent handling outside the scanned code before adding handlers. If delayed methods are supported, implement any missing behavior.",
	}
	f.Body = append(f.Body, "Checkout can finish before a delayed payment succeeds. Make sure your integration distinguishes checkout completion from payment confirmation.")
	if wh.AllPresent {
		f.Body = append(f.Body, "All expected webhook handlers were found in the scanned code.")
	} else {
		f.Body = append(f.Body, "These handlers weren't found in the scanned code. If they're implemented elsewhere, verify them there.")
	}
	var handlers []webhookHandler
	for _, e := range wh.Events {
		detection, body := "not_found", e.Note
		if e.Present {
			detection = "found"
			body = "Found in scanned code"
			if len(e.Files) > 0 {
				body = "Found in " + e.Files[0]
			}
		}
		m := webhookMeanings[e.Event]
		h := webhookHandler{Event: e.Event, Detection: detection, Meaning: m.Meaning, ExpectedBehavior: m.ExpectedBehavior}
		handlers = append(handlers, h)
		f.Items = append(f.Items, remediation.Item{Heading: e.Event, Body: body, Fields: h})
	}
	f.Evidence = webhooksEvidence{Scope: "scanned_code", Handlers: handlers}
	return f
}

// webhookSummaryPhrase is the "Webhooks" summary-table row value: how
// many expected handlers were not found in the scanned code.
func webhookSummaryPhrase(wh *codesignals.HandlerSignals) string {
	if wh == nil {
		return "no webhook signals checked"
	}
	missing := 0
	for _, e := range wh.Events {
		if !e.Present {
			missing++
		}
	}
	if missing == 0 {
		return fmt.Sprintf("all %d expected handlers found in scanned code", len(wh.Events))
	}
	noun := "handlers"
	if missing == 1 {
		noun = "handler"
	}
	return fmt.Sprintf("%d expected %s not found in scanned code", missing, noun)
}
