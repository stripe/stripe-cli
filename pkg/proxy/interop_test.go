package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsInteropEvent(t *testing.T) {
	tests := []struct {
		eventType string
		expected  bool
	}{
		{"v1.charge.succeeded", true},
		{"v1.customer.created", true},
		{"charge.succeeded", false},
		{"v1.billing.meter.no_meter_found", false},
		{"v1.billing.meter.error_report_triggered", false},
		{"v2.core.account.created", false},
		{"v2.money_management.transaction.created", false},
		{"v1.not.a.real.event", false},
		{"v1.*", false},
		{"*", false},
		{"", false},
	}

	for _, test := range tests {
		t.Run(test.eventType, func(t *testing.T) {
			require.Equal(t, test.expected, isInteropEvent(test.eventType))
		})
	}
}

// Interop events are identified by prefixing a snapshot event type with "v1.",
// so a thin event that collides with one would be dropped from wildcard
// subscriptions. Guard against the generated event lists growing into one.
func TestIsInteropEventDoesNotMatchThinEvents(t *testing.T) {
	for event := range validThinEvents {
		require.False(t, isInteropEvent(event), "thin event %s is treated as an interop event", event)
	}

	for event := range validPreviewThinEvents {
		require.False(t, isInteropEvent(event), "preview thin event %s is treated as an interop event", event)
	}
}
