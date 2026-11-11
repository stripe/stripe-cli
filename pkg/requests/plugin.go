package requests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

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
	// AutoUpdateDefault enables automatic updates when the user has not chosen a
	// per-plugin or global setting. Older responses that omit it default to false.
	AutoUpdateDefault bool `json:"auto_update_default"`
	// MinPluginVersions holds the minimum versions of other plugins the resolved
	// release declares it needs, keyed by plugin shortname (e.g. {"apps": "1.17.0"}).
	// Each value is a bare x.y.z minimum — a floor, never a range, and never the
	// plugin itself. A response that omits the field decodes to nil, which means
	// the release declares no requirements and everything behaves as it did before
	// the field existed.
	MinPluginVersions map[string]string `json:"min_plugin_versions"`
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

// getFromPluginEndpoint sends a GET to one of the plugin endpoints, which come in
// pairs: an authenticated endpoint on the API host for a caller with an API key,
// and an anonymous one on the Dashboard host for a caller without one. pathFor
// picks which path of the pair a key gets, as getPluginMetadataPath does.
//
// With anonymousOnExpiredKey, a key the API reports as expired is retried
// against the anonymous endpoint. That endpoint is all a logged-out caller ever
// gets, so without the retry a lapsed key would fail requests that logging out
// lets through. The retry carries no credentials at all, so the expired key
// never reaches the Dashboard host.
//
// Callers opt in because the anonymous endpoint sees less than the account
// does. That is fine for installing a plugin, but an upgrade or a listing
// answered from it would quietly stand in for the account's view: an upgrade
// could move a release only the account can see back to an older public one,
// and a listing would drop plugins without saying why. Those callers get the
// expired key instead, which tells the user to log in again.
//
// If the anonymous endpoint cannot find the plugin either, the expired key is
// what gets reported, since logging in again is the fix: it may be a plugin only
// a logged-in account can see. Any other failure is reported as it is -- an
// outage, a timeout, or a refusal because this CLI is too old -- since no login
// would change it and telling the user to log in would only send them in a
// circle.
//
// A key found expired is remembered for the rest of the process (see
// expiredAPIKeys), so later lookups that may fall back go straight to the
// anonymous endpoint rather than repeating a request certain to fail.
func getFromPluginEndpoint(ctx context.Context, apiBaseURL, dashboardBaseURL, apiVersion, apiKey string, profile *config.Profile, pathFor func(apiKey string) string, requestParams map[string]interface{}, anonymousOnExpiredKey bool) ([]byte, error) {
	authenticatedBaseURL := getPluginEndpointBaseURL(apiKey, apiBaseURL, dashboardBaseURL)
	expiredKey := expiredAPIKey{baseURL: authenticatedBaseURL, apiKey: apiKey}

	var err error
	if knownErr, known := expiredAPIKeys.Load(expiredKey); known && anonymousOnExpiredKey && apiKey != "" {
		err = knownErr.(error)
	} else {
		resolvedCreds, credsErr := profile.ResolveCredentialsForAnyMode(false)
		if credsErr != nil {
			resolvedCreds = stripe.NewAPIKeyCredentials(apiKey)
		}

		var resp []byte
		resp, err = sendPluginEndpointRequest(ctx, profile, resolvedCreds, authenticatedBaseURL, pathFor(apiKey), apiVersion, requestParams)
		if apiKey != "" && IsAPIKeyExpiredError(err) {
			expiredAPIKeys.Store(expiredKey, err)
		}
		if !anonymousOnExpiredKey || apiKey == "" || !IsAPIKeyExpiredError(err) {
			return resp, err
		}
	}

	anonymousBaseURL := getPluginEndpointBaseURL("", apiBaseURL, dashboardBaseURL)
	anonymousPath := pathFor("")
	log.WithFields(log.Fields{
		"prefix":   "requests.getFromPluginEndpoint",
		"base_url": anonymousBaseURL,
		"endpoint": anonymousPath,
	}).Debug("API key has expired, retrying with the anonymous plugin endpoint")

	// No profile either. Given one, a 401 here could have the request layer
	// refresh the profile's OAuth token and retry with it, putting credentials
	// back on a request meant to carry none.
	resp, anonymousErr := sendPluginEndpointRequest(ctx, nil, stripe.Credentials{}, anonymousBaseURL, anonymousPath, apiVersion, requestParams)
	if anonymousErr == nil {
		return resp, nil
	}

	var requestErr RequestError
	if !errors.As(anonymousErr, &requestErr) || requestErr.StatusCode != http.StatusNotFound {
		return nil, anonymousErr
	}

	log.WithFields(log.Fields{
		"prefix": "requests.getFromPluginEndpoint",
	}).Debugf("anonymous plugin endpoint could not find the plugin either, reporting the expired API key: %s", anonymousErr)

	return nil, err
}

