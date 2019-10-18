package requests

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stripe/stripe-cli/pkg/config"
)

const (
	validToken        = "tok_visa"
	declinedToken     = "tok_chargeDeclined"
	disputeToken      = "tok_createDisputeInquiry"
	chargeFailedToken = "tok_chargeCustomerFail"
)

func parseResponse(response []byte) (map[string]interface{}, error) {
	var result map[string]interface{}

	err := json.Unmarshal(response, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// WebhookEndpointList contains the list of webhook endpoints for the account
type WebhookEndpointList struct {
	Data []WebhookEndpoint `json:"data"`
}

// WebhookEndpoint contains the data for each webhook endpoint
type WebhookEndpoint struct {
	Application   string   `json:"application"`
	EnabledEvents []string `json:"enabled_events"`
	URL           string   `json:"url"`
}

// Examples stores possible webhook test events to trigger for the CLI
type Examples struct {
	Profile    config.Profile
	APIBaseURL string
	APIVersion string
	APIKey     string
}

func (ex *Examples) buildRequest(method string, data []string) (*Base, *RequestParameters) {
	params := &RequestParameters{
		data:    data,
		version: ex.APIVersion,
	}

	base := &Base{
		Profile:        &ex.Profile,
		Method:         method,
		SuppressOutput: true,
		APIBaseURL:     ex.APIBaseURL,
	}

	return base, params
}

func (ex *Examples) performStripeRequest(req *Base, endpoint string, params *RequestParameters) (map[string]interface{}, error) {
	resp, err := req.MakeRequest(ex.APIKey, endpoint, params, true)
	if err != nil {
		return nil, err
	}

	return parseResponse(resp)
}

// WebhookEndpointsList returns all the webhook endpoints on a users' account
func (ex *Examples) WebhookEndpointsList() WebhookEndpointList {
	params := &RequestParameters{
		version: ex.APIVersion,
		data:    []string{"limit=30"},
	}

	base := &Base{
		Profile:        &ex.Profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     ex.APIBaseURL,
	}
	resp, _ := base.MakeRequest(ex.APIKey, "/v1/webhook_endpoints", params, true)
	data := WebhookEndpointList{}
	json.Unmarshal(resp, &data)

	return data
}
