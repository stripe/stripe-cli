package doctor

// fix_test.go — the version-forked companion behavior end to end: insert
// branch (below-cutoff / unknown accounts), omit branch (at/after cutoff),
// resource gating, idempotence, and whitespace-clean removals.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func copyFixtures(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		src, err := os.ReadFile(filepath.Join("testdata", n))
		if err != nil {
			t.Fatalf("read fixture %s: %v", n, err)
		}
		if err := os.WriteFile(filepath.Join(dir, n), src, 0o644); err != nil {
			t.Fatalf("copy fixture %s: %v", n, err)
		}
	}
	return dir
}

func mustRead(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

var wsOnlyLine = regexp.MustCompile(`(?m)^[ \t]+$`)

func TestCompanionInsertLanguages(t *testing.T) {
	dir := copyFixtures(t, "pi.py", "var_indirect.py", "pi.rb", "pi.js", "pi.php", "pi.go", "Pi.cs", "Pi.java", "sub.rb")
	dec := &companionDecision{insert: true, reason: "test: below cutoff"}
	rep, err := fixRun(dir, dpmRule, true, false, dec)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.AllClean {
		t.Fatalf("expected every reparse clean, got %+v", rep.Files)
	}
	if rep.Companion == nil || rep.Companion.Mode != "insert" || rep.Companion.Inserts == 0 {
		t.Fatalf("expected companion insert with >0 inserts, got %+v", rep.Companion)
	}

	want := map[string]string{
		"pi.py":           `automatic_payment_methods={"enabled": True}`,
		"var_indirect.py": `"automatic_payment_methods": {"enabled": True}`,
		"pi.rb":           "automatic_payment_methods: {enabled: true}",
		"pi.js":           "automatic_payment_methods: {enabled: true}",
		"pi.php":          "'automatic_payment_methods' => ['enabled' => true]",
		"pi.go":           "AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{Enabled: stripe.Bool(true)}",
		"Pi.cs":           "AutomaticPaymentMethods = new PaymentIntentAutomaticPaymentMethodsOptions { Enabled = true }",
		"Pi.java":         ".setAutomaticPaymentMethods(PaymentIntentCreateParams.AutomaticPaymentMethods.builder().setEnabled(true).build())",
	}
	for name, ins := range want {
		if out := mustRead(t, dir, name); !strings.Contains(out, ins) {
			t.Errorf("%s must gain the companion %q; got:\n%s", name, ins, out)
		}
	}

	// Nested payment_settings.payment_method_types (subscriptions) is removed
	// but must NOT gain the companion — the parameter does not exist there.
	if sub := mustRead(t, dir, "sub.rb"); strings.Contains(sub, "automatic_payment_methods") {
		t.Errorf("sub.rb must not gain the companion:\n%s", sub)
	}

	// The migration is complete: a rescan finds nothing.
	findings, _, _, err := scan(dir, dpmRule)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("rescan after apply must be clean, got %d findings", len(findings))
	}
}

func TestCompanionOmittedIsPlainRemoval(t *testing.T) {
	dir := copyFixtures(t, "pi.py", "pi.php")
	dec := &companionDecision{insert: false, reason: "test: at/after cutoff"}
	rep, err := fixRun(dir, dpmRule, true, false, dec)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.AllClean {
		t.Fatal("expected all reparse clean")
	}
	if rep.Companion == nil || rep.Companion.Mode != "omit" || rep.Companion.Inserts != 0 {
		t.Fatalf("expected companion omit with 0 inserts, got %+v", rep.Companion)
	}
	for _, name := range []string{"pi.py", "pi.php"} {
		out := mustRead(t, dir, name)
		if strings.Contains(out, "automatic_payment_methods") {
			t.Errorf("%s: omit branch must not insert:\n%s", name, out)
		}
		if wsOnlyLine.MatchString(out) {
			t.Errorf("%s: removal must not leave whitespace-only lines:\n%s", name, out)
		}
	}
}

