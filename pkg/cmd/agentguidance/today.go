package agentguidance

import (
	"os"
	"time"
)

// Today returns the current local date, or a value parsed from the
// STRIPE_AGENT_GUIDANCE_TODAY override env var when set. The override
// is for E2E testing; not advertised to users.
func Today() time.Time {
	if override := os.Getenv("STRIPE_AGENT_GUIDANCE_TODAY"); override != "" {
		if t, err := time.ParseInLocation("2006-01-02", override, time.Local); err == nil {
			return t
		}
	}
	return time.Now()
}
