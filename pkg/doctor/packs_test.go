package doctor

import (
	"fmt"
	"strings"
	"testing"
)

// TestPacks asserts each authored pack's fixture corpus produces exactly the
// expected finding count per file — positives fire, negatives stay silent.
func TestPacks(t *testing.T) {
	cases := map[string]struct {
		rule   Rule
		expect map[string]int
	}{
		"tax-percent": {rule: taxPercentRule, expect: map[string]int{
			"negative.js":  0,
			"negative.py":  0,
			"negative.rb":  0,
			"positive.js":  1,
			"positive.php": 2,
			"positive.py":  1,
			"positive.rb":  2,
		}},
		"collection-method": {rule: collectionMethodRule, expect: map[string]int{
			"negative.js":  0,
			"negative.php": 0,
			"negative.py":  0,
			"negative.rb":  0,
			"positive.js":  2,
			"positive.php": 1,
			"positive.py":  2,
			"positive.rb":  4,
		}},
		"prorate": {rule: prorateRule, expect: map[string]int{
			"negative.js":  0,
			"negative.py":  0,
			"negative.rb":  0,
			"positive.js":  2,
			"positive.php": 1,
			"positive.py":  1,
			"positive.rb":  2,
		}},
		"source-types": {rule: sourceTypesRule, expect: map[string]int{
			"negative_comment.py":         0,
			"negative_source_resource.rb": 0,
			"negative_unbound_object.js":  0,
			"positive_confirm.py":         1,
			"positive_create.php":         1,
			"positive_create.rb":          1,
			"positive_update.js":          1,
		}},
		"ewcs": {rule: ewcsRule, expect: map[string]int{
			"server.js":        1,
			"subscribe.rb":     1,
			"checkout_form.js": 0,
			"webhook.rb":       0,
			"negative.js":      0,
		}},
		"pe": {rule: peRule, expect: map[string]int{
			"checkout.js": 0,
			"webhook.rb":  0,
		}},
		"ct": {rule: ctRule, expect: map[string]int{
			"server.js":   1,
			"client.js":   0,
			"migrated.js": 0,
		}},
		"flex": {rule: flexRule, expect: map[string]int{
			"positive.rb": 1,
			"capture.js":  1,
			"negative.py": 0,
		}},
	}
	for topic, c := range cases {
		t.Run(topic, func(t *testing.T) {
			dir := fmt.Sprintf("testdata-packs/%s", topic)
			findings, _, _, err := scan(dir, c.rule)
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]int{}
			for _, f := range findings {
				got[f.File[len(dir)+1:]]++
			}
			for file, want := range c.expect {
				if got[file] != want {
					t.Errorf("%s/%s: got %d findings, want %d", topic, file, got[file], want)
				}
			}
			for file, n := range got {
				if _, known := c.expect[file]; !known {
					t.Errorf("%s/%s: %d unexpected findings", topic, file, n)
				}
			}
		})
	}
}

// TestPackSignals pins the pack-declared signal layer: the EWCS readiness
// fixtures must show all four expected events missing, from-state frontend
// tokens present, and both package floors violated — while the DPM fixtures
// keep their original three-events-present / Card-Element-warning behavior.
func TestPackSignals(t *testing.T) {
	h, fw, mc := scanSignals("testdata-packs/ewcs", &ewcsSignals)
	if h.AllPresent {
		t.Error("ewcs: expected missing checkout.session.* handlers")
	}
	for _, e := range h.Events {
		if e.Present {
			t.Errorf("ewcs: event %s unexpectedly present", e.Event)
		}
	}
	if len(fw) == 0 {
		t.Error("ewcs: expected frontend warnings in checkout_form.js")
	}
	bad := 0
	for _, m := range mc {
		if !m.OK {
			bad++
		}
	}
	if bad != 2 {
		t.Errorf("ewcs: expected 2 failed manifest floors, got %d (%v)", bad, mc)
	}

	h2, fw2, _ := scanSignals("testdata", &dpmSignals)
	if !h2.AllPresent {
		t.Errorf("dpm: expected all three delayed-notification events present: %+v", h2.Events)
	}
	if len(fw2) == 0 {
		t.Error("dpm: expected the Card Element warning from frontend_card.js")
	}
}

// TestTriage pins the elements triage: a mixed-era repo must light up all
// four branches, each with file evidence, and the ct pack's signals must stay
// silent on already-migrated code.
func TestTriage(t *testing.T) {
	results := scanTriage("testdata-packs/elements", elementsTriage)
	if len(results) != 4 {
		t.Fatalf("expected all 4 triage branches to fire, got %d: %+v", len(results), results)
	}
	wantFile := map[int]string{0: "charges_old.rb", 1: "legacy_card.js", 2: "server_confirm.js", 3: "migrated_part.js"}
	for i, r := range results {
		if len(r.Evidence) == 0 {
			t.Errorf("branch %d (%s): no evidence", i, r.Detected)
			continue
		}
		found := false
		for _, ev := range r.Evidence {
			if strings.HasSuffix(ev.File, wantFile[i]) {
				found = true
			}
		}
		if !found {
			t.Errorf("branch %d (%s): expected evidence from %s, got %+v", i, r.Detected, wantFile[i], r.Evidence)
		}
	}

	// ct signals: fire on the legacy client, silent on migrated code
	_, fw, _ := scanSignals("testdata-packs/ct", &ctSignals)
	hitLegacy, hitMigrated := false, false
	for _, w := range fw {
		if strings.HasSuffix(w.File, "client.js") {
			hitLegacy = true
		}
		if strings.HasSuffix(w.File, "migrated.js") {
			hitMigrated = true
		}
	}
	if !hitLegacy {
		t.Error("ct: expected createPaymentMethod signal in client.js")
	}
	if hitMigrated {
		t.Error("ct: migrated.js must not trigger from-state signals")
	}

	// pe signals: partial webhook coverage must read as not-all-present
	h, fw2, _ := scanSignals("testdata-packs/pe", &peSignals)
	if h.AllPresent {
		t.Error("pe: only payment_intent.succeeded is handled; AllPresent must be false")
	}
	if len(fw2) == 0 {
		t.Error("pe: expected legacy Element signals in checkout.js")
	}
}
