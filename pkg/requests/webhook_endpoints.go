package requests

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/stripe/stripe-cli/pkg/config"
)

// WebhookEndpointList contains the list of webhook endpoints for the account
type WebhookEndpointList struct {
	Data []WebhookEndpoint `json:"data"`
}

// WebhookEndpoint contains the data for each webhook endpoint
type WebhookEndpoint struct {
	Application   string   `json:"application"`
	EnabledEvents []string `json:"enabled_events"`
	URL           string   `json:"url"`
	Status        string   `json:"status"`
}

// WebhookEndpointsList returns all the webhook endpoints on a users' account
func WebhookEndpointsList(ctx context.Context, baseURL, apiVersion, apiKey string, profile *config.Profile) WebhookEndpointList {
	params := &RequestParameters{
		data:    []string{"limit=30"},
		version: apiVersion,
	}

	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     baseURL,
	}
	resp, _ := base.MakeRequest(ctx, apiKey, "/v1/webhook_endpoints", params, true)
	data := WebhookEndpointList{}
	json.Unmarshal(resp, &data)

	return data
}