func TestCompanionResourceGate(t *testing.T) {
	// Checkout Sessions have no automatic_payment_methods parameter: the
	// removal must stay a removal even on the insert branch.
	dir := t.TempDir()
	src := `const stripe = require('stripe')('sk_test_x');
await stripe.checkout.sessions.create({
  mode: 'payment',
  payment_method_types: ['card', 'ideal'],
  success_url: 'https://example.com',
});
`
	if err := os.WriteFile(filepath.Join(dir, "sessions.js"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.AllClean {
		t.Fatal("expected reparse clean")
	}
	out := mustRead(t, dir, "sessions.js")
	if strings.Contains(out, "payment_method_types") {
		t.Errorf("param must be removed:\n%s", out)
	}
	if strings.Contains(out, "automatic_payment_methods") {
		t.Errorf("checkout sessions must not gain the companion:\n%s", out)
	}
}

func TestCompanionIdempotent(t *testing.T) {
	// A call that already sets automatic_payment_methods must not gain a
	// second one; the stale payment_method_types is still removed.
	dir := t.TempDir()
	src := `const stripe = require('stripe')('sk_test_x');
await stripe.paymentIntents.create({
  amount: 1099,
  currency: 'eur',
  payment_method_types: ['card', 'ideal'],
  automatic_payment_methods: {enabled: true},
});
`
	if err := os.WriteFile(filepath.Join(dir, "pi.js"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.AllClean {
		t.Fatal("expected reparse clean")
	}
	out := mustRead(t, dir, "pi.js")
	if strings.Contains(out, "payment_method_types") {
		t.Errorf("param must be removed:\n%s", out)
	}
	if n := strings.Count(out, "automatic_payment_methods"); n != 1 {
		t.Errorf("expected exactly one companion, got %d:\n%s", n, out)
	}
}

func TestCompanionJavaSingleInsertPerBuilder(t *testing.T) {
	// Two addPaymentMethodType statements on ONE builder must yield exactly
	// one setAutomaticPaymentMethods, or the call would set it twice.
	dir := t.TempDir()
	src := `import com.stripe.param.PaymentIntentCreateParams;

class Demo {
  void go() {
    PaymentIntentCreateParams.Builder paramsBuilder = PaymentIntentCreateParams.builder();
    paramsBuilder.setAmount(1099L);
    paramsBuilder.addPaymentMethodType("card");
    paramsBuilder.addPaymentMethodType("link");
    paramsBuilder.build();
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "Server.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.AllClean {
		t.Fatal("expected reparse clean")
	}
	out := mustRead(t, dir, "Server.java")
	if strings.Contains(out, "addPaymentMethodType") {
		t.Errorf("both statements must be handled:\n%s", out)
	}
	if n := strings.Count(out, "setAutomaticPaymentMethods"); n != 1 {
		t.Errorf("expected exactly one companion per builder, got %d:\n%s", n, out)
	}
	if wsOnlyLine.MatchString(out) {
		t.Errorf("statement removal must not leave whitespace-only lines:\n%s", out)
	}
}

func TestCompanionDistinctBuildersEachGetInsert(t *testing.T) {
	// Review finding: the dedupe key must identify a builder INSTANCE. Two
	// chains in one file, and same-named builders in different methods, must
	// each get their own companion.
	dir := t.TempDir()
	chains := `import com.stripe.param.PaymentIntentCreateParams;
import com.stripe.param.SetupIntentCreateParams;

class Demo {
  void a() {
    PaymentIntentCreateParams p = PaymentIntentCreateParams.builder().setAmount(1099L).addPaymentMethodType("card").build();
  }
  void b() {
    SetupIntentCreateParams s = SetupIntentCreateParams.builder().addPaymentMethodType("card").build();
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "Chains.java"), []byte(chains), 0o644); err != nil {
		t.Fatal(err)
	}
	stmts := `import com.stripe.param.PaymentIntentCreateParams;

class Demo {
  void a() {
    PaymentIntentCreateParams.Builder paramsBuilder = PaymentIntentCreateParams.builder();
    paramsBuilder.addPaymentMethodType("card");
    paramsBuilder.build();
  }
  void b() {
    PaymentIntentCreateParams.Builder paramsBuilder = PaymentIntentCreateParams.builder();
    paramsBuilder.addPaymentMethodType("card");
    paramsBuilder.build();
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "Stmts.java"), []byte(stmts), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.AllClean {
		t.Fatal("expected reparse clean")
	}
	chainsOut := mustRead(t, dir, "Chains.java")
	if n := strings.Count(chainsOut, "setAutomaticPaymentMethods"); n != 2 {
		t.Errorf("two distinct chains must each get a companion, got %d:\n%s", n, chainsOut)
	}
	if !strings.Contains(chainsOut, "SetupIntentCreateParams.AutomaticPaymentMethods") {
		t.Errorf("the SetupIntent chain must use the SetupIntent params class:\n%s", chainsOut)
	}
	stmtsOut := mustRead(t, dir, "Stmts.java")
	if n := strings.Count(stmtsOut, "setAutomaticPaymentMethods"); n != 2 {
		t.Errorf("same-named builders in different methods must each get a companion, got %d:\n%s", n, stmtsOut)
	}
}

func TestCompanionHalfMigratedFileStillInserts(t *testing.T) {
	// Review finding: the already-present check must be per call site, not
	// per file — a half-migrated file's remaining call still needs the insert.
	dir := t.TempDir()
	src := `const stripe = require('stripe')('sk_test_x');
async function a() {
  await stripe.paymentIntents.create({
    amount: 1099,
    currency: 'eur',
    automatic_payment_methods: {enabled: true},
  });
}
async function b() {
  await stripe.paymentIntents.create({
    amount: 2099,
    currency: 'eur',
    payment_method_types: ['card', 'link'],
  });
}
`
	if err := os.WriteFile(filepath.Join(dir, "half.js"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.AllClean {
		t.Fatal("expected reparse clean")
	}
	if rep.Companion.Inserts != 1 {
		t.Errorf("the un-migrated call must get the insert, got %d", rep.Companion.Inserts)
	}
	out := mustRead(t, dir, "half.js")
	if n := strings.Count(out, "automatic_payment_methods"); n != 2 {
		t.Errorf("expected both calls to carry the companion, got %d:\n%s", n, out)
	}
}

func TestCompanionCreateOnlyOperations(t *testing.T) {
	// Review finding: automatic_payment_methods is CREATE-only — update,
	// confirm, and modify sites must be bare-removed, never companioned.
	dir := t.TempDir()
	files := map[string]string{
		"update.js": `const stripe = require('stripe')('sk_test_x');
await stripe.paymentIntents.update('pi_123', {
  payment_method_types: ['card', 'ideal'],
});
`,
		"confirm.py": `import stripe

stripe.PaymentIntent.confirm(
    "pi_123",
    payment_method_types=["card", "ideal"],
)
`,
		"update.go": `package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	params := &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card", "ideal"}),
	}
	pi, _ := paymentintent.Update("pi_123", params)
	_ = pi
}
`,
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.AllClean {
		t.Fatal("expected reparse clean")
	}
	if rep.Companion.Inserts != 0 {
		t.Errorf("update/confirm sites must never gain the companion, got %d inserts", rep.Companion.Inserts)
	}
	for name := range files {
		out := mustRead(t, dir, name)
		if strings.Contains(out, "payment_method_types") || strings.Contains(out, "PaymentMethodTypes") {
			t.Errorf("%s: param must still be removed:\n%s", name, out)
		}
		if strings.Contains(out, "automatic_payment_methods") || strings.Contains(out, "AutomaticPaymentMethods") {
			t.Errorf("%s: create-only companion must not appear:\n%s", name, out)
		}
	}
}

func TestCompanionJavaSessionBuilderNotMatched(t *testing.T) {
	// Review finding: a Checkout Session builder mentioning
	// setPaymentIntentData must not be mistaken for a PaymentIntent create.
	dir := t.TempDir()
	src := `import com.stripe.param.checkout.SessionCreateParams;

class Demo {
  void go() {
    SessionCreateParams params = SessionCreateParams.builder().addPaymentMethodType("card").setPaymentIntentData(SessionCreateParams.PaymentIntentData.builder().build()).build();
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "Sess.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Companion.Inserts != 0 {
		t.Errorf("session builder must not gain the companion, got %d inserts", rep.Companion.Inserts)
	}
	out := mustRead(t, dir, "Sess.java")
	if strings.Contains(out, "AutomaticPaymentMethods") {
		t.Errorf("no companion belongs in a Session builder:\n%s", out)
	}
}

func TestDecideCompanionUsesRuleCutoffAndConfigNote(t *testing.T) {
	// Review findings: the fork must be computed against THIS rule's cutoff
	// (not the dpm package constant), and must surface the missing-Dashboard-
	// config caveat the doctor raises on the same facts.
	rule := dpmRule
	rule.IntroducedIn = "2024-01-01"
	facts := &accountFacts{
		EventVersions: map[string]int{"2023-10-16": 5},
		OldestVersion: "2023-10-16",
		ConfiguredOK:  true,
	}
	// 2023-10-16 is >= the dpm constant but < this rule's cutoff: must insert.
	if dec := decideCompanion(rule, facts); !dec.insert {
		t.Errorf("traffic below the RULE's cutoff must insert, got %+v", dec)
	}
	facts.EventVersions = map[string]int{"2024-06-20": 3}
	facts.OldestVersion = "2024-06-20"
	if dec := decideCompanion(rule, facts); dec.insert {
		t.Errorf("traffic at/after the rule's cutoff must omit, got %+v", dec)
	}
	facts.ConfiguredOK = false
	if dec := decideCompanion(rule, facts); !strings.Contains(dec.reason, "no active Dashboard") {
		t.Errorf("missing Dashboard config must be surfaced in the reason, got %q", dec.reason)
	}
}

func TestCompanionConfirmSites(t *testing.T) {
	// DPM-lead feedback: APM + confirm:true without a return_url is a
	// runtime 400. Sites like that must get allow_redirects:"never" (or the
	// merchant's --return-url); sites that already pass return_url stay plain.
	write := func(t *testing.T, dir, name, src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	confirmJS := `const stripe = require('stripe')('sk_test_x');
await stripe.paymentIntents.create({
  amount: 1099,
  currency: 'eur',
  payment_method_types: ['card'],
  confirm: true,
  payment_method: 'pm_x',
});
`
	confirmWithURLJS := `const stripe = require('stripe')('sk_test_x');
await stripe.paymentIntents.create({
  amount: 1099,
  currency: 'eur',
  payment_method_types: ['card'],
  confirm: true,
  return_url: 'https://example.com/return',
});
`
	confirmJava := `import com.stripe.param.PaymentIntentCreateParams;

class Demo {
  void go() {
    PaymentIntentCreateParams params = PaymentIntentCreateParams.builder().setAmount(1099L).addPaymentMethodType("card").setConfirm(true).build();
  }
}
`

	t.Run("no return_url pins allow_redirects never", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "confirm.js", confirmJS)
		write(t, dir, "Confirm.java", confirmJava)
		rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
		if err != nil {
			t.Fatal(err)
		}
		if !rep.AllClean {
			t.Fatal("expected reparse clean")
		}
		if out := mustRead(t, dir, "confirm.js"); !strings.Contains(out, "allow_redirects: 'never'") {
			t.Errorf("confirm site must pin allow_redirects:\n%s", out)
		}
		if out := mustRead(t, dir, "Confirm.java"); !strings.Contains(out, "AllowRedirects.NEVER") {
			t.Errorf("java confirm site must pin AllowRedirects.NEVER:\n%s", out)
		}
		if len(rep.Companion.Notes) == 0 || !strings.Contains(rep.Companion.Notes[0], "allow_redirects") {
			t.Errorf("notes must surface the pinned sites, got %v", rep.Companion.Notes)
		}
	})

	t.Run("--return-url adds return_url instead", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "confirm.js", confirmJS)
		rep, err := fixRun(dir, dpmRule, true, false,
			&companionDecision{insert: true, reason: "test", returnURL: "https://example.com/complete"})
		if err != nil {
			t.Fatal(err)
		}
		out := mustRead(t, dir, "confirm.js")
		if !strings.Contains(out, "return_url: 'https://example.com/complete'") {
			t.Errorf("provided return_url must be inserted:\n%s", out)
		}
		if strings.Contains(out, "allow_redirects") {
			t.Errorf("with a return_url the redirects pin must not appear:\n%s", out)
		}
		if len(rep.Companion.Notes) == 0 || !strings.Contains(rep.Companion.Notes[0], "return_url") {
			t.Errorf("notes must record the return_url insertions, got %v", rep.Companion.Notes)
		}
	})

	t.Run("existing return_url stays plain", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "confirm.js", confirmWithURLJS)
		rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
		if err != nil {
			t.Fatal(err)
		}
		out := mustRead(t, dir, "confirm.js")
		if !strings.Contains(out, "automatic_payment_methods: {enabled: true}") || strings.Contains(out, "allow_redirects") {
			t.Errorf("return_url already present: plain companion expected:\n%s", out)
		}
		if len(rep.Companion.Notes) != 0 {
			t.Errorf("no notes expected, got %v", rep.Companion.Notes)
		}
	})

	t.Run("non-confirm create stays plain", func(t *testing.T) {
		dir := copyFixtures(t, "pi.js")
		_, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
		if err != nil {
			t.Fatal(err)
		}
		if out := mustRead(t, dir, "pi.js"); strings.Contains(out, "allow_redirects") {
			t.Errorf("plain create must not gain allow_redirects:\n%s", out)
		}
	})
}

