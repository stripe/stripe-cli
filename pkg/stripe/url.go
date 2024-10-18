package stripe

import (
	"errors"
	"regexp"
)

const (
	// DefaultAPIBaseURL is the default base URL for API requests
	DefaultAPIBaseURL   = "https://api.stripe.com"
	qaAPIBaseURL        = "https://qa-api.stripe.com"
	devAPIBaseURLRegexp = `http(s)?:\/\/[A-Za-z0-9\-]+api-mydev.dev.stripe.me`

	// DefaultFilesAPIBaseURL is the default base URL for Files API requsts
	DefaultFilesAPIBaseURL = "https://files.stripe.com"

	// DefaultDashboardBaseURL is the default base URL for dashboard requests
	DefaultDashboardBaseURL   = "https://dashboard.stripe.com"
	qaDashboardBaseURL        = "https://qa-dashboard.stripe.com"
	devDashboardBaseURLRegexp = `http(s)?:\/\/[A-Za-z0-9\-]+manage-mydev\.dev\.stripe\.me`

	// localhostURLRegexp is used in tests
	localhostURLRegexp = `http:\/\/127\.0\.0\.1(:[0-9]+)?`
)

var (
	errInvalidAPIBaseURL       = errors.New("invalid API base URL")
	errInvalidDashboardBaseURL = errors.New("invalid dashboard base URL")
)

func isValid(url string, exactStrings []string, regexpStrings []string) bool {
	for _, s := range exactStrings {
		if url == s {
			return true
		}
	}
	for _, r := range regexpStrings {
		matched, err := regexp.Match(r, []byte(url))
		if err == nil && matched {
			return true
		}
	}
	return false
}

// ValidateAPIBaseURL returns an error if apiBaseURL isn't allowed
func ValidateAPIBaseURL(apiBaseURL string) error {
	exactStrings := []string{DefaultAPIBaseURL, qaAPIBaseURL}
	regexpStrings := []string{devAPIBaseURLRegexp, localhostURLRegexp}
	if isValid(apiBaseURL, exactStrings, regexpStrings) {
		return nil
	}
	return errInvalidAPIBaseURL
}

// ValidateDashboardBaseURL returns an error if dashboardBaseURL isn't allowed
func ValidateDashboardBaseURL(dashboardBaseURL string) error {
	exactStrings := []string{DefaultDashboardBaseURL, qaDashboardBaseURL}
	regexpStrings := []string{devDashboardBaseURLRegexp, localhostURLRegexp}
	if isValid(dashboardBaseURL, exactStrings, regexpStrings) {
		return nil
	}
	return errInvalidDashboardBaseURL
}
