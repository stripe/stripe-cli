package testutil

import (
	"regexp"
)

// Patterns for sensitive data that should not appear in logs
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`sk_test_[a-zA-Z0-9]{10,}`),     // Test API keys
	regexp.MustCompile(`sk_live_[a-zA-Z0-9]{10,}`),     // Live API keys (should never appear, but just in case)
	regexp.MustCompile(`rk_test_[a-zA-Z0-9]{10,}`),     // Restricted keys
	regexp.MustCompile(`rk_live_[a-zA-Z0-9]{10,}`),     // Restricted keys
	regexp.MustCompile(`whsec_[a-zA-Z0-9]{10,}`),       // Webhook signing secrets
	regexp.MustCompile(`sk_[a-zA-Z0-9]{20,}`),          // Generic secret keys
	regexp.MustCompile(`"access_token"\s*:\s*"[^"]+"`), // Access tokens in JSON
	regexp.MustCompile(`Bearer\s+[a-zA-Z0-9_-]{20,}`),  // Bearer tokens
}

// SanitizeOutput removes sensitive data from strings before logging.
// This helps prevent accidental secret exposure in CI logs.
func SanitizeOutput(s string) string {
	result := s
	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Keep the prefix for debugging, redact the rest
			if len(match) > 8 {
				return match[:8] + "***REDACTED***"
			}
			return "***REDACTED***"
		})
	}
	return result
}

// SanitizeResult sanitizes both stdout and stderr of a Result.
func SanitizeResult(r *Result) *Result {
	if r == nil {
		return nil
	}
	return &Result{
		Stdout:   SanitizeOutput(r.Stdout),
		Stderr:   SanitizeOutput(r.Stderr),
		ExitCode: r.ExitCode,
	}
}
