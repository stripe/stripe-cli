package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// WorkspaceContext is the response from GET /v1/stripecli/workspace_context.
type WorkspaceContext struct {
	WorkspaceID string `json:"workspace_id"`
}

// GetWorkspaceContext fetches the workspace associated with the active OAuth context.
func GetWorkspaceContext(ctx context.Context, apiBaseURL string, profile *config.Profile, creds stripe.Credentials, livemode bool) (WorkspaceContext, error) {
	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     apiBaseURL,
		Livemode:       livemode,
	}

	resp, err := base.MakeRequest(ctx, creds, "/v1/stripecli/workspace_context", &RequestParameters{}, nil, true, nil)
	if err != nil {
		return WorkspaceContext{}, err
	}

	var workspaceContext WorkspaceContext
	if err := json.Unmarshal(resp, &workspaceContext); err != nil {
		return WorkspaceContext{}, fmt.Errorf("failed to decode workspace context response: %w", err)
	}
	return workspaceContext, nil
}
