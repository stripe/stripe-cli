package requests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// ErrCodePluginRequiresNewerCLI is the error.code the plugin metadata endpoints
// send when the requested plugin version declares a minimum core CLI version this
// CLI does not meet.
//
// Such a release is otherwise hidden, so the request would 404 -- and a 404 says
// only that the version does not exist, which is both wrong and the signal that
// sends the CLI off to its cached metadata.
const ErrCodePluginRequiresNewerCLI = "plugin_requires_newer_cli"

// PluginMetadata contains plugin-specific manifest and binary information.
type PluginMetadata struct {
	BinaryURL      string `json:"binary_url"`
	PluginManifest string `json:"plugin_manifest"`
	// AutoInstall reports whether the server wants this plugin installed on first
	// use without asking. The endpoints always send the field, and a response that
	// omits it decodes to false, which is the prompting behavior the CLI has always
	// had — so losing the signal can only cost a prompt, never cause a surprise
	// download.
	AutoInstall bool `json:"auto_install"`
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
//
// machineUUID is the CLI's persistent per-installation identifier. The server
// keys the auto-install rollout on it so a machine stays on the same side of that
// rollout across invocations rather than flipping between prompting and not.
func GetPluginMetadata(ctx context.Context, apiBaseURL, dashboardBaseURL, apiVersion, apiKey string, profile *config.Profile, pluginName, version, os, arch, machineUUID string) (PluginMetadata, error) {
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
		// Logged so a machine that is not being auto-installed to can be told apart
		// from one that never sent the identifier the rollout is keyed on.
		"machine_uuid": machineUUID,
	}).Debug("Fetching plugin metadata")

	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     metadataBaseURL,
	}

	resolvedCreds, err := profile.ResolveCredentialsForAnyMode(base.Livemode)
	if err != nil {
		resolvedCreds = stripe.NewAPIKeyCredentials(apiKey)
	}

	requestParams := map[string]interface{}{
		"plugin":  pluginName,
		"version": version,
		"os":      os,
		"arch":    arch,
	}
	// Left out when there is no uuid to send: the server reads absent and empty
	// identically, as "this caller keeps prompting".
	if machineUUID != "" {
		requestParams["machine_uuid"] = machineUUID
	}

	resp, err := base.MakeRequest(ctx, resolvedCreds, metadataPath, params, requestParams, true, nil)
	if err != nil {
		return PluginMetadata{}, err
	}

	metadata := PluginMetadata{}
	if err := json.Unmarshal(resp, &metadata); err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to decode plugin metadata response: %w", err)
	}

	return metadata, nil
}

// PluginRequiresNewerCLI reports whether err is a plugin metadata response saying
// the requested plugin needs a newer core CLI, along with the minimum version it
// names. The endpoints answer this way for a version the caller named, and for a
// request that named none when every release the plugin has needs a newer CLI --
// reporting the lowest minimum among them, since that is the nearest version that
// would make any of them installable.
//
// The minimum version rides on the error body as an extra attribute rather than in
// RequestError, which is shared by every Stripe API request. An answer that names
// no minimum is still the same answer, so ok is true with an empty version: the
// caller reports the upgrade without naming a target rather than losing the reason.
func PluginRequiresNewerCLI(err error) (minCoreVersion string, ok bool) {
	var requestErr RequestError
	if !errors.As(err, &requestErr) {
		return "", false
	}

	if requestErr.StatusCode != http.StatusBadRequest || requestErr.ErrorCode != ErrCodePluginRequiresNewerCLI {
		return "", false
	}

	body, isString := requestErr.Body.(string)
	if !isString {
		return "", true
	}

	var errorBody struct {
		Error struct {
			MinCoreVersion string `json:"min_core_version"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &errorBody); err != nil {
		return "", true
	}

	return errorBody.Error.MinCoreVersion, true
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

	resolvedCreds, err := profile.ResolveCredentialsForAnyMode(base.Livemode)
	if err != nil {
		resolvedCreds = stripe.NewAPIKeyCredentials(apiKey)
	}

	resp, err := base.MakeRequest(ctx, resolvedCreds, listPath, params, map[string]interface{}{
		"os":   os,
		"arch": arch,
	}, true, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
