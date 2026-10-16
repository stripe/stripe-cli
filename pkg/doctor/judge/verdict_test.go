package judge

import "testing"

// TestVerdictHonesty pins the doctor's judgment order: no-events admits
// ignorance, the version gate fires before anything else, and the
// dashboard-method diff catches the doc's silent-breakage warning.
func TestVerdictHonesty(t *testing.T) {
	ok := &accountFacts{VersionsOK: true, ConfiguredOK: true, EnabledMethods: []string{"card", "ideal"}}

	if v := verdict("default-shaped", `['card']`, &accountFacts{NoEvents: true, ConfiguredOK: true}); !hasPrefix(v, "REVIEW") {
		t.Errorf("no-events should be REVIEW, got %q", v)
	}
	if v := verdict("static", `['card', 'ideal', 'sepa_debit']`, ok); !hasPrefix(v, "CAUTION") || !contains(v, "sepa_debit") {
		t.Errorf("missing dashboard method should CAUTION and name it, got %q", v)
	}
	if v := verdict("default-shaped", `['card']`, ok); !hasPrefix(v, "CANDIDATE") {
		t.Errorf("enabled methods + version ok should be CANDIDATE, got %q", v)
	}
	if v := verdict("deliberate", `['oxxo']`, ok); !hasPrefix(v, "SKIP") {
		t.Errorf("deliberate should SKIP, got %q", v)
	}
}

func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