func TestConfirmSiteHandledOnOmitBranch(t *testing.T) {
	// Round-2 review, finding A: the confirm guard must run on BOTH branches.
	// Post-cutoff accounts (insert:false) plain-removing at a confirm site
	// leaves APM default-on + confirm + no return_url = runtime 400.
	dir := t.TempDir()
	src := `const stripe = require('stripe')('sk_test_x');
await stripe.paymentIntents.create({
  amount: 1099,
  currency: 'usd',
  payment_method_types: ['card'],
  confirm: true,
  payment_method: 'pm_x',
});
`
	if err := os.WriteFile(filepath.Join(dir, "confirm.js"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: false, reason: "test: at/after cutoff"})
	if err != nil {
		t.Fatal(err)
	}
	out := mustRead(t, dir, "confirm.js")
	if !strings.Contains(out, "allow_redirects: 'never'") {
		t.Errorf("omit branch must still pin the confirm site:\n%s", out)
	}
	if len(rep.Companion.PinnedSites) != 1 {
		t.Errorf("pinned_sites must carry the site, got %+v", rep.Companion.PinnedSites)
	}
}

func TestConfirmSiteWithRedirectMethodsIsGated(t *testing.T) {
	// Round-2 review, finding C: pinning allow_redirects:"never" when the
	// removed list named redirect methods silently drops them. Gate instead.
	dir := t.TempDir()
	src := `const stripe = require('stripe')('sk_test_x');
await stripe.paymentIntents.create({
  amount: 1099,
  currency: 'eur',
  payment_method_types: ['card', 'ideal'],
  confirm: true,
});
`
	if err := os.WriteFile(filepath.Join(dir, "confirm.js"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, insert := range []bool{true, false} {
		rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: insert, reason: "test"})
		if err != nil {
			t.Fatal(err)
		}
		out := mustRead(t, dir, "confirm.js")
		if !strings.Contains(out, "payment_method_types") {
			t.Errorf("insert=%v: gated site must remain untouched:\n%s", insert, out)
		}
		found := false
		for _, sk := range rep.Skipped {
			if sk.Intent == "confirm-redirect" && strings.Contains(sk.Reason, "ideal") {
				found = true
			}
		}
		if !found {
			t.Errorf("insert=%v: expected a confirm-redirect skip naming ideal, got %+v", insert, rep.Skipped)
		}
	}
	// With a return_url the same site migrates and keeps redirect methods.
	rep, err := fixRun(dir, dpmRule, true, false,
		&companionDecision{insert: false, reason: "test", returnURL: "https://example.com/done"})
	if err != nil {
		t.Fatal(err)
	}
	out := mustRead(t, dir, "confirm.js")
	if !strings.Contains(out, "return_url: 'https://example.com/done'") || strings.Contains(out, "allow_redirects") {
		t.Errorf("with return_url the site must migrate un-pinned:\n%s", out)
	}
	if len(rep.Skipped) != 0 {
		t.Errorf("nothing should be gated with a return_url, got %+v", rep.Skipped)
	}
}

