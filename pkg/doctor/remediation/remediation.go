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
//     remediation/dpm for a worked example implementing "dpm".
//  3. Blank-import your subpackage from pkg/doctor/cmd.go (next to the
//     dpm import) so your init() actually runs.
//
// See README.md in this directory for the full integration guide and a
// function-by-function reference.
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
//
// See Example (the minimum viable Implementation — just Status and
// Message, no live evidence, no custom rendering) and Example_actionPlan
// (the opt-in pieces: Title, and Findings for both the text action-plan
// renderer and, via FindingsJSON, a custom --json contract) in
// example_test.go; both are runnable (go test -run Example -v), so they
// can't go stale the way a comment-only sample could.
//
// Quick reference, beyond Implementation/Context/Register/Lookup:
//
//   - Result.Status/Message are the only fields a minimal Implementation
//     needs to set.
//   - Result.Title/Summary/Findings drive doctor's richer text
//     action-plan renderer; a topic that sets none of them still
//     renders fine via the older generic layout.
//   - Result.FindingVerdicts lets you override judge's generic REVIEW
//     verdict per scanned finding (see FindingKey).
//   - Result.Report, when set, replaces doctor's entire --json body with
//     your own schema; FindingsJSON converts Result.Findings into that
//     schema's generic finding shape so you don't re-describe the same
//     facts twice.
//   - WebhookChecker is an optional second interface for `doctor --verify-webhooks`'s
//     webhook round-trip check.
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
// Title and Findings drive doctor's human-facing action-plan renderer:
// Title names the migration in plain English (falling back to the bare
// topic string if empty); Summary is the short summary table shown up
// front (see SummaryRow); Findings is the ordered, numbered list of
// action items this remediation wants a human to work through —
// doctor's renderer treats the words inside each Finding opaquely (see
// Finding). A topic that sets neither Title nor Findings still renders
// fine — doctor falls back to its older generic summary/details layout.
type Result struct {
	Topic           string            `json:"topic"`
	Status          Status            `json:"status"`
	Title           string            `json:"title,omitempty"`
	Summary         []SummaryRow      `json:"summary,omitempty"`
	Message         string            `json:"message"`
	Evidence        map[string]any    `json:"evidence,omitempty"`
	FindingVerdicts map[string]string `json:"finding_verdicts,omitempty"`
	Notes           []string          `json:"notes,omitempty"`
	Findings        []Finding         `json:"findings,omitempty"`

	// Report, when set, is this topic's own fully-formed --json document
	// (its own schema, versioned independently of doctor's) — doctor's
	// cmd.go treats it opaquely: if present, it's emitted as the entire
	// --json body in place of the generic DoctorReport shape, rather than
	// nested inside remediation_result (hence json:"-" — it's not part
	// of this Result's own JSON representation). A topic that leaves it
	// nil is rendered via the generic schema as before. A topic whose
	// Report embeds its own findings array typically builds it by calling
	// FindingsJSON over (a subset of) this same Result.Findings, rather
	// than hand-authoring a second, parallel description of the same
	// facts — see FindingsJSON's doc comment.
	Report any `json:"-"`
}

// SummaryRow is one line in doctor's action-plan summary table: a short
// category label and its value (e.g. {"Code", "2 locations need
// review"}). Freeform by design — a remediation defines as many rows,
// with whatever categories fit its own shape, as it needs; doctor's
// renderer treats both strings opaquely, aligning them into columns
// without hardcoding what the categories should be. A nil/empty
// Result.Summary means no table is shown.
type SummaryRow struct {
	Category string `json:"category"`
	Value    string `json:"value"`
}

// Finding is one numbered, human-facing action item a remediation
// contributes to doctor's action-plan renderer — a heading, zero or more
// body paragraphs, and zero or more line items (a flagged code location,
// a webhook event, or similar) — AND, via FindingsJSON, the same
// finding's --json representation. Writing one Finding instead of a
// separate text description and a separate JSON description is the
// point: the structured facts (ID, Items, RequiresVerification,
// SuggestedParameter, Evidence) can only be written once, so they can't
// drift between the two output modes the way two independently-authored
// descriptions of the same fact can.
//
// A body paragraph containing a literal "\n" is split into independently
// word-wrapped segments with no blank line between them (e.g. a lead-in
// sentence followed by a bare URL, which — having no spaces — is left
// intact on its own line regardless of width). Numbering, wrapping, and
// indentation in text are the renderer's job; the words are the
// remediation's.
//
// A Finding with Heading == "" is text-only-excluded: the action-plan
// renderer skips it (a remediation's way of opting a section out, e.g.
// no webhook signals were scanned at all). A Finding with ID == "" is
// JSON-only-excluded: FindingsJSON skips it, for a Finding that only
// makes sense as prose (e.g. a standing Dashboard reminder tied to
// Result.Status rather than describing one discrete, independently
// reportable fact).
//
// Implication and ActionDescription are this Finding's JSON
// "implication" and "recommended_action.description" (see FindingsJSON);
// when left empty, FindingsJSON falls back to Body[0]/Body[1], so a
// Finding whose text and JSON prose should say the same thing only has
// to write it once. Set them explicitly when the two audiences
// genuinely need different wording (as dpm's own findings do — the text
// renderer's conversational voice and the JSON contract's terser one
// aren't meant to match word for word).
type Finding struct {
	Heading string   `json:"heading,omitempty"`
	Body    []string `json:"body,omitempty"`

	ID                       string   `json:"id,omitempty"`
	Status                   Status   `json:"status,omitempty"` // falls back to the enclosing Result's Status when empty
	Implication              string   `json:"implication,omitempty"`
	ActionDescription        string   `json:"action_description,omitempty"`
	Items                    []Item   `json:"items,omitempty"`
	RequiresVerification     []string `json:"requires_verification,omitempty"`
	SuggestedParameter       *Param   `json:"suggested_parameter,omitempty"`
	SafeToApplyWithoutReview bool     `json:"safe_to_apply_without_review,omitempty"`
	Evidence                 any      `json:"evidence,omitempty"`
}

