// Package dpm implements the "dpm" topic's remediation:
// adopting Dynamic Payment Methods by removing a hardcoded
// payment_method_types list. This is the worked example other teams
// should read before writing their own topic's remediation — see
// pkg/doctor/remediation's package doc for the three steps to plug one in.
//
// This package owns every piece of dpm's own domain knowledge — none of
// it lives in doctor's generic engine:
//
//  1. Recent API versions (fetchVersionFacts) — removing
//     payment_method_types only works on its own at/after 2023-08-16.
//  2. The account's payment-method configuration (fetchPMConfigFacts) —
//     adopting dynamic methods only preserves what the Dashboard already
//     offers.
//  3. Per-finding judgment (classifyIntent/intentVerdict, the
//     API-version-driven fix) — doctor's judge package has no idea what
//     "card"/"link" or payment_method_types even mean; hardcodedListsFinding
//     (findings.go) attaches this topic's real judgment to each
//     location via remediation.Finding.Items instead.
//  4. Webhook handler coverage, both statically (webhookSignals, scanned
//     via pkg/doctor/codesignals) and live (WebhookCheckEvents, behind
//     `doctor --verify-webhooks` via pkg/doctor/webhookcheck) — this
//     topic declares its own expected events; doctor's engine wires in
//     neither generically.
//
// assess combines 1-2 and the webhook-handler evidence into the
// complete Result: its Status/Message/Headline are purely about
// Dashboard/webhook readiness, but it also attaches the per-finding
// judgment (3) via hardcodedListsFinding, since that still has to land
// somewhere in the same Result.Findings list. dashboardNotes is
// hardcoded here rather than read off the rule — there's no more
// per-remediation data on the rule at all (just the bare "doctorTopic"
// string that got us to this implementation in the first place); this
// topic is expected to know everything about its own remediation
// already.
//
// Split across:
//   - dynamic.go: registration, Run, and the Dashboard/webhook readiness
//     state machine (assess).
//   - facts.go: the two pieces of live account evidence Run fetches
//     (recent API versions, payment-method configuration).
//   - classify.go: per-finding judgment (classifyIntent/intentVerdict,
//     the API-version-driven fix).
//   - findings.go: building each remediation.Finding — text and, via
//     remediation.FindingsJSON, --json — from the facts above.
//   - jsonreport.go: the --json envelope (schema_version/check/status/
//     summary/dashboard/documentation) this topic's Report wraps
//     everything else in.
package dpm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stripe/stripe-cli/pkg/doctor/codesignals"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
	"github.com/stripe/stripe-cli/pkg/doctor/stripeapi"
)

// Topic is the catalog.Rule.RemediationTopic ("doctorTopic") this package
// implements.
const Topic = "dpm"

// title is this topic's plain-English name for doctor's action-plan
// renderer (remediation.Result.Title) — the renderer itself has no idea
// what "dpm" stands for.
const title = "Dynamic payment methods"

// webhookSignals are the delayed-notification events this topic expects a
// migrated integration to handle — this topic's own declaration, scanned
// for with pkg/doctor/codesignals rather than something doctor's engine
// wires in generically. A different topic can declare none of these,
// different ones, or none at all; that's entirely up to its own Run.
var webhookSignals = &codesignals.PackSignals{
	WebhookEvents: []codesignals.WebhookEvent{
		{Event: "checkout.session.completed", Note: "Checkout finished. If payment is still pending, wait for payment confirmation before fulfilling the order."},
		{Event: "checkout.session.async_payment_succeeded", Note: "A delayed payment succeeded. Fulfill the order if you haven't already done so."},
		{Event: "checkout.session.async_payment_failed", Note: "A delayed payment failed. Update the order and notify the customer as appropriate."},
	},
}

// dashboardNotes are this topic's own reminder to confirm the Dashboard's
// payment-method configuration directly — formerly sourced from the API's
// "fixes" field (a "message"-kind fix); that field is gone now, so this
// topic owns the text itself. Set on Result.Notes unconditionally, so it
// renders as its own separate Dashboard action rather than a trailing
// clause on whatever the Status message happens to be.
var dashboardNotes = []string{
	"Enable the payment methods you want to use at " + dashboardConfigURL + ".",
}