// expiredAPIKey identifies a key on the API host that reported it expired; the
// host is part of it so a key expired against one --api-base says nothing about
// another.
type expiredAPIKey struct {
	baseURL string
	apiKey  string
}

// expiredAPIKeys maps each expiredAPIKey seen this process to the error the API
// answered with, which is what gets reported if the anonymous endpoint cannot
// help. A key only ever lapses one way, and logging in again replaces the key
// rather than reviving it, so nothing is ever removed.
var expiredAPIKeys sync.Map

func sendPluginEndpointRequest(ctx context.Context, profile *config.Profile, creds stripe.Credentials, baseURL, path, apiVersion string, requestParams map[string]interface{}) ([]byte, error) {
	base := &Base{
		Profile:        profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     baseURL,
	}

	params := &RequestParameters{
		data:    []string{},
		version: apiVersion,
	}

	return base.MakeRequest(ctx, creds, path, params, requestParams, true, nil)
}

// GetPluginMetadata returns plugin-specific manifest and binary information.
// It uses the authenticated endpoint when an API key is available and the
// anonymous endpoint otherwise. With anonymousOnExpiredKey it also falls back to
// the anonymous endpoint once the API reports the key as expired; see
// getFromPluginEndpoint for which callers should ask for that.
func GetPluginMetadata(ctx context.Context, apiBaseURL, dashboardBaseURL, apiVersion, apiKey string, profile *config.Profile, pluginName, version, os, arch string, anonymousOnExpiredKey bool) (PluginMetadata, error) {
	log.WithFields(log.Fields{
		"prefix":   "requests.GetPluginMetadata",
		"base_url": getPluginEndpointBaseURL(apiKey, apiBaseURL, dashboardBaseURL),
		"endpoint": getPluginMetadataPath(apiKey),
		"plugin":   pluginName,
		"version":  version,
		"os":       os,
		"arch":     arch,
	}).Debug("Fetching plugin metadata")

	resp, err := getFromPluginEndpoint(ctx, apiBaseURL, dashboardBaseURL, apiVersion, apiKey, profile, getPluginMetadataPath, map[string]interface{}{
		"plugin":  pluginName,
		"version": version,
		"os":      os,
		"arch":    arch,
	}, anonymousOnExpiredKey)
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
// available and the anonymous endpoint otherwise. An expired key is reported
// rather than retried anonymously, since the anonymous list would silently
// leave out plugins only the account can see; see getFromPluginEndpoint.
func GetPluginList(ctx context.Context, apiBaseURL, dashboardBaseURL, apiVersion, apiKey string, profile *config.Profile, os, arch string) ([]byte, error) {
	log.WithFields(log.Fields{
		"prefix":   "requests.GetPluginList",
		"base_url": getPluginEndpointBaseURL(apiKey, apiBaseURL, dashboardBaseURL),
		"endpoint": getPluginListPath(apiKey),
		"os":       os,
		"arch":     arch,
	}).Debug("Fetching plugin list")

	return getFromPluginEndpoint(ctx, apiBaseURL, dashboardBaseURL, apiVersion, apiKey, profile, getPluginListPath, map[string]interface{}{
		"os":   os,
		"arch": arch,
	}, false)
}
