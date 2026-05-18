package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/config"
)

// PluginData contains the plugin download information
type PluginData struct {
	PluginBaseURL       string   `json:"base_url"`
	AdditionalManifests []string `json:"additional_manifests,omitempty"`
}

// PluginMetadata contains plugin-specific manifest and binary information.
type PluginMetadata struct {
	BinaryURL      string `json:"binary_url"`
	PluginManifest string `json:"plugin_manifest"`
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
	if err := json.Unmarshal(resp, &data); err != nil {
		return PluginData{}, fmt.Errorf("failed to decode plugin data response: %w", err)
	}

	return data, nil
}

// GetPluginMetadata returns plugin-specific manifest and binary information.
func GetPluginMetadata(ctx context.Context, baseURL, apiVersion, apiKey string, profile *config.Profile, pluginName, version, os, arch string) (PluginMetadata, error) {
	if apiKey == "" {
		return PluginMetadata{}, fmt.Errorf("plugin metadata endpoint requires an API key")
	}

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

	resp, err := base.MakeRequest(ctx, apiKey, "/v1/stripecli/get-plugin-metadata", params, map[string]interface{}{
		"plugin":  pluginName,
		"version": version,
		"os":      os,
		"arch":    arch,
	}, true, nil)
	if err != nil {
		return PluginMetadata{}, err
	}

	metadata := PluginMetadata{}
	if err := json.Unmarshal(resp, &metadata); err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to decode plugin metadata response: %w", err)
	}

	return metadata, nil
}
