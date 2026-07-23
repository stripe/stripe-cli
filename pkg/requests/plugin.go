package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/config"
)

// PluginMetadata contains plugin-specific manifest and binary information.
type PluginMetadata struct {
	BinaryURL      string `json:"binary_url"`
	PluginManifest string `json:"plugin_manifest"`
}

func getPluginMetadataPath(apiKey string) string {
	if apiKey == "" {
		return "/ajax/stripecli/plugins_metadata"
	}

	return "/v1/stripecli/get-plugin-metadata"
}

func getPluginListPath(apiKey string) string {
	if apiKey == "" {
		return "/ajax/stripecli/list-plugins"
	}

	return "/v1/stripecli/list-plugins"
}

func getPluginEndpointBaseURL(apiKey, apiBaseURL, dashboardBaseURL string) string {
	if apiKey == "" && dashboardBaseURL != "" {
		return dashboardBaseURL
	}

	return apiBaseURL
}

// GetPluginMetadata returns plugin-specific manifest and binary information.
// It uses the authenticated endpoint when an API key is available and the
// anonymous endpoint otherwise.
func GetPluginMetadata(ctx context.Context, apiBaseURL, dashboardBaseURL, apiVersion, apiKey string, profile *config.Profile, pluginName, version, os, arch string) (PluginMetadata, error) {
	params := &RequestParameters{
		data:    []string{},
		version: apiVersion,
	}

	metadataBaseURL := getPluginEndpointBaseURL(apiKey, apiBaseURL, dashboardBaseURL)
	metadataPath := getPluginMetadataPath(apiKey)

	log.WithFields(log.Fields{
		"prefix":   "requests.GetPluginMetadata",
		"base_url": metadataBaseURL,
		"endpoint": metadataPath,
		"plugin":   pluginName,
		"version":  version,
		"os":       os,
		"arch":     arch,
	}).Debug("Fetching plugin metadata")

	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     metadataBaseURL,
	}

	resp, err := base.MakeRequest(ctx, apiKey, metadataPath, params, map[string]interface{}{
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

// GetPluginList returns the list of plugins visible to the current caller for
// the requested platform. It uses the authenticated endpoint when an API key is
// available and the anonymous endpoint otherwise.
func GetPluginList(ctx context.Context, apiBaseURL, dashboardBaseURL, apiVersion, apiKey string, profile *config.Profile, os, arch string) ([]byte, error) {
	params := &RequestParameters{
		data:    []string{},
		version: apiVersion,
	}

	listBaseURL := getPluginEndpointBaseURL(apiKey, apiBaseURL, dashboardBaseURL)
	listPath := getPluginListPath(apiKey)

	log.WithFields(log.Fields{
		"prefix":   "requests.GetPluginList",
		"base_url": listBaseURL,
		"endpoint": listPath,
		"os":       os,
		"arch":     arch,
	}).Debug("Fetching plugin list")

	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     listBaseURL,
	}

	resp, err := base.MakeRequest(ctx, apiKey, listPath, params, map[string]interface{}{
		"os":   os,
		"arch": arch,
	}, true, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
