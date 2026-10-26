package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// contextStatusesBatchSize is the maximum number of contexts the
// /v1/stripecli/context_statuses endpoint accepts per request.
const contextStatusesBatchSize = 100

// ContextStatusQuery identifies an (account, mode) pair to resolve via
// GetContextStatuses.
type ContextStatusQuery struct {
	AccountID string
	Livemode  bool
}

// ContextStatus is the per-context result from POST /v1/stripecli/context_statuses.
// ParentAccountID is only populated for sandbox (non-live) contexts whose
// parent account is both known and within the caller's permission grant.
type ContextStatus struct {
	AccountID       string `json:"account_id"`
	Livemode        bool   `json:"livemode"`
	Valid           bool   `json:"valid"`
	ParentAccountID string `json:"parent_account_id,omitempty"`
}

type contextStatusesResponse struct {
	Contexts []ContextStatus `json:"contexts"`
}

// GetContextStatuses resolves parent-account and validity info for a batch of
// (account_id, livemode) contexts via POST /v1/stripecli/context_statuses.
func GetContextStatuses(ctx context.Context, apiBaseURL string, profile *config.Profile, uat string, queries []ContextStatusQuery) ([]ContextStatus, error) {
	if len(queries) == 0 {
		return nil, nil
	}

	creds := stripe.Credentials{Token: uat}
	var results []ContextStatus
	for start := 0; start < len(queries); start += contextStatusesBatchSize {
		end := start + contextStatusesBatchSize
		if end > len(queries) {
			end = len(queries)
		}

		batch, err := getContextStatusesBatch(ctx, apiBaseURL, profile, creds, queries[start:end])
		if err != nil {
			return nil, err
		}
		results = append(results, batch...)
	}
	return results, nil
}

func getContextStatusesBatch(ctx context.Context, apiBaseURL string, profile *config.Profile, creds stripe.Credentials, queries []ContextStatusQuery) ([]ContextStatus, error) {
	contexts := make([]interface{}, 0, len(queries))
	for _, q := range queries {
		contexts = append(contexts, map[string]interface{}{
			"account_id": q.AccountID,
			"livemode":   q.Livemode,
		})
	}

	base := &Base{
		Profile:        profile,
		Method:         http.MethodPost,
		SuppressOutput: true,
		APIBaseURL:     apiBaseURL,
	}

	resp, err := base.MakeRequest(ctx, creds, "/v1/stripecli/context_statuses", &RequestParameters{}, map[string]interface{}{
		"contexts": contexts,
	}, true, nil)
	if err != nil {
		return nil, err
	}

	var result contextStatusesResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to decode context statuses response: %w", err)
	}
	return result.Contexts, nil
}
