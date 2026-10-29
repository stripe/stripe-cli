// Package dpm implements the "dpm" topic's remediation: adopting Dynamic
// Payment Methods by removing a hardcoded payment_method_types list.
// See pkg/doctor/remediation's package doc for how to plug in a new
// topic; this is the worked example.
package dpm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/codesignals"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/stripeapi"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

// Topic is the catalog.Rule.RemediationTopic this package implements.
const Topic = "dpm"

// title is this topic's plain-English name, for doctor's action-plan
// renderer.
const title = "Dynamic payment methods"

// webhookSignals are the delayed-notification events a migrated
// integration is expected to handle.
var webhookSignals = &codesignals.PackSignals{
	WebhookEvents: []codesignals.WebhookEvent{
		{Event: "checkout.session.completed", Note: "Checkout finished. If payment is still pending, wait for payment confirmation before fulfilling the order."},
		{Event: "checkout.session.async_payment_succeeded", Note: "A delayed payment succeeded. Fulfill the order if you haven't already done so."},
		{Event: "checkout.session.async_payment_failed", Note: "A delayed payment failed. Update the order and notify the customer as appropriate."},
	},
}

// dashboardNotes is the standing reminder to confirm the Dashboard's
// payment-method configuration, set on Result.Notes unconditionally so
// it renders as its own action rather than a trailing clause.
var dashboardNotes = []string{
	"Enable the payment methods you want to use at " + dashboardConfigURL + ".",
}

// dashboardReviewLead/dashboardConfigURL are the same reminder as
// dashboardNotes, split into a lead-in sentence and a bare URL so the
// action-plan renderer can print them as two separate lines.
const dashboardReviewLead = "Enable or disable the payment methods you want in the Dashboard:"
const dashboardConfigURL = "https://dashboard.stripe.com/settings/payment_methods?types[0]=direct&types[1]=subaccount"

func init() {
	remediation.Register(Topic, implementation{})
}

type implementation struct{}

// WebhookCheckEvents satisfies remediation.WebhookChecker.
func (implementation) WebhookCheckEvents() ([]string, string) {
	events := make([]string, len(webhookSignals.WebhookEvents))
	for i, e := range webhookSignals.WebhookEvents {
		events[i] = e.Event
	}
	return events, "checkout.session.async_payment_succeeded"
}

// Run scans for webhook-handler signals, fetches live account evidence,
// and assesses readiness. A fetch failure returns a StatusUnknown
// Result instead of an error, so the webhook evidence already gathered
// isn't discarded.
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

// assess judges whether the Dashboard's payment-method configuration
// covers what the code hardcodes, and builds the full Result around
// that judgment.
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

	// Every hardcoded method must be both enabled and available, or
	// removing the hardcoded list would make it disappear for customers.
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
