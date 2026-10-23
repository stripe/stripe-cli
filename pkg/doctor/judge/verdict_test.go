package judge

import "testing"

// TestVerdictByIntent pins that the finding's intent alone decides the
// class now that there's no account state to judge against.
func TestVerdictByIntent(t *testing.T) {
	if v := verdict("default-shaped"); !hasPrefix(v, "CANDIDATE") {
		t.Errorf("card/link only should be CANDIDATE, got %q", v)
	}
	if v := verdict("deliberate"); !hasPrefix(v, "SKIP") {
		t.Errorf("deliberate should SKIP, got %q", v)
	}
	if v := verdict("dynamic"); !hasPrefix(v, "REVIEW") {
		t.Errorf("dynamic should REVIEW, got %q", v)
	}
	if v := verdict("static"); !hasPrefix(v, "REVIEW") {
		t.Errorf("static multi-method list should REVIEW, got %q", v)
	}
}

func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
