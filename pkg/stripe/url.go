package stripe

import (
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const (
	// DefaultAPIBaseURL is the default base URL for API requests
	DefaultAPIBaseURL = "https://api.stripe.com"
	qaAPIBaseURL      = "https://qa-api.stripe.com"

	// DefaultFilesAPIBaseURL is the default base URL for Files API requsts
	DefaultFilesAPIBaseURL = "https://files.stripe.com/"

	// DefaultDashboardBaseURL is the default base URL for dashboard requests
	DefaultDashboardBaseURL = "https://dashboard.stripe.com"
	qaDashboardBaseURL      = "https://qa-dashboard.stripe.com"

	// devHostSuffix matches Stripe's per-developer dev domains, e.g.
	// foo-lv5r9y--api-mydev.dev.stripe.me
	devHostSuffix = ".dev.stripe.me"
)

var (
	errInvalidAPIBaseURL       = errorcategory.New(errorcategory.UserInput, "invalid API base URL")
	errInvalidDashboardBaseURL = errorcategory.New(errorcategory.UserInput, "invalid dashboard base URL")

	devHostLabelRegexp   = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	apiVersionPathRegexp = regexp.MustCompile(`^/v\d+$`)
)

// parseBaseURL parses raw as a URL and rejects forms that let the parsed
// host diverge from what a quick read of the string would suggest, namely
// userinfo (`user@host`), which is never legitimate in a base URL flag.
func parseBaseURL(raw string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Hostname() == "" {
		return nil, false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, false
	}
	return u, true
}

func isDevHost(host string) bool {
	label, ok := strings.CutSuffix(strings.ToLower(host), devHostSuffix)
	return ok && devHostLabelRegexp.MatchString(label)
}

func isLoopbackHost(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ValidateAPIBaseURL returns an error if apiBaseURL isn't allowed
func ValidateAPIBaseURL(apiBaseURL string) error {
	if apiBaseURL == DefaultAPIBaseURL || apiBaseURL == qaAPIBaseURL || apiBaseURL == DefaultFilesAPIBaseURL {
		return nil
	}

	u, ok := parseBaseURL(apiBaseURL)
	if !ok {
		return errInvalidAPIBaseURL
	}

	host := strings.ToLower(u.Hostname())
	switch {
	case host == "api.stripe.com" && u.Scheme == "https" && apiVersionPathRegexp.MatchString(u.Path):
		return nil
	case isDevHost(host):
		return nil
	case isLoopbackHost(host) && u.Scheme == "http":
		return nil
	}

	return errInvalidAPIBaseURL
}

// ValidateDashboardBaseURL returns an error if dashboardBaseURL isn't allowed
func ValidateDashboardBaseURL(dashboardBaseURL string) error {
	if dashboardBaseURL == DefaultDashboardBaseURL || dashboardBaseURL == qaDashboardBaseURL {
		return nil
	}

	u, ok := parseBaseURL(dashboardBaseURL)
	if !ok {
		return errInvalidDashboardBaseURL
	}

	host := strings.ToLower(u.Hostname())
	switch {
	case isDevHost(host):
		return nil
	case isLoopbackHost(host) && u.Scheme == "http":
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
