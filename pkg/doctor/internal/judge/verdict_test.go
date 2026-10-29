package judge

import (
	"strings"
	"testing"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
)

// TestEvaluateRemoveIsGenericReview pins that a "remove" rule's findings
// get a generic REVIEW verdict, with no topic-specific knowledge.
func TestEvaluateRemoveIsGenericReview(t *testing.T) {
	rule := catalog.Rule{Action: "remove"}
	findings := []catalog.Finding{{Param: "payment_method_types", Value: `['card']`}}

	r := Evaluate(rule, findings)
	if len(r.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(r.Findings))
	}
	got := r.Findings[0]
	if got.Class != "REVIEW" {
		t.Errorf("expected class REVIEW, got %q", got.Class)
	}
	if !strings.Contains(got.Verdict, "payment_method_types") {
		t.Errorf("expected verdict to name the param, got %q", got.Verdict)
	}
	if r.Summary["REVIEW"] != 1 {
		t.Errorf("expected summary REVIEW=1, got %v", r.Summary)
	}
}

// TestEvaluateAdvise pins that an "advise" rule's findings carry the
// rule's own message/action/version.
func TestEvaluateAdvise(t *testing.T) {
	rule := catalog.Rule{Action: "advise", Message: "renamed field", ActionText: "use the new field instead", IntroducedIn: "2024-01-01"}
	findings := []catalog.Finding{{Param: "old_field"}}

	r := Evaluate(rule, findings)
	got := r.Findings[0]
	if got.Class != "ADVISE" {
		t.Errorf("expected class ADVISE, got %q", got.Class)
	}
	for _, want := range []string{"renamed field", "use the new field instead", "2024-01-01"} {
		if !strings.Contains(got.Verdict, want) {
			t.Errorf("expected verdict to contain %q, got %q", want, got.Verdict)
		}
	}
}
