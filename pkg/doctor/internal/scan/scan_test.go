package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
)

// testRule mirrors the real dpm rule's shape, kept local so scan's tests
// stay hermetic.
var testRule = catalog.Rule{
	ID: "dpm",
	Match: []catalog.ParamMatch{
		{Param: "payment_method_types", Operations: []string{"POST /v1/payment_intents"}},
		{Param: "payment_method_types", Operations: []string{"POST /v1/setup_intents"}},
		{Param: "payment_method_types", Operations: []string{"POST /v1/checkout/sessions"}},
	},
}

// want is the exact expected finding set: file:line:col. Anything extra is a
// false positive; anything missing is a false negative. The fixtures live in
// ../../testdata.
var want = []string{
	"../../testdata/Comp.tsx:7:5",        // direct (TSX grammar, JSX in file)
	"../../testdata/Pi.cs:10:7",          // direct
	"../../testdata/Pi.java:9:10",        // direct (builder chain)
	"../../testdata/Recv.java:8:19",      // recv: receiver's declared type
	"../../testdata/pi.go:13:3",          // direct
	"../../testdata/pi.js:6:3",           // direct
	"../../testdata/pi.php:7:4",          // direct
	"../../testdata/pi.py:7:5",           // direct
	"../../testdata/pi.rb:7:3",           // direct
	"../../testdata/static.php:4:6",      // direct (legacy \Stripe\X::create static style)
	"../../testdata/esm.mjs:4:3",         // direct (ESM via the JS grammar)
	"../../testdata/typed_decl.cs:10:7",  // type: C# target-typed new()
	"../../testdata/var_indirect.py:5:6", // var: bag bound then passed

	"../../testdata/bulk_adder.java:10:10", // direct (Java addAll* bulk builder adder)
	"../../testdata/client_pi.py:6:6",      // direct (StripeClient resource-style call)
	"../../testdata/client_pi.rb:6:3",      // direct (StripeClient resource-style call)
	"../../testdata/create_params.go:13:3", // direct (operation-specific *CreateParams type)
	"../../testdata/hash_rocket.rb:4:30",   // direct (hash-rocket symbol key)
	"../../testdata/hash_rocket.rb:5:29",   // direct (hash-rocket string key)
	"../../testdata/str_key.js:4:4",        // direct (quoted string key)
	"../../testdata/str_key.ts:4:4",        // direct (quoted string key)

	"../../testdata/client_params_kw.py:6:6",       // direct (params= request envelope)
	"../../testdata/client_params_indirect.py:7:6", // var: bag passed via params= envelope
	"../../testdata/spread_merge.js:5:3",           // var: bag merged via object spread
	"../../testdata/nested_same_scope.py:4:16",     // var: bag bound and used in the same nested function

	"../../testdata/create_async.py:4:41",                   // direct (Python asyncio create_async)
	"../../testdata/client_create_async.py:6:14",            // direct (StripeClient resource-style create_async)
	"../../testdata/typed_decl.ts:5:3",                      // type: explicit params type, assigned separately
	"../../testdata/unrelated_local_reassign.py:4:12",       // var: an unrelated nested function's own local reassignment must not invalidate the outer binding
	"../../testdata/function_defined_before_assign.js:11:3", // var: a function's body runs when called, not at its earlier textual definition site
	"../../testdata/conditional_reassign.py:5:12",           // var: a conditional reassignment isn't a definite overwrite on every path
	"../../testdata/unrelated_block_scope.js:4:18",          // var: an unrelated block-scoped const in a sibling if-branch must not invalidate the outer binding
	"../../testdata/do_block.rb:5:30",                       // direct (Ruby do...end block isn't the real argument list)
	"../../testdata/comment_before_args.js:6:3",             // direct (comment between method name and argument list)
	"../../testdata/if_block_scope.js:4:18",                 // var: an if-branch's own block-scoped const must not shadow the module-level bag reaching the call after it
	"../../testdata/shorthand_prop.js:4:32",                 // direct (JS/TS shorthand property, {payment_method_types} sugar for {payment_method_types: payment_method_types})
	"../../testdata/shorthand_prop.ts:4:32",                 // direct (same, TS grammar)
	"../../testdata/var_legacy_create.go:12:3",              // var: Go's legacy ambiguous type bound with `var` (not `:=`), reaching New
	"../../testdata/nested_helper_scope.py:3:12",            // var: a nested helper function's own local must not shadow the module-level bag reaching the call in the outer function
	"../../testdata/nested_helper_scope.js:3:18",            // var: same, JS grammar
	"../../testdata/closure_conditional_write.js:3:16",      // var: a write (not a declaration) to a captured outer variable isn't a new local shadowing it
	"../../testdata/braceless_conditional.php:4:13",         // var: PHP's brace-less single-statement if body is still conditional, not an unconditional overwrite
	"../../testdata/braceless_conditional.js:3:16",          // var: JS's brace-less single-statement if body is still conditional, not an unconditional overwrite
}

