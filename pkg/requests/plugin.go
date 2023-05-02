package requests

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/stripe/stripe-cli/pkg/config"
)

// PluginData contains the plugin download information
type PluginData struct {
	PluginBaseURL       string   `json:"base_url"`
	AdditionalManifests []string `json:"additional_manifests,omitempty"`
}

// GetPluginData returns the plugin download information
func GetPluginData(ctx context.Context, baseURL, apiVersion, apiKey string, profile *config.Profile) (PluginData, error) {
	params := &RequestParameters{
		data:    []string{},
		version: apiVersion,
	}

	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     baseURL,
	}
	// /v1/stripecli/get-plugin-url
	resp, err := base.MakeRequest(ctx, apiKey, "/v1/stripecli/get-plugin-url", params, true)
	if err != nil {
		return PluginData{}, err
	}

	data := PluginData{}
	json.Unmarshal(resp, &data)

	return data, nil
}