func TestConfirmDetectionIsBagScoped(t *testing.T) {
	// Round-2 review, finding B: one confirming create must not pin its
	// siblings, and confirm-lookalike keys must not trigger at all.
	dir := t.TempDir()
	src := `const stripe = require('stripe')('sk_test_x');
async function checkout() {
  await stripe.paymentIntents.create({
    amount: 1099,
    currency: 'usd',
    payment_method_types: ['card'],
  });
  await stripe.paymentIntents.create({
    amount: 2099,
    currency: 'usd',
    payment_method_types: ['card'],
    confirm: true,
  });
}
async function audit() {
  const auditOpts = { reconfirm: true, notify_confirmation: true };
  await stripe.paymentIntents.create({
    amount: 3099,
    currency: 'usd',
    payment_method_types: ['card'],
    metadata: auditOpts,
  });
}
`
	if err := os.WriteFile(filepath.Join(dir, "multi.js"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.AllClean {
		t.Fatal("expected reparse clean")
	}
	out := mustRead(t, dir, "multi.js")
	if n := strings.Count(out, "allow_redirects"); n != 1 {
		t.Errorf("exactly the confirming create gets pinned, got %d pins:\n%s", n, out)
	}
	if n := strings.Count(out, "automatic_payment_methods"); n != 3 {
		t.Errorf("all three creates still get the companion, got %d:\n%s", n, out)
	}
}

func TestJavaForeignSetReturnUrlDoesNotSuppressPin(t *testing.T) {
	// Round-2 review, finding B (Java): another builder's setReturnUrl in the
	// same method must not convince us THIS builder has one.
	dir := t.TempDir()
	src := `import com.stripe.param.PaymentIntentCreateParams;
import com.stripe.param.SetupIntentCreateParams;

class Demo {
  void go() {
    PaymentIntentCreateParams.Builder other = PaymentIntentCreateParams.builder();
    other.setReturnUrl("https://example.com/other");
    other.setConfirm(true);
    other.build();
    SetupIntentCreateParams.Builder mine = SetupIntentCreateParams.builder();
    mine.addPaymentMethodType("card");
    mine.setConfirm(true);
    mine.build();
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "Two.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	out := mustRead(t, dir, "Two.java")
	if !strings.Contains(out, "AllowRedirects.NEVER") {
		t.Errorf("mine has confirm and NO return_url — the foreign setReturnUrl must not suppress the pin:\n%s", out)
	}
}

func TestValidateReturnURL(t *testing.T) {
	// Round-2 review, finding F: the URL is spliced into source verbatim.
	for _, bad := range []string{
		"ftp://example.com/x", "https://", "example.com/no-scheme",
		"https://ex.com/l'orange", `https://ex.com/a"b`, "https://ex.com/a b",
		"https://ex.com/x', extra_param: 'y",
	} {
		if err := validateReturnURL(bad); err == nil {
			t.Errorf("expected rejection of %q", bad)
		}
	}
	for _, good := range []string{"https://example.com/complete", "http://localhost:4242/return?x=1"} {
		if err := validateReturnURL(good); err != nil {
			t.Errorf("expected %q accepted, got %v", good, err)
		}
	}
}

func TestScanVersionPins(t *testing.T) {
	// Round-2 review, finding E1: a Stripe-Version pinned in code below the
	// cutoff overrides an omit census (events only show the account default).
	dir := t.TempDir()
	files := map[string]string{
		"client.js": "const stripe = require('stripe')('sk_test_x', {apiVersion: '2022-11-15'});\n",
		"api.py":    "import stripe\nstripe.api_version = \"2020-08-27\"\n",
		"clean.rb":  "require 'stripe'\n",
	}
	for n, s := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	oldest, at := scanVersionPins(dir)
	if oldest != "2020-08-27" || !strings.Contains(at, "api.py") {
		t.Errorf("expected oldest pin 2020-08-27 in api.py, got %q at %q", oldest, at)
	}
	// The offline decision carries the pin evidence in its reason.
	dec := resolveCompanion(dpmRule, nil, "", dir, true)
	if !dec.insert || !strings.Contains(dec.reason, "2020-08-27") {
		t.Errorf("offline decision must cite the code pin, got %+v", dec)
	}
}

func TestJavaMixedListJudgedAsOneSite(t *testing.T) {
	// Round-3 verification F1: Java splits a method list across findings;
	// gates must judge the BUILDER, never a single link — a partial removal
	// leaves payment_method_types beside the companion, which the API rejects.
	confirmMixed := `import com.stripe.param.PaymentIntentCreateParams;

class Demo {
  void go() {
    PaymentIntentCreateParams params = PaymentIntentCreateParams.builder().setAmount(1099L).setConfirm(true).addPaymentMethodType("card").addPaymentMethodType("ideal").build();
  }
}
`
	for _, all := range []bool{false, true} {
		for _, insert := range []bool{false, true} {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "Mixed.java"), []byte(confirmMixed), 0o644); err != nil {
				t.Fatal(err)
			}
			rep, err := fixRun(dir, dpmRule, true, all, &companionDecision{insert: insert, reason: "test"})
			if err != nil {
				t.Fatal(err)
			}
			out := mustRead(t, dir, "Mixed.java")
			if n := strings.Count(out, "addPaymentMethodType"); n != 2 {
				t.Errorf("all=%v insert=%v: gated builder must be fully untouched, %d adds remain:\n%s", all, insert, n, out)
			}
			if strings.Contains(out, "AutomaticPaymentMethods") {
				t.Errorf("all=%v insert=%v: no companion may coexist with the remaining list:\n%s", all, insert, out)
			}
			gated := false
			for _, sk := range rep.Skipped {
				if sk.Intent == "confirm-redirect" && strings.Contains(sk.Reason, "ideal") {
					gated = true
				}
			}
			if !gated {
				t.Errorf("all=%v insert=%v: expected one confirm-redirect skip for the whole builder, got %+v", all, insert, rep.Skipped)
			}
		}
	}

	// Without confirm, a mixed safe list migrates coherently: every link
	// removed, exactly one plain companion.
	noConfirm := `import com.stripe.param.PaymentIntentCreateParams;

class Demo {
  void go() {
    PaymentIntentCreateParams params = PaymentIntentCreateParams.builder().setAmount(1099L).addPaymentMethodType("card").addPaymentMethodType("oxxo").build();
  }
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Safe.java"), []byte(noConfirm), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"}); err != nil {
		t.Fatal(err)
	}
	out := mustRead(t, dir, "Safe.java")
	if strings.Contains(out, "addPaymentMethodType") || strings.Count(out, "setAutomaticPaymentMethods") != 1 {
		t.Errorf("safe mixed list must fully migrate with one companion:\n%s", out)
	}
}

func TestVarBagConfirmMutationDetected(t *testing.T) {
	// Round-3 verification F2: confirm set on the bag VARIABLE (outside the
	// literal) must be attributed to this site via its var: resolution.
	jsSrc := `const stripe = require('stripe')('sk_test_x');
