// Package remediation is the plugin boundary for doctor remediation
// implementations. A migration rule names its own topic (e.g. "dpm") via
// catalog.Rule.RemediationTopic; that string is all doctor's engine knows
// about remediations. To add one:
//
//  1. Write a type satisfying Implementation, with an exported
//     constructor (e.g. New()) — see remediation/dpm for a worked
//     example.
//  2. Add it to the remediation.Set built in pkg/cmd/root.go, keyed by
//     your topic string.
//
// See README.md in this directory for the full guide, and the example
// subpackage for runnable samples (go test -run Example -v).
package remediation

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
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

// Result is one topic's remediation assessment for this run. Only
// Topic/Status/Message are required; the rest are opt-in.
//
// FindingVerdicts overrides judge's generic per-finding verdict, keyed by
// FindingKey, with a "CLASS: message" string. Notes are standing action
// items rendered separately from Message, regardless of Status. Title
// and Findings drive doctor's text action-plan renderer; a Result that
// sets neither falls back to the older generic layout.
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

	// Report, when set, replaces the entire --json body with this
	// topic's own schema instead of doctor's generic one (json:"-" since
	// it isn't nested inside remediation_result). Build its findings via
	// FindingsJSON to avoid re-describing Result.Findings a second time.
	Report any `json:"-"`
}

// SummaryRow is one line in doctor's action-plan summary table. A
// remediation can report as many rows, with whatever categories it
// wants; a nil/empty Result.Summary shows no table.
type SummaryRow struct {
	Category string `json:"category"`
	Value    string `json:"value"`
}

// Finding is one numbered action item: a heading, body paragraphs, and
// line items for doctor's text renderer, plus (via FindingsJSON) its
// --json representation — one Finding drives both, so they can't drift
// out of sync.
//
// Heading == "" excludes it from text; ID == "" excludes it from
// FindingsJSON (for a Finding that's only prose, not a discrete
// reportable fact). Implication/ActionDescription fall back to
// Body[0]/Body[1] when empty; set them explicitly only when the text and
// JSON need different wording.
//
// A Body paragraph containing "\n" renders as pre-broken lines with no
// blank line between them (e.g. a lead-in sentence followed by a bare
// URL).
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

// Item is one line item under a Finding (a flagged code location, a
// webhook event, etc). Fields, when set, feeds FindingsJSON's default
// evidence envelope ({"items": [...]}); ignored if the Finding's own
// Evidence overrides that envelope.
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

// FindingsJSON converts findings to their generic --json shape, for
// embedding in a custom Result.Report. fallback supplies the status for
// any Finding whose own Status is empty.
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

// JSONStatus maps Status to the vocabulary used by FindingsJSON and
// custom --json contracts.
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

// FindingKey builds the key used in Result.FindingVerdicts.
func FindingKey(file string, line int) string {
	return fmt.Sprintf("%s:%d", file, line)
}

// Context is everything an Implementation's Run gets handed. Findings is
// already scanned — reuse it rather than re-scanning.
type Context struct {
	Dir           string
	Rule          catalog.Rule
	Findings      []catalog.Finding
	Config        *config.Config
	StripeAccount string
}

// Implementation is what a topic's remediation does. Run should never
// mutate the Stripe account or the scanned directory — doctor only
// diagnoses.
type Implementation interface {
	Run(ctx Context) (Result, error)
}

// Set maps a topic string (catalog.Rule.RemediationTopic) to the
// Implementation that handles it. Built explicitly in pkg/cmd/root.go
// and passed into doctor.NewDoctorCmd — a nil Set is valid and behaves
// as if no topic has a remediation.
type Set map[string]Implementation
