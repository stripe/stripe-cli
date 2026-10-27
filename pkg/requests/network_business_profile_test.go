package requests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestGetNetworkBusinessProfile(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		want    *NetworkBusinessProfile
		wantErr string
	}{
		{"success", http.StatusOK, `{"id":"profile_test_123","username":"furever"}`, &NetworkBusinessProfile{ID: "profile_test_123", Username: "furever"}, ""},
		{"not found", http.StatusNotFound, `{"error":{"message":"No business profile"}}`, nil, ""},
		{"unauthorized", http.StatusUnauthorized, `{"error":{"message":"Invalid token"}}`, nil, "failed to fetch network business profile"},
		{"forbidden", http.StatusForbidden, `{"error":{"message":"Permission denied"}}`, nil, "failed to fetch network business profile"},
		{"server error", http.StatusInternalServerError, `{"error":{"message":"Server error"}}`, nil, "failed to fetch network business profile"},
		{"invalid JSON", http.StatusOK, `invalid`, nil, "failed to decode network business profile response"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/v2/network/business_profiles/me", r.URL.Path)
				assert.Empty(t, r.URL.RawQuery)
				assert.Equal(t, "Bearer oak_test_123", r.Header.Get("Authorization"))
				assert.Equal(t, "acct_123", r.Header.Get("Stripe-Context"))
				assert.Equal(t, "false", r.Header.Get("Stripe-Livemode"))
				assert.Equal(t, StripePreviewVersionHeaderValue, r.Header.Get("Stripe-Version"))
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body)) //nolint:errcheck
			}))
			defer server.Close()

			profile, err := GetNetworkBusinessProfile(context.Background(), server.URL, nil, stripe.NewOAKCredentials("oak_test_123", "acct_123", false))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, profile)
		})
	}
}
