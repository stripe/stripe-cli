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
// a uniform FindingsVerdict override instead. See findingsVerdict.

func TestFindingsVerdictNoEvents(t *testing.T) {
	got := findingsVerdict(versionFacts{NoEvents: true})
	if !strings.HasPrefix(got, "REVIEW:") {
		t.Errorf("expected REVIEW, got %q", got)
	}
}

func TestFindingsVerdictStaleVersion(t *testing.T) {
	got := findingsVerdict(versionFacts{VersionsOK: false, VersionsMixed: false})
	// CAUTION, not BLOCKED: the fix is known and actionable (add
	// automatic_payment_methods[enabled]=true), not a dead end.
	if !strings.HasPrefix(got, "CAUTION:") {
		t.Errorf("expected CAUTION, got %q", got)
	}
	if !strings.Contains(got, "automatic_payment_methods[enabled]=true") {
		t.Errorf("expected the message to name the fix, got %q", got)
	}
}

func TestFindingsVerdictMixedVersions(t *testing.T) {
	got := findingsVerdict(versionFacts{VersionsMixed: true})
	if !strings.HasPrefix(got, "CAUTION:") {
		t.Errorf("expected CAUTION, got %q", got)
	}
}

func TestFindingsVerdictVersionsOK(t *testing.T) {
	got := findingsVerdict(versionFacts{VersionsOK: true})
	if got != "" {
		t.Errorf("expected no override when versions are OK, got %q", got)
	}
}

func TestAssessCarriesFindingsVerdict(t *testing.T) {
	r := assess(remediation.Context{}, versionFacts{NoEvents: true}, pmConfigFacts{}, nil)
	if !strings.HasPrefix(r.FindingsVerdict, "REVIEW:") {
		t.Errorf("expected assess to carry the FindingsVerdict override, got %q", r.FindingsVerdict)
	}
	// The Dashboard Result itself is about Dashboard readiness only, not
	// version safety.
	if strings.Contains(r.Message, "API version") {
		t.Errorf("Dashboard message should no longer mention API version, got %q", r.Message)
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
