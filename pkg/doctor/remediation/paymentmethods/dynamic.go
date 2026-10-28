// Package paymentmethods implements the "dpm" topic's remediation:
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
//  3. Per-finding judgment (classifyIntent/intentVerdict, findingVerdicts)
//     — doctor's judge package has no idea what "card"/"link" or
//     payment_method_types even mean; it gives every finding a generic
//     REVIEW by default, and findingVerdicts supplies the real
//     CANDIDATE/SKIP/REVIEW judgment per finding via Result.FindingVerdicts.
//  4. Webhook handler coverage, both statically (webhookSignals, scanned
//     via pkg/doctor/codesignals) and live (LiveCheckEvents, behind
//     `doctor --live` via pkg/doctor/webhookcheck) — this topic declares
//     its own expected events; doctor's engine wires in neither
//     generically.
//
// assess combines 1-2 and the webhook-handler evidence into one
// Dashboard-facing Result, purely about Dashboard/webhook readiness.
// findingVerdicts (1 again, plus 3) produces the per-finding overrides
// separately, since those are facts about specific code sites, not
// Dashboard actions. offeringPolicy and dashboardNotes are hardcoded
// here rather than read off the rule — there's no more per-remediation
// data on the rule at all (just the bare "doctorTopic" string that got
// us to this implementation in the first place); this topic is expected
// to know everything about its own remediation already.
package paymentmethods

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/codesignals"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
	"github.com/stripe/stripe-cli/pkg/doctor/stripeapi"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// Topic is the catalog.Rule.RemediationTopic ("doctorTopic") this package
// implements.
const Topic = "dpm"

// title is this topic's plain-English name for doctor's action-plan
// renderer (remediation.Result.Title) — the renderer itself has no idea
// what "dpm" stands for.
const title = "Dynamic payment methods"

// offeringPolicy governs how Dashboard adoption treats methods the code
// already hardcodes: preserve_existing requires the Dashboard to keep
// offering everything the code currently does. It's the only policy this
// topic implements, so it's a constant rather than something read off
// the rule.
const offeringPolicy = "preserve_existing"

// dpmCutoff is the API version at/after which removing payment_method_types
// alone enables dynamic payment methods; before it, automatic_payment_methods
// [enabled]=true must be added or methods are silently lost. Specific to
// this topic's judgment, not a doctor-wide concept.
const dpmCutoff = "2023-08-16"

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
// remediation.ActionSection's doc comment) rather than one long sentence
// with an inline link.
const dashboardReviewLead = "Once you've verified your flows support them, enable the payment methods you want in the Dashboard:"
const dashboardConfigURL = "https://dashboard.stripe.com/settings/payment_methods?types[0]=direct&types[1]=subaccount"

func init() {
	remediation.Register(Topic, implementation{})
}

type implementation struct{}

// LiveCheckEvents satisfies remediation.LiveChecker: these are the same
// three webhookSignals events, plus which one `doctor --live` should
// trigger to exercise the round trip.
func (implementation) LiveCheckEvents() ([]string, string) {
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
		return remediation.Result{Topic: Topic, Title: title, Status: remediation.StatusUnknown, Message: err.Error(), Evidence: evidence, Notes: dashboardNotes}, nil
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
	result.FindingVerdicts = findingVerdicts(vf, ctx.Findings)
	if n := len(ctx.Findings); n > 0 {
		result.Sections = append([]remediation.ActionSection{findingsSection(ctx, vf)}, result.Sections...)
		listNoun := "lists"
		if n == 1 {
			listNoun = "list"
		}
		result.CodeSummary = fmt.Sprintf("%d explicit payment method %s found", n, listNoun)
	}
	return result, nil
}

// versionFacts summarizes the API versions this account's recent traffic
// actually used.
type versionFacts struct {
	Sampled       int
	NoEvents      bool // no recent events: version facts are unknowable
	VersionsOK    bool // every sampled event version >= dpmCutoff
	VersionsMixed bool // some but not all versions >= dpmCutoff
}