func TestScanExactFindings(t *testing.T) {
	findings, scanned, parsed, skipped, incomplete, err := Scan("../../testdata", testRule)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, f := range findings {
		got = append(got, fmt.Sprintf("%s:%d:%d", f.File, f.Line, f.Col))
	}
	sort.Strings(got)

	wantSet := map[string]bool{}
	for _, w := range want {
		wantSet[w] = true
	}
	gotSet := map[string]bool{}
	for _, g := range got {
		gotSet[g] = true
	}

	for _, g := range got {
		if !wantSet[g] {
			t.Errorf("FALSE POSITIVE: %s", g)
		}
	}
	for _, w := range want {
		if !gotSet[w] {
			t.Errorf("FALSE NEGATIVE: %s", w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("count: got %d findings, want %d", len(got), len(want))
	}
	if incomplete != 0 {
		t.Errorf("expected every fixture to be readable, got %d incomplete", incomplete)
	}
	t.Logf("%d findings, %d files scanned, %d parsed, %d skipped by prefilter",
		len(got), scanned, parsed, skipped)
}

// TestNegativesRejected names the adversarial cases explicitly so a regression
// that starts matching them is obvious from the test name alone.
func TestNegativesRejected(t *testing.T) {
	findings, _, _, skipped, incomplete, err := Scan("../../testdata", testRule)
	_ = skipped
	if err != nil {
		t.Fatal(err)
	}
	if incomplete != 0 {
		t.Errorf("expected every fixture to be readable, got %d incomplete", incomplete)
	}
	banned := map[string]string{
		"../../testdata/neg.rb":                         "non-Stripe resource (MyOrderModel.create)",
		"../../testdata/neg.py":                         "bare variable and plain dict, not a call argument",
		"../../testdata/neg.js":                         "plain object literal with no enclosing call",
		"../../testdata/attr_read.rb":                   "reading the field off a response, not passing it",
		"../../testdata/client_opt.js":                  "client-side Elements option, not a server API param",
		"../../testdata/collide.py":                     "variable in an unrelated function must not resolve cross-function",
		"../../testdata/neg_pool.rb":                    "(param, operation) pairing not in the rule: nested-on-PI / top-level-on-subscription",
		"../../testdata/metadata_misattribution.py":     "bag bound to an unrelated keyword (metadata=params), not merged into the call's own params",
		"../../testdata/neg_spread.js":                  "bag nested under metadata, not spread into the call's own params",
		"../../testdata/neg_wrong_method.py":            "PaymentIntent.modify targets a different operation than the create-only rule",
		"../../testdata/neg_construct_from.py":          "construct_from makes no API request at all",
		"../../testdata/neg_shadowed_param.js":          "the call's own params is a shadowing function parameter, not the module-level object",
		"../../testdata/neg_reassigned.py":              "the bag is overwritten with a clean one before the call that actually runs",
		"../../testdata/neg_value_py.py":                "the parameter name appears as a string VALUE, not a key",
		"../../testdata/neg_value.rb":                   "the parameter name appears as a string VALUE, not a key",
		"../../testdata/neg_value.js":                   "the parameter name appears as a string VALUE, not a key",
		"../../testdata/neg_value.php":                  "the parameter name appears as a string VALUE, not a key",
		"../../testdata/neg_value.go":                   "the parameter name appears as an identifier VALUE, not a field name",
		"../../testdata/neg_value.cs":                   "the parameter name appears as an identifier VALUE, not a field name",
		"../../testdata/neg_untyped_decl.ts":            "untyped declarator whose initializer merely mentions the resource, not a typed declaration",
		"../../testdata/neg_reassigned_after_call.py":   "the call already ran with a clean bag before the dirty reassignment ever happened",
		"../../testdata/neg_legacy_update.go":           "the legacy, create-and-update-shared PaymentIntentParams type is passed to Update, not New",
		"../../testdata/neg_local_shadow.js":            "the call's own params is a body-level local re-declaration, not the outer module-level object",
		"../../testdata/neg_lambda_shadow.py":           "the call's own params is the lambda's own parameter, not the module-level dict",
		"../../testdata/neg_reassigned_js.js":           "the bag is overwritten with a clean one before the call, despite mismatched statement-wrapper node types",
		"../../testdata/neg_reassigned.php":             "the bag is overwritten with a clean one before the call that actually runs",
		"../../testdata/neg_bare_block_shadow.js":       "the call's own params is a bare block's own let/const re-declaration, not the outer module-level object",
		"../../testdata/neg_typed_decl_scope.ts":        "the typed declaration of the same name belongs to an unrelated nested function, not the outer module-level binding",
		"../../testdata/neg_var_reassigned.go":          "the var-bound bag is overwritten with a clean one before the call that actually runs",
		"../../testdata/neg_typed_decl_block_shadow.ts": "the typed declaration belongs to an unrelated inner block's own re-declaration, not the outer bag",
		"../../testdata/neg_var_block_scoped.go":        "the dirty params never escapes its if-branch, so it must not be associated with the call after the branch closes",
		"../../testdata/neg_let_block_scoped.js":        "the block-scoped const never escapes its if-branch, so it must not be associated with the call outside it",
		"../../testdata/neg_wrong_model.rb":             "PaymentIntentAudit merely contains PaymentIntent as a substring, it is not that resource",
		"../../testdata/neg_unrelated_type_string.cs":   "the resource token only appears inside a string property value, not the constructed type itself",
		"../../testdata/neg_go_block_shadow.go":         "the outer bag is used only by Update; the inner New call uses its own, block-scoped clean params",
		"../../testdata/neg_do_block_local.rb":          "the do...end block's own local hash is never passed to create — not a request argument at all",
		"../../testdata/neg_callback_local.js":          "the callback's own local object is never passed to create — not a request argument at all",
	}
	for _, f := range findings {
		if why, bad := banned[f.File]; bad {
			t.Errorf("matched %s but should not: %s", f.File, why)
		}
	}
}

// TestResolutionMechanisms asserts each indirection path stays exercised, so a
// regression in one cannot hide behind the others still passing.
func TestResolutionMechanisms(t *testing.T) {
	findings, _, _, skipped, incomplete, err := Scan("../../testdata", testRule)
	_ = skipped
	if err != nil {
		t.Fatal(err)
	}
	if incomplete != 0 {
		t.Errorf("expected every fixture to be readable, got %d incomplete", incomplete)
	}
	byFile := map[string]string{}
	for _, f := range findings {
		byFile[f.File] = f.Via
	}
	for file, wantVia := range map[string]string{
		"../../testdata/var_indirect.py": "var:params",
		"../../testdata/typed_decl.cs":   "type:options",
		"../../testdata/typed_decl.ts":   "type:params",
		"../../testdata/Recv.java":       "recv:paramsBuilder",
		"../../testdata/pi.rb":           "direct",
	} {
		if got := byFile[file]; got != wantVia {
			t.Errorf("%s resolved via %q, want %q", file, got, wantVia)
		}
	}
}

// TestPrefilterSkipsCleanFiles guards the property that makes large repos fast.
// TestSymlinkedRootWalked guards against filepath.WalkDir's lstat-the-
// root-itself behavior silently turning a symlinked scan root into an
// apparently clean scan: zero files seen, zero findings, no error.
func TestSymlinkedRootWalked(t *testing.T) {
	real := t.TempDir()
	src := "Stripe::PaymentIntent.create(payment_method_types: ['card'])\n"
	if err := os.WriteFile(filepath.Join(real, "dirty.rb"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation requires elevated privileges on Windows")
		}
		t.Fatal(err)
	}

	findings, scanned, parsed, skipped, _, err := Scan(link, testRule)
	_, _ = parsed, skipped
	if err != nil {
		t.Fatal(err)
	}
	if scanned != 1 {
		t.Errorf("scanned = %d, want 1 (a symlinked root must still be walked)", scanned)
	}
	if len(findings) != 1 {
		t.Errorf("got %d findings from a symlinked root, want 1", len(findings))
	}
}

func TestPrefilterSkipsCleanFiles(t *testing.T) {
	_, scanned, parsed, skipped, incomplete, _ := Scan("../../testdata", testRule)
	if incomplete != 0 {
		t.Errorf("expected every fixture to be readable, got %d incomplete", incomplete)
	}
	if parsed >= scanned {
		t.Errorf("prefilter skipped nothing: scanned=%d parsed=%d", scanned, parsed)
	}
	if skipped != scanned-parsed {
		t.Errorf("skipped = %d, want %d (scanned-parsed, with nothing incomplete here)", skipped, scanned-parsed)
	}
}

// TestSkippedCountsPrefilterDirectly guards against skipped being
// derived as scanned-parsed-incomplete, which double-subtracts a
// partially-parsed file: it already counts toward both parsed and
// incomplete, so subtracting incomplete again undercounts true
// prefilter skips.
func TestSkippedCountsPrefilterDirectly(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 9; i++ {
		clean := fmt.Sprintf("# file %d, nothing the rule cares about\nStripe::PaymentIntent.create(amount: 1099)\n", i)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("clean%d.rb", i)), []byte(clean), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	dirty := "Stripe::PaymentIntent.create(payment_method_types: ['card'])\n"
	if err := os.WriteFile(filepath.Join(dir, "budget.rb"), []byte(dirty), 0o600); err != nil {
		t.Fatal(err)
	}

	parseTimeoutMicros = 1
	defer func() { parseTimeoutMicros = 0 }()

	_, scanned, parsed, skipped, incomplete, err := Scan(dir, testRule)
	if err != nil {
		t.Fatal(err)
	}
	if scanned != 10 {
		t.Errorf("scanned = %d, want 10", scanned)
	}
	if parsed != 1 {
		t.Errorf("parsed = %d, want 1 (only the dirty file passes the prefilter)", parsed)
	}
	if incomplete != 1 {
		t.Errorf("incomplete = %d, want 1 (the one file parsed never finished)", incomplete)
	}
	if skipped != 9 {
		t.Errorf("skipped = %d, want 9 (the nine prefilter rejects, not 8 from double-subtracting the partial parse)", skipped)
	}
}

// TestIncompleteFileSurfaced guards against an incomplete file being
// silently folded into "skipped by prefilter" (indistinguishable from a
// clean file) rather than reported as incomplete coverage.
func TestIncompleteFileSurfaced(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits don't deny reads on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root: chmod 0 doesn't block reads")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked.rb")
	src := "Stripe::PaymentIntent.create(payment_method_types: ['card'])\n"
	if err := os.WriteFile(locked, []byte(src), 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(locked, 0o600)

	findings, scanned, parsed, _, incomplete, err := Scan(dir, testRule)
	if err != nil {
		t.Fatal(err)
	}
	if incomplete != 1 {
		t.Errorf("incomplete = %d, want 1", incomplete)
	}
	if scanned != 1 {
		t.Errorf("scanned = %d, want 1 (the file should still be counted as seen)", scanned)
	}
	if parsed != 0 {
		t.Errorf("parsed = %d, want 0 (an incomplete file was never actually inspected)", parsed)
	}
	if len(findings) != 0 {
		t.Errorf("got %d findings from an incomplete file, want 0", len(findings))
	}
}

// TestPartialParseSurfaced guards against a parser that exhausts its
// time budget on an input — returning a partial tree with a nil error
// — being mistaken for a complete, clean parse.
func TestPartialParseSurfaced(t *testing.T) {
	dir := t.TempDir()
	src := "Stripe::PaymentIntent.create(payment_method_types: ['card'])\n"
	if err := os.WriteFile(filepath.Join(dir, "budget.rb"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	parseTimeoutMicros = 1
	defer func() { parseTimeoutMicros = 0 }()

	_, scanned, parsed, _, incomplete, err := Scan(dir, testRule)
	if err != nil {
		t.Fatal(err)
	}
	if scanned != 1 || parsed != 1 {
		t.Errorf("scanned=%d parsed=%d, want 1 and 1 (the file was read and attempted)", scanned, parsed)
	}
	if incomplete != 1 {
		t.Errorf("incomplete = %d, want 1 (the parse never finished)", incomplete)
	}
}

// BenchmarkScanSyntheticRepo measures the whole pipeline on a repo where only a
// small fraction of files mention the parameter — the realistic shape.
func BenchmarkScanSyntheticRepo(b *testing.B) {
	dir := b.TempDir()
	const clean, dirty = 2000, 20

	for i := 0; i < clean; i++ {
		src := fmt.Sprintf("# file %d\nStripe::PaymentIntent.create(amount: %d, currency: 'eur')\n", i, i)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("c%d.rb", i)), []byte(src), 0o600); err != nil {
			b.Fatal(err)
		}
	}
	for i := 0; i < dirty; i++ {
		src := fmt.Sprintf("Stripe::PaymentIntent.create(\n  amount: %d,\n  payment_method_types: ['card'],\n)\n", i)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("d%d.rb", i)), []byte(src), 0o600); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings, scanned, parsed, _, _, _ := Scan(dir, testRule)
		if len(findings) != dirty {
			b.Fatalf("got %d findings, want %d (scanned=%d parsed=%d)",
				len(findings), dirty, scanned, parsed)
		}
	}
}