const params = {
  amount: 1099,
  currency: 'usd',
  payment_method_types: ['card'],
};
params.confirm = true;
await stripe.paymentIntents.create(params);
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "outside.js"), []byte(jsSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: false, reason: "test: omit"})
	if err != nil {
		t.Fatal(err)
	}
	out := mustRead(t, dir, "outside.js")
	if !strings.Contains(out, "allow_redirects: 'never'") {
		t.Errorf("var-mutated confirm must pin (omit branch too):\n%s", out)
	}
	if len(rep.Companion.PinnedSites) != 1 {
		t.Errorf("pinned_sites must record it, got %+v", rep.Companion.PinnedSites)
	}

	// The same shape with a redirect method gates instead.
	jsRedirect := strings.Replace(jsSrc, "['card']", "['card', 'ideal']", 1)
	dir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir2, "outside.js"), []byte(jsRedirect), 0o644); err != nil {
		t.Fatal(err)
	}
	rep2, err := fixRun(dir2, dpmRule, true, false, &companionDecision{insert: true, reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if out := mustRead(t, dir2, "outside.js"); !strings.Contains(out, "payment_method_types") {
		t.Errorf("redirect-method var-confirm site must be gated:\n%s", out)
	}
	if len(rep2.Skipped) != 1 || rep2.Skipped[0].Intent != "confirm-redirect" {
		t.Errorf("expected confirm-redirect skip, got %+v", rep2.Skipped)
	}

	// And a var-assigned return_url resolves the confirm site plainly.
	jsWithURL := strings.Replace(jsSrc, "params.confirm = true;",
		"params.confirm = true;\nparams.return_url = 'https://example.com/r';", 1)
	dir3 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir3, "outside.js"), []byte(jsWithURL), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fixRun(dir3, dpmRule, true, false, &companionDecision{insert: true, reason: "test"}); err != nil {
		t.Fatal(err)
	}
	if out := mustRead(t, dir3, "outside.js"); strings.Contains(out, "allow_redirects") {
		t.Errorf("var-assigned return_url must suppress the pin:\n%s", out)
	}
}

