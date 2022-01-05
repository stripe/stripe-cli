package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stripe/stripe-cli/pkg/config"
)

// GetIntegrationInsightResponse is the mapping for the fields in integration insight response
type GetIntegrationInsightResponse struct {
	Message string `json:"message"`
}

// IntegrationInsight returns integration insight
func IntegrationInsight(ctx context.Context, baseURL, apiVersion, apiKey string, profile *config.Profile, logID string) string {
	params := &RequestParameters{
		data:    []string{fmt.Sprintf("log=%s", logID)},
		version: apiVersion,
	}

	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     baseURL,
	}

	resp, err := base.MakeRequest(ctx, apiKey, "/v1/stripecli/integration_insight", params, true)
	if err != nil {
		return fmt.Sprintf("Failed to retrieve insight. Error: %s", err)
	}

	data := GetIntegrationInsightResponse{}
	json.Unmarshal(resp, &data)

	return data.Message
}
