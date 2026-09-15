package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// PlaygroundContext is the response from GET /v1/stripecli/playground_context.
type PlaygroundContext struct {
	PlaygroundID string `json:"playground_id"`
}

// GetPlaygroundContext fetches the playground associated with the active OAuth context.
func GetPlaygroundContext(ctx context.Context, apiBaseURL string, profile *config.Profile, creds stripe.Credentials, livemode bool) (PlaygroundContext, error) {
	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     apiBaseURL,
		Livemode:       livemode,
	}

	resp, err := base.MakeRequest(ctx, creds, "/v1/stripecli/playground_context", &RequestParameters{}, nil, true, nil)
	if err != nil {
		return PlaygroundContext{}, err
	}

	var playgroundContext PlaygroundContext
	if err := json.Unmarshal(resp, &playgroundContext); err != nil {
		return PlaygroundContext{}, fmt.Errorf("failed to decode playground context response: %w", err)
	}
	return playgroundContext, nil
}
