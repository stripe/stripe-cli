package paymentmethods

import (
	"strings"
	"testing"

	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/codesignals"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

func finding(value string) catalog.Finding {
	return catalog.Finding{Param: "payment_method_types", Value: value}
}

// Version-safety facts no longer gate the Dashboard Result (its Status
// reflects Dashboard-config/webhook readiness only) — they're surfaced as
// a per-finding override instead. See findingVerdicts.

func TestFindingsVersionVerdictNoEvents(t *testing.T) {
	got := findingsVersionVerdict(versionFacts{NoEvents: true})
	if !strings.HasPrefix(got, "REVIEW:") {
		t.Errorf("expected REVIEW, got %q", got)
	}
}

func TestFindingsVersionVerdictStaleVersion(t *testing.T) {
	got := findingsVersionVerdict(versionFacts{VersionsOK: false, VersionsMixed: false})
	// CAUTION, not BLOCKED: the fix is known and actionable (add
	// automatic_payment_methods[enabled]=true), not a dead end.
	if !strings.HasPrefix(got, "CAUTION:") {
		t.Errorf("expected CAUTION, got %q", got)
	}
	if !strings.Contains(got, "automatic_payment_methods[enabled]=true") {
		t.Errorf("expected the message to name the fix, got %q", got)
	}
}

func TestFindingsVersionVerdictMixedVersions(t *testing.T) {
	got := findingsVersionVerdict(versionFacts{VersionsMixed: true})
	if !strings.HasPrefix(got, "CAUTION:") {
		t.Errorf("expected CAUTION, got %q", got)
	}
}

func TestFindingsVersionVerdictVersionsOK(t *testing.T) {
	got := findingsVersionVerdict(versionFacts{VersionsOK: true})
	if got != "" {
		t.Errorf("expected no override when versions are OK, got %q", got)
	}
}

// TestClassifyIntent pins the per-finding code-shape buckets this topic
// classifies on its own — doctor's judge package has no idea "card"/
// "link" are payment method names.
func TestClassifyIntent(t *testing.T) {
	cases := map[string]string{
		`['card']`:                "default-shaped",
		`['card', 'link']`:        "default-shaped",
		`['oxxo']`:                "deliberate",
		`['card', 'ideal']`:       "static",
		`paymentMethods`:          "dynamic",
		`cond ? ['card'] : ['x']`: "dynamic",
	}
	for value, want := range cases {
		if got := classifyIntent(value); got != want {
			t.Errorf("classifyIntent(%q) = %q, want %q", value, got, want)
		}
	}
}

// TestFindingVerdicts pins that a version problem overrides every
// finding uniformly, and that otherwise each finding is classified on
// its own code shape via classifyIntent/intentVerdict.
func TestFindingVerdicts(t *testing.T) {
	findings := []catalog.Finding{
		{File: "a.rb", Line: 1, Value: `['card']`},       // default-shaped
		{File: "b.rb", Line: 2, Value: `paymentMethods`}, // dynamic
	}

	// Version problem: every finding gets the same gate message.
	gated := findingVerdicts(versionFacts{NoEvents: true}, findings)
	for _, f := range findings {
		key := remediation.FindingKey(f.File, f.Line)
		if !strings.HasPrefix(gated[key], "REVIEW: Couldn't find any recent API traffic") {
			t.Errorf("%s: expected the version-gate message, got %q", key, gated[key])
		}
	}

	// Versions OK: each finding classified on its own shape.
	clean := findingVerdicts(versionFacts{VersionsOK: true}, findings)
	if !strings.HasPrefix(clean[remediation.FindingKey("a.rb", 1)], "CANDIDATE:") {
		t.Errorf("expected a.rb:1 to be CANDIDATE, got %q", clean[remediation.FindingKey("a.rb", 1)])
	}
	if !strings.HasPrefix(clean[remediation.FindingKey("b.rb", 2)], "REVIEW:") {
		t.Errorf("expected b.rb:2 to be REVIEW, got %q", clean[remediation.FindingKey("b.rb", 2)])
	}
}

func TestAssessBlockedNoDashboardConfig(t *testing.T) {
	vf := versionFacts{VersionsOK: true}
	r := assess(remediation.Context{}, vf, pmConfigFacts{ConfiguredOK: false}, nil)
	if r.Status != remediation.StatusBlocked {
		t.Errorf("expected blocked, got %q (%s)", r.Status, r.Message)
	}
}

func TestAssessCautionMissingMethod(t *testing.T) {
	vf := versionFacts{VersionsOK: true}
	pmf := pmConfigFacts{ConfiguredOK: true, EnabledMethods: []string{"card"}}
	ctx := remediation.Context{Findings: []catalog.Finding{finding(`['card', 'ideal']`)}}

	r := assess(ctx, vf, pmf, nil)
	if r.Status != remediation.StatusCaution {
		t.Fatalf("expected caution, got %q (%s)", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "ideal") {
		t.Errorf("expected message to name the missing method, got %q", r.Message)
	}
}

func TestAssessReviewDynamicFinding(t *testing.T) {
	vf := versionFacts{VersionsOK: true}
	pmf := pmConfigFacts{ConfiguredOK: true, EnabledMethods: []string{"card"}}
	ctx := remediation.Context{Findings: []catalog.Finding{finding(`paymentMethods`)}}

	r := assess(ctx, vf, pmf, nil)
	if r.Status != remediation.StatusReview {
		t.Errorf("expected review, got %q (%s)", r.Status, r.Message)
	}
}

func TestAssessCautionMissingWebhookHandlers(t *testing.T) {
	vf := versionFacts{VersionsOK: true}
	pmf := pmConfigFacts{ConfiguredOK: true, EnabledMethods: []string{"card", "link"}}
	ctx := remediation.Context{Findings: []catalog.Finding{finding(`['card']`)}}
	wh := &codesignals.HandlerSignals{AllPresent: false}

	r := assess(ctx, vf, pmf, wh)
	if r.Status != remediation.StatusCaution {
		t.Errorf("expected caution, got %q (%s)", r.Status, r.Message)
	}
}

func TestAssessReady(t *testing.T) {
	vf := versionFacts{VersionsOK: true}
	pmf := pmConfigFacts{ConfiguredOK: true, EnabledMethods: []string{"card", "link"}}
	ctx := remediation.Context{Findings: []catalog.Finding{finding(`['card']`)}}
	wh := &codesignals.HandlerSignals{AllPresent: true}

	r := assess(ctx, vf, pmf, wh)
	if r.Status != remediation.StatusReady {
		t.Errorf("expected ready, got %q (%s)", r.Status, r.Message)
	}
}

func TestAssessOfferingPolicyConstant(t *testing.T) {
	vf := versionFacts{VersionsOK: true}
	pmf := pmConfigFacts{ConfiguredOK: true, EnabledMethods: []string{"card"}}
	ctx := remediation.Context{Findings: []catalog.Finding{finding(`['card']`)}}

	r := assess(ctx, vf, pmf, nil)
	if got := r.Evidence["offering_policy"]; got != "preserve_existing" {
		t.Errorf("expected offering_policy evidence to be the hardcoded constant, got %v", got)
	}
}

// TestAssessNotesAppended pins that this kind's own hardcoded
// dashboardNotes are surfaced on every Result unconditionally, as their
// own separate Notes — not appended onto Message as a trailing clause,
// and not scoped to individual findings or sourced from rule data
// (there's no "fixes" field anymore).
func TestAssessNotesAppended(t *testing.T) {
	cases := []struct {
		name string
		vf   versionFacts
		pmf  pmConfigFacts
		ctx  remediation.Context
	}{
		{"no events", versionFacts{NoEvents: true}, pmConfigFacts{}, remediation.Context{}},
		{"stale version", versionFacts{}, pmConfigFacts{}, remediation.Context{}},
		{"mixed versions", versionFacts{VersionsMixed: true}, pmConfigFacts{}, remediation.Context{}},
		{"no dashboard config", versionFacts{VersionsOK: true}, pmConfigFacts{ConfiguredOK: false}, remediation.Context{}},
		{"review (dynamic)", versionFacts{VersionsOK: true}, pmConfigFacts{ConfiguredOK: true, EnabledMethods: []string{"card"}}, remediation.Context{Findings: []catalog.Finding{finding(`paymentMethods`)}}},
		{"ready", versionFacts{VersionsOK: true}, pmConfigFacts{ConfiguredOK: true, EnabledMethods: []string{"card"}}, remediation.Context{Findings: []catalog.Finding{finding(`['card']`)}}},
	}
	for _, c := range cases {
		r := assess(c.ctx, c.vf, c.pmf, nil)
		if len(r.Notes) == 0 || !strings.Contains(r.Notes[0], "dashboard.stripe.com") {
			t.Errorf("%s: expected Notes to carry the note, got %v", c.name, r.Notes)
		}
		if strings.Contains(r.Message, "dashboard.stripe.com") {
			t.Errorf("%s: expected the note NOT to be appended onto Message, got %q", c.name, r.Message)
		}
	}
}

// TestLiveCheckEvents pins that this topic's live-check events match its
// static webhookSignals declaration, and that the trigger event is one
// of them.
func TestLiveCheckEvents(t *testing.T) {
	events, trigger := implementation{}.LiveCheckEvents()
	if len(events) != len(webhookSignals.WebhookEvents) {
		t.Fatalf("expected %d events, got %d", len(webhookSignals.WebhookEvents), len(events))
	}
	found := false
	for _, e := range events {
		if e == trigger {
			found = true
		}
	}
	if !found {
		t.Errorf("trigger event %q is not among the listened-for events %v", trigger, events)
	}
}
