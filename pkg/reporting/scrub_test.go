package reporting

import (
	"testing"

	sentry "github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
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

func TestScrubEventReplacesWrapperExceptionType(t *testing.T) {
	cases := []struct {
		name           string
		category       string
		exceptionTypes []string
		expected       []string
	}{
		{
			name:           "replaces the wrapper type with the category",
			category:       "api",
			exceptionTypes: []string{"*errors.errorString", errorcategory.WrapperTypeName},
			expected:       []string{"*errors.errorString", "api"},
		},
		{
			name:           "leaves uncategorized exception types alone",
			category:       "filesystem",
			exceptionTypes: []string{"*fs.PathError"},
			expected:       []string{"*fs.PathError"},
		},
		{
			name:           "leaves the wrapper type alone without a category tag",
			category:       "",
			exceptionTypes: []string{errorcategory.WrapperTypeName},
			expected:       []string{errorcategory.WrapperTypeName},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			event := sentry.NewEvent()
			if c.category != "" {
				event.Tags["error_category"] = c.category
			}
			for _, exceptionType := range c.exceptionTypes {
				event.Exception = append(event.Exception, sentry.Exception{Type: exceptionType, Value: "boom"})
			}

			scrubbed := scrubEvent(event, nil)

			actual := make([]string, 0, len(scrubbed.Exception))
			for _, exception := range scrubbed.Exception {
				actual = append(actual, exception.Type)
			}
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestScrubEventRedactsExceptionValues(t *testing.T) {
	event := sentry.NewEvent()
	event.Exception = []sentry.Exception{{Type: "*errors.errorString", Value: "key sk_live_abc123 rejected"}}

	scrubbed := scrubEvent(event, nil)

	assert.Equal(t, "key sk_live_[REDACTED] rejected", scrubbed.Exception[0].Value)
}
