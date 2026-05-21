package agentguidance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToday(t *testing.T) {
	tests := []struct {
		name       string
		override   string
		assertFunc func(t *testing.T, got time.Time)
	}{
		{
			name:     "no_override",
			override: "",
			assertFunc: func(t *testing.T, got time.Time) {
				assert.WithinDuration(t, time.Now(), got, 5*time.Second)
			},
		},
		{
			name:     "valid_override",
			override: "2026-12-25",
			assertFunc: func(t *testing.T, got time.Time) {
				assert.Equal(t, 2026, got.Year())
				assert.Equal(t, time.December, got.Month())
				assert.Equal(t, 25, got.Day())
			},
		},
		{
			name:     "malformed_override_falls_back",
			override: "not-a-date",
			assertFunc: func(t *testing.T, got time.Time) {
				assert.WithinDuration(t, time.Now(), got, 5*time.Second)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STRIPE_AGENT_GUIDANCE_TODAY", tc.override)
			got := Today()
			tc.assertFunc(t, got)
		})
	}
}
