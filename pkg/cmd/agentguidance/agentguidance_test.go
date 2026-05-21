package agentguidance

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMaybeEmit(t *testing.T) {
	today := time.Date(2026, 5, 21, 12, 0, 0, 0, time.Local)
	todayISO := "2026-05-21"
	yesterdayISO := "2026-05-20"

	noEnv := func(string) string { return "" }
	claudeEnv := func(k string) string {
		if k == "CLAUDECODE" {
			return "1"
		}
		return ""
	}

	tests := []struct {
		name          string
		getEnv        func(string) string
		snoozedUntil  string
		args          []string
		expectMessage bool
	}{
		{
			name:          "not_an_agent_silent",
			getEnv:        noEnv,
			snoozedUntil:  "",
			args:          []string{"customers", "list"},
			expectMessage: false,
		},
		{
			name:          "agent_writes_message",
			getEnv:        claudeEnv,
			snoozedUntil:  "",
			args:          []string{"customers", "list"},
			expectMessage: true,
		},
		{
			name:          "snoozed_today_silent",
			getEnv:        claudeEnv,
			snoozedUntil:  todayISO,
			args:          []string{"customers", "list"},
			expectMessage: false,
		},
		{
			name:          "stale_snooze_writes",
			getEnv:        claudeEnv,
			snoozedUntil:  yesterdayISO,
			args:          []string{"customers", "list"},
			expectMessage: true,
		},
		{
			name:          "suppressed_agent_guidance",
			getEnv:        claudeEnv,
			snoozedUntil:  "",
			args:          []string{"agent-guidance", "snooze"},
			expectMessage: false,
		},
		{
			name:          "garbage_snooze_value_writes",
			getEnv:        claudeEnv,
			snoozedUntil:  "not-a-date",
			args:          []string{"customers", "list"},
			expectMessage: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			MaybeEmit(tc.getEnv, &buf, tc.snoozedUntil, today, tc.args)
			if tc.expectMessage {
				assert.Contains(t, buf.String(), "Stripe CLI Agent Guidance")
				assert.Contains(t, buf.String(), "stripe spec search")
				assert.Contains(t, buf.String(), "stripe agent-guidance snooze")
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}
