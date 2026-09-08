package login

import (
	"net/http"
	"net/url"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

var errInvalidAccessBaseURL = errorcategory.New(errorcategory.UserInput,
	"invalid access base URL: must be exactly https://access.stripe.com or https://qa-access.stripe.com")

// ValidateAccessBaseURL returns an error unless accessBaseURL is exactly the
// production or QA access-srv origin.
//
// Requests built from this value carry the OAuth user access token
// (Authorization: Bearer) or the stored refresh token (form body), so unlike
// --api-base/--dashboard-base there is no dev-host, localhost, or path
// allowance: any scheme, host, port, userinfo, path, query, or fragment
// other than these two exact strings risks sending live credentials to an
// attacker-controlled destination.
func ValidateAccessBaseURL(accessBaseURL string) error {
	if accessBaseURL == DefaultAccessBaseURL || accessBaseURL == QAAccessBaseURL {
		return nil
	}
	return errInvalidAccessBaseURL
}

// accessSrvHTTPClient is used for every request to access-srv. Redirects are
// disabled: an access-srv redirect response could otherwise cause the
// Authorization header or POST body (which may carry the OAuth access or
// refresh token) to be forwarded to a different, redirect-controlled origin.
var accessSrvHTTPClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// trustedBrowserHosts returns the hosts that may legitimately appear in a
// browser URL for the given access-srv origin: the access-srv host itself
// (device-code verification URIs are hosted there) and the corresponding
// dashboard host (reauthentication URIs are hosted there).
func trustedBrowserHosts(accessBaseURL string) []string {
	if accessBaseURL == QAAccessBaseURL {
		return []string{"qa-access.stripe.com", "qa-dashboard.stripe.com"}
	}
	return []string{"access.stripe.com", "dashboard.stripe.com"}
}

// validateBrowserURL returns an error unless rawURL is an https URL on a host
// trusted for accessBaseURL. access-srv responses that embed a browser URL
// (device-code verification, reauth) are checked here before the CLI
// displays or opens them, so a compromised or misconfigured access-srv can't
// redirect the user's browser to an arbitrary destination.
func validateBrowserURL(rawURL, accessBaseURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return errorcategory.Errorf(errorcategory.Auth, "access-srv returned an unparseable URL: %s", rawURL)
	}
	if u.Scheme != "https" {
		return errorcategory.Errorf(errorcategory.Auth, "refusing to display or open untrusted URL returned by access-srv: %s", rawURL)
	}
	for _, host := range trustedBrowserHosts(accessBaseURL) {
		if u.Host == host {
			return nil
		}
	}
	return errorcategory.Errorf(errorcategory.Auth, "refusing to display or open untrusted URL returned by access-srv: %s", rawURL)
}
