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
// FindingsVerdict, when non-empty, is a "CLASS: message" string (the same
// shape judge.Evaluate produces) that doctor's engine substitutes for
// every finding's own verdict, uniformly. Use it for account-wide facts
// that apply the same way to every finding regardless of that finding's
// own code shape — e.g. whether recent traffic is on a safe API version —
// which a per-finding intent classifier has no way to know on its own.
// "" means no override: findings keep their intent-based verdict.
//
// Notes are additional action items that apply regardless of Status
// (e.g. "go turn on the payment methods you want in the Dashboard") —
// rendered as their own separate actions, not appended onto Message, so
// they don't get lost as a trailing clause of an unrelated sentence.
type Result struct {
	Topic           string         `json:"topic"`
	Status          Status         `json:"status"`
	Message         string         `json:"message"`
	Evidence        map[string]any `json:"evidence,omitempty"`
	FindingsVerdict string         `json:"findings_verdict,omitempty"`
	Notes           []string       `json:"notes,omitempty"`
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
