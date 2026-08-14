package login

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAccessBaseURL_AllowsExactProdAndQAOrigins(t *testing.T) {
	assert.NoError(t, ValidateAccessBaseURL(DefaultAccessBaseURL))
	assert.NoError(t, ValidateAccessBaseURL(QAAccessBaseURL))
}

func TestValidateAccessBaseURL_RejectsArbitraryBases(t *testing.T) {
	cases := map[string]string{
		"http scheme":             "http://access.stripe.com",
		"userinfo":                "https://attacker@access.stripe.com",
		"unexpected port":         "https://access.stripe.com:8443",
		"path suffix":             "https://access.stripe.com/../attacker.com",
		"query string":            "https://access.stripe.com?redirect=evil.com",
		"fragment":                "https://access.stripe.com#evil.com",
		"lookalike host suffix":   "https://access.stripe.com.evil.com",
		"lookalike host prefix":   "https://evil-access.stripe.com",
		"arbitrary attacker host": "https://evil.example.com",
		"trailing slash":          "https://access.stripe.com/",
		"empty string":            "",
		"local dev override":      "http://localhost:8080",
		"qa lookalike":            "https://qa-access.stripe.com.evil.com",
	}

	for name, base := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, ValidateAccessBaseURL(base))
		})
	}
}

func TestValidateBrowserURL_AllowsMatchingDashboardHost(t *testing.T) {
	assert.NoError(t, validateBrowserURL("https://dashboard.stripe.com/verify", DefaultAccessBaseURL))
	assert.NoError(t, validateBrowserURL("https://qa-dashboard.stripe.com/verify", QAAccessBaseURL))
}

func TestValidateBrowserURL_RejectsMismatchedOrUntrustedURLs(t *testing.T) {
	cases := map[string]struct {
		url           string
		accessBaseURL string
	}{
		"wrong scheme":               {"http://dashboard.stripe.com/verify", DefaultAccessBaseURL},
		"arbitrary host":             {"https://evil.example.com/verify", DefaultAccessBaseURL},
		"qa dashboard used for prod": {"https://qa-dashboard.stripe.com/verify", DefaultAccessBaseURL},
		"prod dashboard used for qa": {"https://dashboard.stripe.com/verify", QAAccessBaseURL},
		"unparseable URL":            {"https://[::1", DefaultAccessBaseURL},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, validateBrowserURL(tc.url, tc.accessBaseURL))
		})
	}
}
