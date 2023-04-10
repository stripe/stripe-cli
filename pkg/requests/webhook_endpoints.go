package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
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

// WebhookEndpointsListWithClient returns all the webhook endpoints on a users' account
func WebhookEndpointsListWithClient(ctx context.Context, client stripe.RequestPerformer, apiVersion string, profile *config.Profile) WebhookEndpointList {
	params := &RequestParameters{
		data:    []string{"limit=30"},
		version: apiVersion,
	}

	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
	}
	resp, _ := base.MakeRequestWithClient(ctx, client, "/v1/webhook_endpoints", params, true)
	data := WebhookEndpointList{}
	json.Unmarshal(resp, &data)

	return data
}

// WebhookEndpointCreate creates a new webhook endpoint
func WebhookEndpointCreate(ctx context.Context, baseURL, apiVersion, apiKey, url, description string, connect bool, profile *config.Profile) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("url cannot be empty")
	}

	data := []string{
		fmt.Sprintf("url=%s", url),
		"enabled_events[]=*",
	}
	if description != "" {
		data = append(data, fmt.Sprintf("description=%s", description))
	}
	if connect {
		data = append(data, "connect=true") // connect is false by default for webhook endpoint creation
	}

	params := &RequestParameters{
		data:    data,
		version: apiVersion,
	}

	base := &Base{
		Profile:        profile,
		Method:         http.MethodPost,
		SuppressOutput: true,
		APIBaseURL:     baseURL,
	}
	_, err := base.MakeRequest(ctx, apiKey, "/v1/webhook_endpoints", params, true)
	if err != nil {
		return err
	}
	return nil
}
