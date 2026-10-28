// Package remediation is the plugin boundary for doctor remediation
// implementations. A migration rule names its own topic — e.g. "dpm" —
// via its "doctorTopic" field (catalog.Rule.RemediationTopic), fetched
// from the doctor_rules API. That string is the only thing doctor's
// engine knows about remediations; it has no opinion on what a topic's
// remediation actually does. Teams that own a topic implement what it
// actually does here:
//
//  1. Write a type satisfying Implementation for your topic.
//  2. Register it from an init() in your own subpackage — see
//     remediation/paymentmethods for a worked example implementing "dpm".
//  3. Blank-import your subpackage from pkg/doctor/cmd.go (next to the
//     paymentmethods import) so your init() actually runs.
//
// Context carries what doctor already computed for you (the scanned
// source findings) so you don't need to re-scan, plus what you need to
// fetch your own read-only account evidence (Config/StripeAccount, passed
// to pkg/doctor/stripeapi) and any code signals your topic cares about
// (Dir, passed to pkg/doctor/codesignals) — doctor's engine doesn't
// declare those generically; each topic owns deciding what it needs, the
// same way it owns everything else about its own assessment. doctor
// never acts on a remediation itself — it only looks up the rule's topic
// in the registry and renders whatever Result comes back.
package remediation

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
)

// Status is a Result's top-line verdict on whether a remediation can be
// applied.
type Status string

const (
	StatusReady   Status = "ready"   // safe to apply as-is
	StatusCaution Status = "caution" // apply, but Message names something to fix first
	StatusBlocked Status = "blocked" // not safe yet; Message names the blocker
	StatusReview  Status = "review"  // needs human judgment, not auto-decidable
	StatusUnknown Status = "unknown" // couldn't gather enough evidence (e.g. no credentials)
)

// Result is one topic's remediation assessment for this run. Evidence is
// free-form, topic-specific supporting data (e.g. the account's API
// versions, its payment-method configuration) — doctor doesn't interpret
// it, just passes it through to --json for debugging/auditing.
//
// FindingVerdicts, when non-nil, maps a finding key ("<file>:<line>",
// matching catalog.Finding.File/Line) to a "CLASS: message" string (the
// same shape judge.Evaluate produces) that doctor's engine substitutes
// for that finding's own verdict. judge.Evaluate has no topic-specific
// domain knowledge, so this is how a registered remediation supplies
// real per-finding judgment — set the same value for every key to apply
// one account-wide fact uniformly (e.g. whether recent traffic is on a
// safe API version), or a different value per key to classify each
// finding on its own code shape. A finding with no entry keeps whatever
// generic verdict judge.Evaluate already gave it.
//
// Notes are additional action items that apply regardless of Status
// (e.g. "go turn on the payment methods you want in the Dashboard") —
// rendered as their own separate actions, not appended onto Message, so
// they don't get lost as a trailing clause of an unrelated sentence.
//
// Title and Sections drive doctor's human-facing action-plan renderer:
// Title names the migration in plain English (falling back to the bare
// topic string if empty); Headline is a short clause for the one-line
// summary row named by SummaryLabel (falling back to "Dashboard" when
// SummaryLabel is empty); CodeSummary overrides the summary table's
// "Code" row value entirely, when the generic finding-count-by-class
// phrase doctor would otherwise compute doesn't fit this topic's own
// framing; Sections is the ordered, numbered list of action items this
// remediation wants a human to work through — doctor's renderer treats
// the words inside each Section opaquely (see ActionSection). A topic
// that sets neither Title nor Sections still renders fine — doctor falls
// back to its older generic summary/details layout.
type Result struct {
	Topic           string            `json:"topic"`
	Status          Status            `json:"status"`
	Title           string            `json:"title,omitempty"`
	Headline        string            `json:"headline,omitempty"`
	SummaryLabel    string            `json:"summary_label,omitempty"`
	CodeSummary     string            `json:"code_summary,omitempty"`
	Message         string            `json:"message"`
	Evidence        map[string]any    `json:"evidence,omitempty"`
	FindingVerdicts map[string]string `json:"finding_verdicts,omitempty"`
	Notes           []string          `json:"notes,omitempty"`
	Sections        []ActionSection   `json:"sections,omitempty"`
}

// ActionSection is one numbered, human-facing action item a remediation
// contributes to doctor's action-plan renderer: a heading, zero or more
// body paragraphs, and zero or more line items (a flagged code location,
// a webhook event, or similar). A body paragraph containing a literal
// "\n" is split into independently word-wrapped segments with no blank
// line between them (e.g. a lead-in sentence followed by a bare URL,
// which — having no spaces — is left intact on its own line regardless
// of width). Numbering, wrapping, and indentation are the renderer's
// job; the words are the remediation's.
type ActionSection struct {
	Heading string       `json:"heading"`
	Body    []string     `json:"body,omitempty"`
	Items   []ActionItem `json:"items,omitempty"`
}

// ActionItem is one line item under an ActionSection: a short heading
// (e.g. "lib/checkout.rb:11" or a webhook event name) plus an optional
// body line (e.g. the flagged code snippet, or that event's note).
type ActionItem struct {
	Heading string `json:"heading"`
	Body    string `json:"body,omitempty"`
}

// FindingKey is the stable key a Result.FindingVerdicts entry is looked
// up by — the same key doctor's engine computes from each finding's own
// File/Line when applying overrides.
func FindingKey(file string, line int) string {
	return fmt.Sprintf("%s:%d", file, line)
}

// Context is everything an Implementation's Run gets handed. Findings is
// already computed by doctor's normal scan — reuse it rather than
// re-scanning. Dir is there for anything you want to scan yourself (e.g.
// your own webhook-handler signals, via pkg/doctor/codesignals).
// Config/StripeAccount are for fetching whatever additional read-only
// account evidence your topic needs (see pkg/doctor/stripeapi).
type Context struct {
	Dir           string
	Rule          catalog.Rule
	Findings      []catalog.Finding
	Config        *config.Config
	StripeAccount string
}

// Implementation is what a topic's remediation actually does: assess
// whether (and how) it can be applied, given ctx. Run should never mutate
// the Stripe account or the scanned directory — doctor only diagnoses.
type Implementation interface {
	Run(ctx Context) (Result, error)
}

// LiveChecker is an optional interface an Implementation can satisfy to
// support `doctor --live`'s webhook round-trip check
// (pkg/doctor/webhookcheck). doctor's engine has no idea which webhook
// events matter for any given topic; LiveCheckEvents names the full set
// of events the round-trip check should listen for, and which one of
// those to trigger to exercise it. A topic with nothing to verify this
// way simply doesn't implement LiveChecker — doctor --live then fails
// clearly rather than guessing.
type LiveChecker interface {
	LiveCheckEvents() (events []string, triggerEvent string)
}

var registry = map[string]Implementation{}

// Register makes impl available under topic — the same string a rule's
// "doctorTopic" field names (catalog.Rule.RemediationTopic). Call this
// from an init() in your own package (see remediation/paymentmethods for
// a worked example); it panics if topic is already registered, since two
// implementations silently fighting over one topic is always a bug, not
// a runtime decision.
func Register(topic string, impl Implementation) {
	if _, exists := registry[topic]; exists {
		panic(fmt.Sprintf("remediation: topic %q already registered", topic))
	}
	registry[topic] = impl
}

// Lookup returns the Implementation registered for topic, if any.
func Lookup(topic string) (Implementation, bool) {
	impl, ok := registry[topic]
	return impl, ok
}
