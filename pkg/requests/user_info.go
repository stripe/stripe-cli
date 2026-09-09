package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// UserInfo is the response from GET /v1/stripecli/user_info.
type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
}

// GetUserInfo fetches identity information for the OAuth user associated with creds.
func GetUserInfo(ctx context.Context, apiBaseURL string, profile *config.Profile, creds stripe.Credentials, livemode bool) (UserInfo, error) {
	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     apiBaseURL,
		Livemode:       livemode,
	}

	resp, err := base.MakeRequest(ctx, creds, "/v1/stripecli/user_info", &RequestParameters{}, nil, true, nil)
	if err != nil {
		return UserInfo{}, err
	}

	var info UserInfo
	if err := json.Unmarshal(resp, &info); err != nil {
		return UserInfo{}, fmt.Errorf("failed to decode user info response: %w", err)
	}
	return info, nil
}
