package requests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// NetworkBusinessProfile identifies the authenticated account's network business profile.
type NetworkBusinessProfile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// GetNetworkBusinessProfile retrieves the current account's profile, or nil if none exists.
func GetNetworkBusinessProfile(ctx context.Context, apiBaseURL string, profile *config.Profile, creds stripe.Credentials) (*NetworkBusinessProfile, error) {
	base := &Base{
		Profile:          profile,
		Method:           http.MethodGet,
		SuppressOutput:   true,
		APIBaseURL:       apiBaseURL,
		Livemode:         creds.Livemode(),
		IsPreviewCommand: true,
	}
	resp, err := base.MakeRequest(ctx, creds, "/v2/network/business_profiles/me", &RequestParameters{}, nil, true, nil)
	if err != nil {
		var requestErr RequestError
		if errors.As(err, &requestErr) && requestErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch network business profile: %w", err)
	}

	var businessProfile NetworkBusinessProfile
	if err := json.Unmarshal(resp, &businessProfile); err != nil {
		return nil, fmt.Errorf("failed to decode network business profile response: %w", err)
	}
	return &businessProfile, nil
}
