package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// AccountFacts is the response from GET /v1/stripecli/account_facts.
type AccountFacts struct {
	// APIVersions are the API versions the account used in the last seven days.
	APIVersions []string `json:"api_versions"`
	// DefaultAPIVersion is the account's default API version.
	DefaultAPIVersion string `json:"default_api_version"`
}

func GetAccountFacts(ctx context.Context, apiBaseURL string, profile *config.Profile, creds stripe.Credentials, livemode bool) (AccountFacts, error) {
	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     apiBaseURL,
		Livemode:       livemode,
	}

	resp, err := base.MakeRequest(ctx, creds, "/v1/stripecli/account_facts", &RequestParameters{}, nil, true, nil)
	if err != nil {
		return AccountFacts{}, err
	}

	var accountFacts AccountFacts
	if err := json.Unmarshal(resp, &accountFacts); err != nil {
		return AccountFacts{}, fmt.Errorf("failed to decode account facts response: %w", err)
	}
	return accountFacts, nil
}
