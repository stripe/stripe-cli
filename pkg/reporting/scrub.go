package reporting

import (
	"regexp"
	"strings"

	sentry "github.com/getsentry/sentry-go"
)

var (
	// Matches sk_live_*, sk_test_*, rk_live_*, rk_test_*, pk_live_*, pk_test_*
	apiKeyPattern = regexp.MustCompile(`\b((?:sk|rk|pk)_(?:live|test)_)[a-zA-Z0-9_]+\b`)
	// Matches oak_* (OAuth access keys — no live/test segment)
	oauthKeyPattern = regexp.MustCompile(`\b(oak_)[a-zA-Z0-9_]+\b`)
	// Matches whsec_* (base64 alphabet + optional = padding)
	webhookSecretPattern = regexp.MustCompile(`\bwhsec_[a-zA-Z0-9+/]+=*`)
	// Matches email addresses
	emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
)

func redactSensitiveStrings(s string) string {
	s = apiKeyPattern.ReplaceAllString(s, "${1}[REDACTED]")
	s = oauthKeyPattern.ReplaceAllString(s, "${1}[REDACTED]")
	s = webhookSecretPattern.ReplaceAllString(s, "whsec_[REDACTED]")
	s = emailPattern.ReplaceAllString(s, "[REDACTED]")
	return s
}

func scrubEvent(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	event.Modules = nil
	event.ServerName = ""

	event.Message = redactSensitiveStrings(event.Message)

	for i := range event.Exception {
		event.Exception[i].Value = redactSensitiveStrings(event.Exception[i].Value)
		if st := event.Exception[i].Stacktrace; st != nil {
			// Drop internal reporting wrapper frames (frames are outermost-first,
			// so the innermost call — our CaptureException — is at the end).
			for len(st.Frames) > 0 && strings.HasSuffix(st.Frames[len(st.Frames)-1].Module, "/pkg/reporting") {
				st.Frames = st.Frames[:len(st.Frames)-1]
			}
		}
	}

	for i := range event.Breadcrumbs {
		event.Breadcrumbs[i].Message = redactSensitiveStrings(event.Breadcrumbs[i].Message)
		for k, v := range event.Breadcrumbs[i].Data {
			if s, ok := v.(string); ok {
				event.Breadcrumbs[i].Data[k] = redactSensitiveStrings(s)
			}
		}
	}

	for k, v := range event.Tags {
		event.Tags[k] = redactSensitiveStrings(v)
	}

	return event
}
