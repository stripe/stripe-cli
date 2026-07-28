package reporting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactSensitiveStrings(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"sk_live_abc123", "sk_live_[REDACTED]"},
		{"sk_test_abc123", "sk_test_[REDACTED]"},
		{"rk_live_abc123", "rk_live_[REDACTED]"},
		{"rk_test_abc123", "rk_test_[REDACTED]"},
		{"pk_live_abc123", "pk_live_[REDACTED]"},
		{"pk_test_abc123", "pk_test_[REDACTED]"},
		{"whsec_abc123==", "whsec_[REDACTED]"},
		{"oak_abc123", "oak_[REDACTED]"},
		{"oak_live_abc123", "oak_[REDACTED]"},
		{"token oak_abc123 is invalid", "token oak_[REDACTED] is invalid"},
		{"API key sk_live_abc123 is invalid", "API key sk_live_[REDACTED] is invalid"},
		{"user@example.com", "[REDACTED]"},
		{"login failed for user@stripe.com: invalid password", "login failed for [REDACTED]: invalid password"},
		{"no sensitive data here", "no sensitive data here"},
		{"", ""},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, redactSensitiveStrings(c.input), "input: %q", c.input)
	}
}
