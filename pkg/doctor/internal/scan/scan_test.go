package scan

import (
	"fmt"
	"os"
	"path/filepath"
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
}

func TestScanExactFindings(t *testing.T) {
	findings, scanned, parsed, err := Scan("../../testdata", testRule)
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
	t.Logf("%d findings, %d files scanned, %d parsed, %d skipped by prefilter",
		len(got), scanned, parsed, scanned-parsed)
}

// TestNegativesRejected names the adversarial cases explicitly so a regression
// that starts matching them is obvious from the test name alone.
func TestNegativesRejected(t *testing.T) {
	findings, _, _, err := Scan("../../testdata", testRule)
	if err != nil {
		t.Fatal(err)
	}
	banned := map[string]string{
		"../../testdata/neg.rb":        "non-Stripe resource (MyOrderModel.create)",
		"../../testdata/neg.py":        "bare variable and plain dict, not a call argument",
		"../../testdata/neg.js":        "plain object literal with no enclosing call",
		"../../testdata/attr_read.rb":  "reading the field off a response, not passing it",
		"../../testdata/client_opt.js": "client-side Elements option, not a server API param",
		"../../testdata/collide.py":    "variable in an unrelated function must not resolve cross-function",
		"../../testdata/neg_pool.rb":   "(param, operation) pairing not in the rule: nested-on-PI / top-level-on-subscription",
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
	findings, _, _, err := Scan("../../testdata", testRule)
	if err != nil {
		t.Fatal(err)
	}
	byFile := map[string]string{}
	for _, f := range findings {
		byFile[f.File] = f.Via
	}
	for file, wantVia := range map[string]string{
		"../../testdata/var_indirect.py": "var:params",
		"../../testdata/typed_decl.cs":   "type:options",
		"../../testdata/Recv.java":       "recv:paramsBuilder",
		"../../testdata/pi.rb":           "direct",
	} {
		if got := byFile[file]; got != wantVia {
			t.Errorf("%s resolved via %q, want %q", file, got, wantVia)
		}
	}
}

// TestPrefilterSkipsCleanFiles guards the property that makes large repos fast.
func TestPrefilterSkipsCleanFiles(t *testing.T) {
	_, scanned, parsed, _ := Scan("../../testdata", testRule)
	if parsed >= scanned {
		t.Errorf("prefilter skipped nothing: scanned=%d parsed=%d", scanned, parsed)
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
		findings, scanned, parsed, _ := Scan(dir, testRule)
		if len(findings) != dirty {
			b.Fatalf("got %d findings, want %d (scanned=%d parsed=%d)",
				len(findings), dirty, scanned, parsed)
		}
	}
}
