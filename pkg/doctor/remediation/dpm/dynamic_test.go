package dpm

import (
	"strings"
	"testing"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/codesignals"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

func finding(value string) catalog.Finding {
	return catalog.Finding{Param: "payment_method_types", Value: value}
}

func TestFindingsVersionFixNoEvents(t *testing.T) {
	got := findingsVersionFix(versionFacts{NoEvents: true})
	if got.Class != "REVIEW" {
		t.Errorf("expected REVIEW, got %q", got.Class)
	}
}

func TestFindingsVersionFixStaleVersion(t *testing.T) {
	got := findingsVersionFix(versionFacts{VersionsOK: false, VersionsMixed: false})
	if got.Class != "CAUTION" {
		t.Errorf("expected CAUTION, got %q", got.Class)
	}
	if !strings.Contains(got.Instruction, "automatic_payment_methods[enabled]=true") {
		t.Errorf("expected the instruction to name the fix, got %q", got.Instruction)
	}
}

func TestFindingsVersionFixMixedVersions(t *testing.T) {
	got := findingsVersionFix(versionFacts{VersionsMixed: true})
	if got.Class != "CAUTION" {
		t.Errorf("expected CAUTION, got %q", got.Class)
	}
}

func TestFindingsVersionFixVersionsOK(t *testing.T) {
	got := findingsVersionFix(versionFacts{VersionsOK: true})
	if got.Class != "" {
		t.Errorf("expected no version-driven fix when versions are OK, got %q", got.Class)
	}
}

func TestClassifyIntent(t *testing.T) {
	cases := map[string]string{
		`['card']`:                  "default-shaped",
		`['card', 'link']`:          "default-shaped",
		`['oxxo']`:                  "deliberate",
		`['card', 'ideal']`:         "static",
		`paymentMethods`:            "dynamic",
		`cond ? ['card'] : ['x']`:   "dynamic",
		`%w[card link]`:             "default-shaped",
		`%w[card ideal bancontact]`: "static",
		`%w(oxxo)`:                  "deliberate",
		`%i[card ideal]`:            "static",
	}
	for value, want := range cases {
		if got := classifyIntent(value); got != want {
			t.Errorf("classifyIntent(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestQuotedMethodsPercentArray(t *testing.T) {
	got := quotedMethods(`%w[card ideal bancontact]`)
	want := []string{"card", "ideal", "bancontact"}
	if len(got) != len(want) {
		t.Fatalf("quotedMethods(%%w[...]) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("quotedMethods(%%w[...])[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIntentVerdictStaticNamesExcludedPaymentMethodTypes(t *testing.T) {
	got := intentVerdict("static")
	if !strings.Contains(got, "excluded_payment_method_types") {
		t.Errorf("expected the static bucket to name excluded_payment_method_types, got %q", got)
	}
}

func TestHardcodedListsFindingVersionFix(t *testing.T) {
	ctx := remediation.Context{Dir: ".", Findings: []catalog.Finding{
		{File: "a.rb", Line: 1, Param: "payment_method_types", Value: `['card']`},
	}}

	f := hardcodedListsFinding(ctx, versionFacts{NoEvents: true})
	if f.ID != "explicit_payment_method_lists" {
		t.Errorf("expected id explicit_payment_method_lists, got %q", f.ID)
	}
	if len(f.Items) != 1 || f.Items[0].Body != "Current list: ['card']" {
		t.Errorf("expected one item naming the current list, got %+v", f.Items)
	}
	if f.SuggestedParameter == nil || f.SuggestedParameter.Name != "automatic_payment_methods[enabled]" {
		t.Errorf("expected a suggested parameter, got %+v", f.SuggestedParameter)
	}
}

func TestHardcodedListsFindingNoVersionFix(t *testing.T) {
	ctx := remediation.Context{Dir: ".", Findings: []catalog.Finding{
		{File: "a.rb", Line: 1, Param: "payment_method_types", Value: `['card']`},
		{File: "b.rb", Line: 2, Param: "payment_method_types", Value: `paymentMethods`},
	}}

	f := hardcodedListsFinding(ctx, versionFacts{VersionsOK: true})
	if f.SuggestedParameter != nil {
		t.Errorf("expected no suggested parameter when versions are OK, got %+v", f.SuggestedParameter)
	}
	if len(f.Items) != 2 || !strings.HasPrefix(f.Items[0].Body, "CANDIDATE:") {
		t.Errorf("expected a.rb:1's own CANDIDATE verdict, got %+v", f.Items)
	}
	if !strings.HasPrefix(f.Items[1].Body, "REVIEW:") {
		t.Errorf("expected b.rb:2's own REVIEW verdict, got %+v", f.Items)
	}
}

func TestHardcodedListsFindingRuntimeNote(t *testing.T) {
	withDynamic := remediation.Context{Dir: ".", Findings: []catalog.Finding{
		{File: "a.rb", Line: 1, Param: "payment_method_types", Value: `paymentMethods`},
	}}
	f := hardcodedListsFinding(withDynamic, versionFacts{VersionsOK: true})
	if !strings.Contains(strings.Join(f.Body, " "), "built at runtime") {
		t.Errorf("expected the runtime-routing note in Body, got %+v", f.Body)
	}

	withoutDynamic := remediation.Context{Dir: ".", Findings: []catalog.Finding{
		{File: "a.rb", Line: 1, Param: "payment_method_types", Value: `['card']`},
	}}
	f = hardcodedListsFinding(withoutDynamic, versionFacts{VersionsOK: true})
	if strings.Contains(strings.Join(f.Body, " "), "built at runtime") {
		t.Errorf("expected no runtime-routing note without a dynamic finding, got %+v", f.Body)
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

func TestAssessReadyDespiteDynamicFinding(t *testing.T) {
	vf := versionFacts{VersionsOK: true}
	pmf := pmConfigFacts{ConfiguredOK: true, EnabledMethods: []string{"card"}}
	ctx := remediation.Context{Findings: []catalog.Finding{finding(`paymentMethods`)}}

	r := assess(ctx, vf, pmf, nil)
	if r.Status != remediation.StatusReady {
		t.Errorf("expected ready, got %q (%s)", r.Status, r.Message)
	}
}

func TestAssessReadyDespiteMissingWebhookHandlers(t *testing.T) {
	vf := versionFacts{VersionsOK: true}
	pmf := pmConfigFacts{ConfiguredOK: true, EnabledMethods: []string{"card", "link"}}
	ctx := remediation.Context{Findings: []catalog.Finding{finding(`['card']`)}}
	wh := &codesignals.HandlerSignals{AllPresent: false}

	r := assess(ctx, vf, pmf, wh)
	if r.Status != remediation.StatusReady {
		t.Errorf("expected ready, got %q (%s)", r.Status, r.Message)
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
	if !strings.Contains(r.Message, "card, link") {
		t.Errorf("expected the message to list the Dashboard's enabled methods, got %q", r.Message)
	}
	if strings.Contains(r.Message, "default payment method configuration") {
		t.Errorf("expected no default-config mention when the chosen config isn't the default, got %q", r.Message)
	}
}

func TestAssessReadyNamesDefaultConfig(t *testing.T) {
	vf := versionFacts{VersionsOK: true}
	pmf := pmConfigFacts{ConfiguredOK: true, EnabledMethods: []string{"card"}, ActiveConfigIsDefault: true}
	ctx := remediation.Context{Findings: []catalog.Finding{finding(`['card']`)}}

	r := assess(ctx, vf, pmf, nil)
	if !strings.Contains(r.Message, "your default payment method configuration") {
		t.Errorf("expected the message to name the default config, got %q", r.Message)
	}
}

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