func TestReceiverSuffixCollision(t *testing.T) {
	// Round-3 verification F3/NEW-1: receiver token matching must be
	// word-bounded — `resp.` is not `p.`, `sb.` is not `b.`.
	src := `import com.stripe.param.PaymentIntentCreateParams;
import com.stripe.param.SetupIntentCreateParams;

class Demo {
  void go() {
    PaymentIntentCreateParams.Builder p = PaymentIntentCreateParams.builder();
    p.setConfirm(true);
    p.addPaymentMethodType("card");
    SetupIntentCreateParams.Builder resp = SetupIntentCreateParams.builder();
    resp.setReturnUrl("https://example.com/other");
    resp.build();
    p.build();
  }
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Collide.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"}); err != nil {
		t.Fatal(err)
	}
	out := mustRead(t, dir, "Collide.java")
	if !strings.Contains(out, "AllowRedirects.NEVER") {
		t.Errorf("resp.setReturnUrl must not satisfy p's return_url check — pin required:\n%s", out)
	}
}

func TestWrappedSetConfirmDetected(t *testing.T) {
	// Round-3 verification NEW-2: formatter-wrapped arguments must not hide
	// the confirm (receiver scope splits on ';', not newlines).
	src := `import com.stripe.param.PaymentIntentCreateParams;

class Demo {
  void go() {
    PaymentIntentCreateParams.Builder p = PaymentIntentCreateParams.builder();
    p.setConfirm(
        true);
    p.addPaymentMethodType("card");
    p.build();
  }
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Wrapped.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: true, reason: "test"}); err != nil {
		t.Fatal(err)
	}
	if out := mustRead(t, dir, "Wrapped.java"); !strings.Contains(out, "AllowRedirects.NEVER") {
		t.Errorf("wrapped setConfirm(true) must still pin:\n%s", out)
	}
}