// fetchVersionFacts samples the account's most recent events for their
// api_version, the same evidence doctor's old account-facts fetch used.
func fetchVersionFacts(client *stripe.Client, stripeAccount string) (versionFacts, error) {
	var evts struct {
		Data []struct {
			APIVersion string `json:"api_version"`
		} `json:"data"`
	}
	if err := stripeapi.Get(client, stripeAccount, "/v1/events", "limit=20", nil, &evts); err != nil {
		return versionFacts{}, err
	}
	ge, total := 0, 0
	for _, e := range evts.Data {
		if e.APIVersion == "" {
			continue
		}
		total++
		if datePrefix(e.APIVersion) >= dpmCutoff {
			ge++
		}
	}
	return versionFacts{
		Sampled:       total,
		NoEvents:      total == 0,
		VersionsOK:    total > 0 && ge == total,
		VersionsMixed: ge > 0 && ge < total,
	}, nil
}

// datePrefix compares versions by their YYYY-MM-DD prefix, which handles
// both classic ("2023-08-16") and named ("2026-06-24.dahlia") formats.
func datePrefix(v string) string {
	if len(v) >= 10 {
		return v[:10]
	}
	return v
}

// pmConfigFacts summarizes the account's active payment-method
// configuration: which methods are toggled on, and which of those are
// toggled on but not actually available (capability inactive — they will
// NOT render; both are required).
type pmConfigFacts struct {
	ConfigCount        int
	ActiveConfigID     string
	ActiveConfigName   string
	EnabledMethods     []string
	UnavailableMethods []string
	ConfiguredOK       bool
}

// fetchPMConfigFacts reads /v1/payment_method_configurations and picks ONE
// governing config — the default, else the first active one — the same
// selection doctor's old account-facts fetch used.
func fetchPMConfigFacts(client *stripe.Client, stripeAccount string) (pmConfigFacts, error) {
	var pmc struct {
		Data []map[string]any `json:"data"`
	}
	if err := stripeapi.Get(client, stripeAccount, "/v1/payment_method_configurations", "", nil, &pmc); err != nil {
		return pmConfigFacts{}, err
	}

	var f pmConfigFacts
	var chosen map[string]any
	for _, cfg := range pmc.Data {
		active, _ := cfg["active"].(bool)
		if !active {
			continue // deactivated demo leftovers shouldn't inflate anything
		}
		f.ConfigCount++
		isDefault, _ := cfg["is_default"].(bool)
		if isDefault || chosen == nil {
			chosen = cfg
		}
	}
	if chosen != nil {
		if name, ok := chosen["name"].(string); ok {
			f.ActiveConfigName = name
		}
		if id, ok := chosen["id"].(string); ok {
			f.ActiveConfigID = id
		}
		for field, v := range chosen {
			m, ok := v.(map[string]any)
			if !ok {
				continue
			}
			dp, ok := m["display_preference"].(map[string]any)
			if !ok || dp["value"] != "on" {
				continue
			}
			f.EnabledMethods = append(f.EnabledMethods, field)
			if avail, ok := m["available"].(bool); ok && !avail {
				f.UnavailableMethods = append(f.UnavailableMethods, field)
			}
		}
		sort.Strings(f.EnabledMethods)
		sort.Strings(f.UnavailableMethods)
	}
	f.ConfiguredOK = f.ConfigCount > 0 && len(f.EnabledMethods) > 0
	return f, nil
}

var quotedMethodRe = regexp.MustCompile(`["']([a-z0-9_]+)["']`)

// quotedMethods extracts the static method names from a finding's value
// expression, e.g. "['card', 'ideal']" -> ["card", "ideal"].
func quotedMethods(value string) []string {
	var out []string
	for _, m := range quotedMethodRe.FindAllStringSubmatch(value, -1) {
		out = append(out, m[1])
	}
	return out
}

var identRe = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

