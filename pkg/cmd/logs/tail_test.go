package logs

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/logtailing"
)

func hasZeroValueString(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if hasZeroValueString(v.Field(i)) {
				return true
			}
		}
		return false
	case reflect.String:
		return v.IsZero()
	default:
		return false
	}
}

func containsZeroValueStrings(x interface{}) bool {
	v := reflect.ValueOf(x)

	// If it's a pointer, dereference it
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if !v.IsValid() {
		return true
	}

	return hasZeroValueString(v)
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "does not change basic strings",
			input:    "GET",
			expected: "GET",
		},
		{
			name:     "removes ansi escape codes",
			input:    "\x0d\x0a\x1b[90mvery cool\x0d\x0a\x1b[32m and very legal",
			expected: "very cool and very legal",
		},
		{
			name:     "removes newlines",
			input:    "\x0d\x0a\x1b[90mvery cool",
			expected: "very cool",
		},
		{
			name:     "removes both ansi escape codes and newlines",
			input:    "\x0d\x0a\x1b[90ma horse\r\n a dog\n a cat\x0d\x0a\x1b[32m and a bird",
			expected: "a horse a dog a cat and a bird",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := sanitize(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestSanitizePayload(t *testing.T) {
	withAnsi := func(s string) string {
		return fmt.Sprintf("\x1b[90m%s\x1b[0m", s)
	}

	payload := logtailing.EventPayload{
		Error: logtailing.RedactedError{
			Charge:       withAnsi("ch_123"),
			Code:         withAnsi("invlaid_argument"),
			DeclineCode:  withAnsi("card_declined"),
			ErrorInsight: withAnsi("make fewer errors"),
			Message:      withAnsi("an error occurred"),
			Param:        withAnsi("card"),
			Type:         withAnsi("invalid_request"),
		},
		Method:    withAnsi("POST"),
		RequestID: withAnsi("req_123"),
		URL:       withAnsi("https://example.com"),
	}

	expected := logtailing.EventPayload{
		Error: logtailing.RedactedError{
			Charge:       "ch_123",
			Code:         "invlaid_argument",
			DeclineCode:  "card_declined",
			ErrorInsight: "make fewer errors",
			Message:      "an error occurred",
			Param:        "card",
			Type:         "invalid_request",
		},
		Method:    "POST",
		RequestID: "req_123",
		URL:       "https://example.com",
	}

	// Ensures that we're testing/covering the entire payload in case
	// any new fields are added
	require.Equal(t, containsZeroValueStrings(payload), false)

	sanitizePayload(&payload)

	assert.Equal(t, expected, payload)
}