// Item is one line item under a Finding: a flagged code location, a
// webhook event, or similar. Heading/Body are the text sub-heading and
// body line (e.g. "lib/checkout.rb:11" / "Current list: ['card']", or a
// webhook event name / its note). Fields, when set, is this item's own
// structured data, used by FindingsJSON's default evidence envelope
// ({"items": [...]}) — ignored when the enclosing Finding's Evidence
// overrides that envelope with something more specific.
type Item struct {
	Heading string `json:"heading,omitempty"`
	Body    string `json:"body,omitempty"`
	Fields  any    `json:"fields,omitempty"`
}

// Param names a suggested parameter/value pair, e.g. for
// Finding.SuggestedParameter.
type Param struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// FindingJSON is the generic --json shape one Finding becomes via
// FindingsJSON.
type FindingJSON struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	Evidence          any    `json:"evidence,omitempty"`
	Implication       string `json:"implication,omitempty"`
	RecommendedAction any    `json:"recommended_action,omitempty"`
}

type recommendedActionJSON struct {
	Description              string   `json:"description"`
	SuggestedParameter       *Param   `json:"suggested_parameter,omitempty"`
	RequiresVerification     []string `json:"requires_verification,omitempty"`
	SafeToApplyWithoutReview bool     `json:"safe_to_apply_without_review"`
}

// FindingsJSON converts findings to their generic --json shape, for a
// topic building its own top-level Report to embed under a "findings"
// key — see Result.Report. A Finding with ID == "" is skipped (see
// Finding's doc comment). fallback supplies the status for any included
// Finding whose own Status is empty — typically the enclosing Result's
// Status.
func FindingsJSON(findings []Finding, fallback Status) []FindingJSON {
	var out []FindingJSON
	for _, f := range findings {
		if f.ID == "" {
			continue
		}
		status := f.Status
		if status == "" {
			status = fallback
		}
		implication := f.Implication
		if implication == "" && len(f.Body) > 0 {
			implication = f.Body[0]
		}
		description := f.ActionDescription
		if description == "" && len(f.Body) > 1 {
			description = f.Body[1]
		}
		evidence := f.Evidence
		if evidence == nil && len(f.Items) > 0 {
			fields := make([]any, 0, len(f.Items))
			for _, it := range f.Items {
				if it.Fields != nil {
					fields = append(fields, it.Fields)
				}
			}
			evidence = map[string]any{"items": fields}
		}
		out = append(out, FindingJSON{
			ID:          f.ID,
			Status:      JSONStatus(status),
			Evidence:    evidence,
			Implication: implication,
			RecommendedAction: recommendedActionJSON{
				Description:              description,
				SuggestedParameter:       f.SuggestedParameter,
				RequiresVerification:     f.RequiresVerification,
				SafeToApplyWithoutReview: f.SafeToApplyWithoutReview,
			},
		})
	}
	return out
}

// JSONStatus maps remediation's Status vocabulary to the vocabulary
// doctor's richer --json contracts use — generic across any topic, not
// specific to any one of them.
func JSONStatus(s Status) string {
	switch s {
	case StatusReady:
		return "ready"
	case StatusCaution:
		return "action_required"
	case StatusBlocked:
		return "blocked"
	case StatusReview:
		return "review_required"
	case StatusUnknown:
		return "unknown"
	default:
		return string(s)
	}
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

// WebhookChecker is an optional interface an Implementation can satisfy to
// support `doctor --verify-webhooks`'s webhook round-trip check
// (pkg/doctor/webhookcheck). doctor's engine has no idea which webhook
// events matter for any given topic; WebhookCheckEvents names the full set
// of events the round-trip check should listen for, and which one of
// those to trigger to exercise it. A topic with nothing to verify this
// way simply doesn't implement WebhookChecker — doctor --verify-webhooks then fails
// clearly rather than guessing.
type WebhookChecker interface {
	WebhookCheckEvents() (events []string, triggerEvent string)
}

var registry = map[string]Implementation{}

// Register makes impl available under topic — the same string a rule's
// "doctorTopic" field names (catalog.Rule.RemediationTopic). Call this
// from an init() in your own package (see remediation/dpm for
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
