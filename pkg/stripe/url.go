package stripe

import (
	"errors"
	"regexp"
)

// DefaultAPIBaseURL is the default base URL for API requests
const DefaultAPIBaseURL = "https://api.stripe.com"

// qaAPIBaseURL is the base URL for API requests in QA
const qaAPIBaseURL = "https://qa-api.stripe.com"

// devAPIBaseURLRegexp is the base URL for API requests in dev
const devAPIBaseURLRegexp = `http(s)?:\/\/[A-Za-z0-9\-]+api-mydev.dev.stripe.me`

// DefaultFilesAPIBaseURL is the default base URL for Files API requsts
const DefaultFilesAPIBaseURL = "https://files.stripe.com"

// DefaultDashboardBaseURL is the default base URL for dashboard requests
const DefaultDashboardBaseURL = "https://dashboard.stripe.com"

// qaDashboardBaseURL is the base URL for dashboard requests in QA
const qaDashboardBaseURL = "https://qa-dashboard.stripe.com"

// devDashboardBaseURLRegexp is the base URL for dashboard requests in dev
const devDashboardBaseURLRegexp = `http(s)?:\/\/[A-Za-z0-9\-]+manage-mydev.dev.stripe.me`

var errInvalidAPIBaseURL = errors.New("invalid API base URL")
var errInvalidDashboardBaseURL = errors.New("invalid dashboard base URL")

// ValidateAPIBaseUrl returns an error if apiBaseURL isn't allowed
func ValidateAPIBaseUrl(apiBaseURL string) error {
	if apiBaseURL == DefaultAPIBaseURL {
		return nil
	}
	if apiBaseURL == qaAPIBaseURL {
		return nil
	}
	matched, err := regexp.Match(devAPIBaseURLRegexp, []byte(apiBaseURL))
	if err != nil {
		return errInvalidAPIBaseURL
	}
	if !matched {
		return errInvalidAPIBaseURL
	}
	return nil
}

// ValidateDashboardBaseURL returns true if dashboardBaseURL is allowed
func ValidateDashboardBaseURL(dashboardBaseURL string) error {
	if dashboardBaseURL == DefaultDashboardBaseURL {
		return nil
	}
	if dashboardBaseURL == qaDashboardBaseURL {
		return nil
	}
	matched, err := regexp.Match(devDashboardBaseURLRegexp, []byte(dashboardBaseURL))
	if err != nil {
		return errInvalidDashboardBaseURL
	}
	if !matched {
		return errInvalidDashboardBaseURL
	}
	return nil
}