// dashboardReviewLead/dashboardConfigURL are the same reminder as
// dashboardNotes, split into the lead-in-sentence-then-bare-URL shape the
// action-plan renderer prints as two pre-broken lines (see
// remediation.Finding's doc comment) rather than one long sentence
// with an inline link.
const dashboardReviewLead = "Enable or disable the payment methods you want in the Dashboard:"
const dashboardConfigURL = "https://dashboard.stripe.com/settings/payment_methods?types[0]=direct&types[1]=subaccount"

func init() {
	remediation.Register(Topic, implementation{})
}

type implementation struct{}

// WebhookCheckEvents satisfies remediation.WebhookChecker: these are the
// same three webhookSignals events, plus which one `doctor
// --verify-webhooks` should trigger to exercise the round trip.
func (implementation) WebhookCheckEvents() ([]string, string) {
	events := make([]string, len(webhookSignals.WebhookEvents))
	for i, e := range webhookSignals.WebhookEvents {
		events[i] = e.Event
	}
	return events, "checkout.session.async_payment_succeeded"
}

// Run scans the directory for this kind's own webhook-handler signals
// first — that needs no credentials and can't fail, so it's always
// available — then fetches this kind's two pieces of live account
// evidence (recent API versions, payment-method configuration) and
// combines everything with the evidence doctor already computed
// (ctx.Findings). A live-evidence fetch failure still surfaces the
// webhook-handler evidence gathered so far; it's reported as a
// StatusUnknown Result (not a Go error), so the caller (doctor's cmd.go)
// doesn't discard it the way it would an actual error.
func (implementation) Run(ctx remediation.Context) (remediation.Result, error) {
	wh := codesignals.ScanSignals(ctx.Dir, webhookSignals)
	unknown := func(err error) (remediation.Result, error) {
		evidence := map[string]any{}
		if wh != nil {
			evidence["webhook_handlers_present"] = wh.AllPresent
			evidence["webhook_events"] = wh.Events
		}
		return remediation.Result{
			Topic: Topic, Title: title, Status: remediation.StatusUnknown, Message: err.Error(), Evidence: evidence, Notes: dashboardNotes,
			Report: &jsonReport{
				SchemaVersion: schemaVersion,
				Check:         checkID,
				Status:        jsonUnknown,
				Summary:       err.Error(),
				Dashboard:     jsonDashboard{RecommendedAction: "Review which payment methods to enable in the Dashboard.", URL: dashboardConfigURL},
				Documentation: []jsonDoc{{Title: title, URL: ctx.Rule.Docs}},
			},
		}, nil
	}

	creds, err := stripeapi.LoadCredentials(ctx.Config)
	if err != nil {
		return unknown(err)
	}
	client, err := stripeapi.NewClient(creds)
	if err != nil {
		return unknown(err)
	}
	vf, err := fetchVersionFacts(client, ctx.StripeAccount)
	if err != nil {
		return unknown(err)
	}
	pmf, err := fetchPMConfigFacts(client, ctx.StripeAccount)
	if err != nil {
		return unknown(err)
	}
	result := assess(ctx, vf, pmf, wh)
	result.Report = buildJSONReport(ctx, pmf, result)
	return result, nil
}