// classifyIntent inspects a finding's value expression and buckets it.
// This is dpm's own domain logic (what a hardcoded payment_method_types
// value shape implies) — doctor's judge package has no idea "card" and
// "link" are payment method names, so this can't live there.
//
//	default-shaped — static list of only card/link: the classic pre-DPM
//	                 hardcode and the migration's actual target
//	deliberate     — static single non-card method (['oxxo']): per-method
//	                 integration, almost certainly intentional
//	static         — static multi-method list beyond card/link
//	dynamic        — computed at runtime (identifiers, conditionals)
func classifyIntent(value string) string {
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

// intentVerdict turns a classifyIntent bucket into a CANDIDATE/SKIP/REVIEW
// judgment — the real per-finding verdict doctor's generic judge package
// can't produce on its own.
func intentVerdict(intent string) string {
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

// versionGate is a uniform account-wide fact derived from the live
// API-version evidence alone — a version problem applies the same way to
// every payment_method_types removal candidate regardless of that
// finding's own code shape. A zero-value versionGate (empty Class) means
// no version problem: each finding falls through to its own
// classifyIntent/intentVerdict instead (see findingVerdicts). Reason and
// Instruction are kept separate (rather than one sentence) so the
// action-plan renderer can show them as two paragraphs (see
// findingsSection); flat joins them back into the single-line
// "CLASS: message" shape findingVerdicts needs for the JSON
// per-finding-override contract.
//
// A pre-cutoff version is CAUTION, not BLOCKED: the fix isn't "do
// nothing" or "wait for traffic to move versions" — it's "delete
// payment_method_types AND add automatic_payment_methods[enabled]=true",
// a known, actionable change, not a dead end. Only NoEvents (no version
// evidence at all) stays REVIEW, since there's no fix to name yet.
type versionGate struct {
	Class       string
	Reason      string
	Instruction string
}

func (g versionGate) flat() string {
	if g.Class == "" {
		return ""
	}
	return g.Class + ": " + g.Reason + " " + g.Instruction
}

func findingsVersionGate(vf versionFacts) versionGate {
	switch {
	case vf.NoEvents:
		return versionGate{
			Class:       "REVIEW",
			Reason:      "Couldn't find any recent API traffic for this account, so its API version is unknown.",
			Instruction: "Confirm it's at/after " + dpmCutoff + " before removing payment_method_types — on an older version, removing it by itself silently drops payment methods.",
		}
	case vf.VersionsMixed:
		return versionGate{
			Class:       "CAUTION",
			Reason:      "This account's recent traffic mixes API versions from before and after " + dpmCutoff + ".",
			Instruction: "Plain removal is only safe for the newer requests; on the older ones, add automatic_payment_methods[enabled]=true instead, which keeps them working on every version.",
		}
	case !vf.VersionsOK:
		return versionGate{
			Class:       "CAUTION",
			Reason:      "Recent API traffic for this account uses versions before " + dpmCutoff + ". On those versions, you must explicitly enable automatic payment methods. Removing payment_method_types alone is not enough.",
			Instruction: "In each flow, confirm the API version, then replace payment_method_types with automatic_payment_methods[enabled]=true.",
		}
	default:
		return versionGate{}
	}
}

// findingsVersionVerdict is findingsVersionGate's flattened single-line
// form — kept as its own function since that's the shape findingVerdicts
// and its tests deal in.
func findingsVersionVerdict(vf versionFacts) string {
	return findingsVersionGate(vf).flat()
}

// findingVerdicts builds the per-finding override map doctor's engine
// applies in place of judge's generic REVIEW (see
// remediation.Result.FindingVerdicts). A version problem, when present,
// overrides every finding uniformly (same key, same message) since it's
// an account-wide fact; otherwise each finding is classified on its own
// code shape.
func findingVerdicts(vf versionFacts, findings []catalog.Finding) map[string]string {
	gate := findingsVersionVerdict(vf)
	out := make(map[string]string, len(findings))
	for _, f := range findings {
		key := remediation.FindingKey(f.File, f.Line)
		if gate != "" {
			out[key] = gate
			continue
		}
		out[key] = intentVerdict(classifyIntent(f.Value))
	}
	return out
}

// relLocation formats a finding's file:line relative to the scanned
// directory, for the action-plan renderer — the JSON findings contract
// (catalog.Finding.File) keeps the absolute path; this is purely a
// display nicety. Falls back to the absolute path if it isn't under dir.
func relLocation(dir string, f catalog.Finding) string {
	rel, err := filepath.Rel(dir, f.File)
	if err != nil {
		rel = f.File
	}
	return fmt.Sprintf("%s:%d", rel, f.Line)
}

// findingsSection builds the action-plan item for the scanned code
// locations: one shared reason/instruction when a version problem gates
// every finding uniformly (the common case), else each finding reviewed
// on its own classifyIntent/intentVerdict judgment.
func findingsSection(ctx remediation.Context, vf versionFacts) remediation.ActionSection {
	sec := remediation.ActionSection{Heading: "Replace explicit lists with automatic payment methods"}
	if gate := findingsVersionGate(vf); gate.Class != "" {
		sec.Body = []string{gate.Reason, gate.Instruction}
		for _, f := range ctx.Findings {
			sec.Items = append(sec.Items, remediation.ActionItem{Heading: relLocation(ctx.Dir, f), Body: "Current list: " + f.Value})
		}
		return sec
	}
	sec.Body = []string{"Each flagged location below needs its own review before removing payment_method_types."}
	for _, f := range ctx.Findings {
		sec.Items = append(sec.Items, remediation.ActionItem{
			Heading: relLocation(ctx.Dir, f),
			Body:    intentVerdict(classifyIntent(f.Value)),
		})
	}
	return sec
}

// assess combines payment-method-configuration facts, the scanned
// findings, and webhook-handler signals into one Dashboard-facing
// Result — purely about Dashboard/webhook readiness, independent of
// API-version safety and per-finding judgment (both handled separately
// by findingVerdicts, attached by Run instead, since those are facts
// about specific code sites, not Dashboard actions).
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
	evidence["offering_policy"] = offeringPolicy

	result := func(status remediation.Status, headline, msg string) remediation.Result {
		return remediation.Result{
			Topic:        Topic,
			Title:        title,
			Status:       status,
			Headline:     headline,
			SummaryLabel: "Routing",
			Message:      msg,
			Evidence:     evidence,
			Notes:        dashboardNotes,
			Sections:     []remediation.ActionSection{dashboardSection(msg), webhooksSection(wh)},
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
			"Turn on "+strings.Join(missing, ", ")+" in the Dashboard first. Your code hardcodes them today, and offeringPolicy preserve_existing requires the Dashboard to keep offering everything the code currently does.")
	}
	if len(inactive) > 0 {
		return result(remediation.StatusCaution, "Caution — some enabled methods aren't active yet",
			strings.Join(inactive, ", ")+" are toggled on in the Dashboard, but their capability isn't active for this account yet (check Settings > Payment methods, or contact support). They won't render either way until that's resolved.")
	}
	if hasDynamic {
		return result(remediation.StatusReview, "Review required — some lists are built at runtime",
			"Some payment method lists are built at runtime. Check whether that logic restricts payment methods for particular checkout flows. Preserve any required restrictions when switching to dynamic payment methods.")
	}
	if wh != nil && !wh.AllPresent {
		return result(remediation.StatusCaution, "Caution — webhook handlers missing",
			"Ready to adopt dynamic payment methods, but add the delayed-notification webhook handlers first. Some newly available methods only confirm payment asynchronously.")
	}
	return result(remediation.StatusReady, "Ready — Dashboard already covers your code",
		"Ready: the Dashboard already offers everything the code hardcodes, and webhook handling is in place.")
}

// dashboardSection is the action-plan item for the Dashboard-facing
// readiness check: msg (assess's own Message for this Result) plus the
// standing reminder to review which payment methods are enabled,
// rendered as a lead-in sentence followed by a bare URL rather than one
// long sentence with an inline link.
func dashboardSection(msg string) remediation.ActionSection {
	return remediation.ActionSection{
		Heading: "Verify routing before enabling more payment methods",
		Body:    []string{msg, dashboardReviewLead + "\n" + dashboardConfigURL},
	}
}

// webhooksSection is the action-plan item for webhook-handler coverage.
// Returns a zero-value ActionSection (skipped by the renderer) when wh is
// nil — no webhook signals were scanned for at all.
func webhooksSection(wh *codesignals.HandlerSignals) remediation.ActionSection {
	if wh == nil {
		return remediation.ActionSection{}
	}
	sec := remediation.ActionSection{Heading: "Verify handling for delayed payments"}
	sec.Body = append(sec.Body, "Checkout can finish before a delayed payment succeeds. Make sure your integration distinguishes checkout completion from payment confirmation.")
	if wh.AllPresent {
		sec.Body = append(sec.Body, "All expected webhook handlers were found in the scanned code.")
	} else {
		sec.Body = append(sec.Body, "These handlers weren't found in the scanned code. If they're implemented elsewhere, verify them there.")
	}
	for _, e := range wh.Events {
		item := remediation.ActionItem{Heading: e.Event}
		switch {
		case e.Present && len(e.Files) > 0:
			item.Body = "Found in " + e.Files[0]
		case e.Present:
			item.Body = "Found in scanned code"
		default:
			item.Body = e.Note
		}
		sec.Items = append(sec.Items, item)
	}
	return sec
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
