package requests

import (
	"context"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/config"
)

// PluginData contains the plugin download information
type PluginData struct {
	PluginBaseURL       string   `json:"base_url"`
	AdditionalManifests []string `json:"additional_manifests,omitempty"`
}

// defaultPluginBaseURL is the repository where plugins are hosted.
// This is used as a fallback if the user is not logged in to the CLI.
const defaultPluginBaseURL = "https://stripe.jfrog.io/artifactory/stripe-cli-plugins-local"

var DefaultPluginData = PluginData{
	PluginBaseURL:       defaultPluginBaseURL,
	AdditionalManifests: []string{},
}

// GetPluginData returns the plugin download information
func GetPluginData(ctx context.Context, baseURL, apiVersion, apiKey string, profile *config.Profile) (PluginData, error) {
	// If no API key is available, use hardcoded fallback values
	if apiKey == "" {
		log.Debug("No API key available, using default plugin data")
		return DefaultPluginData, nil
	}

	log.Debug("API key available, fetching plugin URL")

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
	resp, err := base.MakeRequest(ctx, apiKey, "/v1/stripecli/get-plugin-url", params, make(map[string]interface{}), true, nil)
	if err != nil {
		return PluginData{}, err
	}

	data := PluginData{}
	json.Unmarshal(resp, &data)

	return data, nil
}