// assess combines payment-method-configuration facts, the scanned
// findings, and webhook-handler signals into one complete Result: every
// branch's Dashboard-facing Status/Message/headline comes from pmf/wh
// alone, but the result closure also attaches the hardcodedListsFinding
// (per-code-location judgment, independent of Dashboard readiness) and
// the Code/Dashboard/Webhooks Summary rows uniformly, so assess always
// returns a fully-formed Result rather than leaving Run to patch one up
// afterward.
func assess(ctx remediation.Context, vf versionFacts, pmf pmConfigFacts, wh *codesignals.HandlerSignals) remediation.Result {
	evidence := map[string]any{
		"versions_sampled":              vf.Sampled,
		"versions_ok":                   vf.VersionsOK,
		"versions_mixed":                vf.VersionsMixed,
		"payment_method_configurations": pmf.ConfigCount,
		"enabled_methods":               pmf.EnabledMethods,
		"unavailable_methods":           pmf.UnavailableMethods,
	}
	if wh != nil {
		evidence["webhook_handlers_present"] = wh.AllPresent
		evidence["webhook_events"] = wh.Events
	}

	result := func(status remediation.Status, headline, msg string) remediation.Result {
		findings := []remediation.Finding{dashboardFinding(msg), webhooksFinding(wh)}
		codeSummary := "No explicit payment method lists found"
		if n := len(ctx.Findings); n > 0 {
			findings = append([]remediation.Finding{hardcodedListsFinding(ctx, vf)}, findings...)
			listNoun := "lists"
			if n == 1 {
				listNoun = "list"
			}
			codeSummary = fmt.Sprintf("%d explicit payment method %s found", n, listNoun)
		}
		return remediation.Result{
			Topic:   Topic,
			Title:   title,
			Status:  status,
			Message: msg,
			Summary: []remediation.SummaryRow{
				{Category: "Code", Value: codeSummary},
				{Category: "Dashboard", Value: headline},
				{Category: "Webhooks", Value: webhookSummaryPhrase(wh)},
			},
			Evidence: evidence,
			Notes:    dashboardNotes,
			Findings: findings,
		}
	}

	if !pmf.ConfiguredOK {
		return result(remediation.StatusBlocked, "Blocked — no payment methods enabled in the Dashboard",
			"No payment methods are turned on in an active Dashboard payment-method configuration yet.")
	}

	// preserve_existing: every hardcoded method in source must already be
	// enabled AND available, or adopting dynamic methods would make it
	// disappear for customers — the doc's loudest warning about this
	// migration.
	hardcoded := map[string]bool{}
	hasDynamic := false
	for _, f := range ctx.Findings {
		switch classifyIntent(f.Value) {
		case "default-shaped", "static":
			for _, m := range quotedMethods(f.Value) {
				hardcoded[m] = true
			}
		case "dynamic":
			hasDynamic = true
		}
	}
	enabled := map[string]bool{}
	for _, m := range pmf.EnabledMethods {
		enabled[m] = true
	}
	unavailable := map[string]bool{}
	for _, m := range pmf.UnavailableMethods {
		unavailable[m] = true
	}
	var missing, inactive []string
	for m := range hardcoded {
		switch {
		case !enabled[m]:
			missing = append(missing, m)
		case unavailable[m]:
			inactive = append(inactive, m)
		}
	}
	sort.Strings(missing)
	sort.Strings(inactive)
	evidence["hardcoded_methods"] = sortedKeys(hardcoded)
	evidence["missing_methods"] = missing
	evidence["inactive_methods"] = inactive
	evidence["dynamic_findings"] = hasDynamic

	if len(missing) > 0 {
		return result(remediation.StatusCaution, "Caution — Dashboard doesn't yet offer every hardcoded method",
			"Turn on "+strings.Join(missing, ", ")+" in the Dashboard first. Your code hardcodes them today, and the Dashboard needs to keep offering everything the code currently does.")
	}
	if len(inactive) > 0 {
		return result(remediation.StatusCaution, "Caution — some enabled methods aren't active yet",
			strings.Join(inactive, ", ")+" are toggled on in the Dashboard, but their capability isn't active for this account yet (check Settings > Payment methods, or contact support). They won't render either way until that's resolved.")
	}
	source := "the Dashboard"
	if pmf.ActiveConfigIsDefault {
		source = "your default payment method configuration"
	}
	return result(remediation.StatusReady, "Ready — Dashboard already covers your code",
		"Ready: "+source+" already offers everything the code hardcodes — "+strings.Join(pmf.EnabledMethods, ", ")+".")
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