func TestScanVersionPinsIgnoresForeignAPIs(t *testing.T) {
	// Round-3 verification E1: only lines naming Stripe count — AWS, GitHub,
	// Azure, and prose all pin their own date-shaped API versions.
	dir := t.TempDir()
	files := map[string]string{
		"aws.js":     "const s3 = new AWS.S3({apiVersion: '2006-03-01'});\n",
		"gh.py":      "headers = {\"X-GitHub-Api-Version\": \"2022-11-28\"}\n",
		"azure.js":   "const u = base + '?api-version=2023-05-15';\n",
		"comment.rb": "# we upgraded our api version on 2019-04-01 during the rewrite\n",
		"real.js":    "const stripe = require('stripe')(key, {apiVersion: '2022-11-15'});\n",
	}
	for n, s := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	oldest, at := scanVersionPins(dir)
	if oldest != "2022-11-15" || !strings.Contains(at, "real.js") {
		t.Errorf("only the Stripe pin may count, got %q at %q", oldest, at)
	}
}

func TestVariantFieldAndOmitCoherence(t *testing.T) {
	// Round-3 verification F4: agents branch on structured fields — the
	// companion edit carries .variant, and omit+inserts is explained.
	dir := t.TempDir()
	src := `const stripe = require('stripe')('sk_test_x');
await stripe.paymentIntents.create({
  amount: 1099,
  currency: 'usd',
  payment_method_types: ['card'],
  confirm: true,
});
`
	if err := os.WriteFile(filepath.Join(dir, "confirm.js"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := fixRun(dir, dpmRule, true, false, &companionDecision{insert: false, reason: "test: omit"})
	if err != nil {
		t.Fatal(err)
	}
	variant := ""
	for _, f := range rep.Files {
		for _, e := range f.Edits {
			if e.Variant != "" {
				variant = e.Variant
			}
		}
	}
	if variant != "never-pin" {
		t.Errorf("companion edit must carry variant never-pin, got %q", variant)
	}
	if rep.Companion.Mode != "omit" || rep.Companion.Inserts != 1 {
		t.Fatalf("expected omit+1 insert, got %+v", rep.Companion)
	}
	explained := false
	for _, n := range rep.Companion.Notes {
		if strings.Contains(n, "omit verdict") {
			explained = true
		}
	}
	if !explained {
		t.Errorf("omit+inserts must be explained in notes, got %v", rep.Companion.Notes)
	}
}

func TestRemovalLeavesNoBlankArtifacts(t *testing.T) {
	// The bed-A review finding: every deleted entry used to leave a
	// whitespace-only line. Full-line expansion must prevent that in every
	// pair-shaped language.
	names := []string{"pi.py", "pi.rb", "pi.js", "pi.php", "pi.go", "Pi.cs"}
	dir := copyFixtures(t, names...)
	before := map[string]int{}
	for _, n := range names {
		before[n] = strings.Count(mustRead(t, dir, n), "\n")
	}
	if _, err := fixRun(dir, dpmRule, true, false, nil); err != nil {
		t.Fatal(err)
	}
	for n, lines := range before {
		out := mustRead(t, dir, n)
		if wsOnlyLine.MatchString(out) {
			t.Errorf("%s: whitespace-only line left behind:\n%s", n, out)
		}
		if got := strings.Count(out, "\n"); got != lines-1 {
			t.Errorf("%s: expected exactly the param line to disappear (%d -> %d lines), got %d", n, lines, lines-1, got)
		}
	}
}
