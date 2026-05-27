package stripe

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

const (
	// DefaultAPIBaseURL is the default base URL for API requests
	DefaultAPIBaseURL   = "https://api.stripe.com"
	APIBaseURLRegexp    = `^https:\/\/api\.stripe\.com\/v\d+$`
	qaAPIBaseURL        = "https://qa-api.stripe.com"
	devAPIBaseURLRegexp = `http(s)?:\/\/[A-Za-z0-9\-]+.dev.stripe.me`

	// DefaultFilesAPIBaseURL is the default base URL for Files API requsts
	DefaultFilesAPIBaseURL = "https://files.stripe.com/"

	// DefaultDashboardBaseURL is the default base URL for dashboard requests
	DefaultDashboardBaseURL   = "https://dashboard.stripe.com"
	qaDashboardBaseURL        = "https://qa-dashboard.stripe.com"
	devDashboardBaseURLRegexp = `http(s)?:\/\/[A-Za-z0-9\-]+\.dev\.stripe\.me`

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
	exactStrings := []string{DefaultAPIBaseURL, qaAPIBaseURL, DefaultFilesAPIBaseURL}
	regexpStrings := []string{APIBaseURLRegexp, devAPIBaseURLRegexp, localhostURLRegexp}
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

// DashboardBaseURLForAPIBaseURL derives the matching dashboard base URL for a
// given API base URL. This keeps dev and QA hosts aligned without requiring a
// separate dashboard override in the common case.
func DashboardBaseURLForAPIBaseURL(apiBaseURL string) string {
	if apiBaseURL == "" {
		return DefaultDashboardBaseURL
	}

	parsedBaseURL, err := url.Parse(apiBaseURL)
	if err != nil || parsedBaseURL.Host == "" {
		return DefaultDashboardBaseURL
	}

	switch {
	case parsedBaseURL.Host == "api.stripe.com":
		parsedBaseURL.Host = "dashboard.stripe.com"
	case parsedBaseURL.Host == "qa-api.stripe.com":
		parsedBaseURL.Host = "qa-dashboard.stripe.com"
	case strings.Contains(parsedBaseURL.Host, "--api-dev.dev.stripe.me"):
		parsedBaseURL.Host = strings.Replace(parsedBaseURL.Host, "--api-dev.dev.stripe.me", "--dashboard-dev.dev.stripe.me", 1)
	case strings.Contains(parsedBaseURL.Host, "--api-iso.dev.stripe.me"):
		parsedBaseURL.Host = strings.Replace(parsedBaseURL.Host, "--api-iso.dev.stripe.me", "--dashboard-iso.dev.stripe.me", 1)
	default:
		parsedBaseURL.Host = strings.Replace(parsedBaseURL.Host, "api-", "manage-", 1)
	}

	parsedBaseURL.Path = ""
	parsedBaseURL.RawPath = ""
	parsedBaseURL.RawQuery = ""
	parsedBaseURL.Fragment = ""

	return parsedBaseURL.String()
}
