package stripe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatURLPath(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"ordinary ID", "cus_123", "/v1/customers/cus_123"},
		{"slash and traversal", "id1/../../customers/cus_123", "/v1/customers/id1%2F..%2F..%2Fcustomers%2Fcus_123"},
		{"percent", "50%", "/v1/customers/50%25"},
		// Positional arguments are raw IDs. A pre-encoded slash is intentionally escaped again.
		{"pre-encoded slash", "%2F", "/v1/customers/%252F"},
		{"space", "a b", "/v1/customers/a%20b"},
		{"plus", "a+b", "/v1/customers/a+b"},
		{"query and fragment", "a?b#c", "/v1/customers/a%3Fb%23c"},
		{"unicode", "café", "/v1/customers/caf%C3%A9"},
		{"punctuation", "a;b,c@d:e", "/v1/customers/a%3Bb%2Cc@d:e"},
		{"other dots", "test..", "/v1/customers/test.."},
		{"dots with suffix", "..?", "/v1/customers/..%3F"},
		{"pre-encoded dots", "%2E%2E", "/v1/customers/%252E%252E"},
		{"backslash", `a\..\b`, "/v1/customers/a%5C..%5Cb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatURLPath("/v1/customers/{customer}", []string{tt.input})
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
	got, err := FormatURLPath("/v1/accounts/{account}/customers/{customer}", []string{"acct/one", "cus?two"})
	require.NoError(t, err)
	require.Equal(t, "/v1/accounts/acct%2Fone/customers/cus%3Ftwo", got)

	for _, value := range []string{".", ".."} {
		for _, params := range [][]string{{value, "cus_123"}, {"acct_123", value}} {
			got, err := FormatURLPath("/v1/accounts/{account}/customers/{customer}", params)
			require.Empty(t, got)
			require.ErrorContains(t, err, "path arguments cannot be . or ..")
		}
	}
	got, err = FormatURLPath("/v1/customers/{customer}", []string{""})
	require.Empty(t, got)
	require.ErrorContains(t, err, "path arguments cannot be empty")
}
