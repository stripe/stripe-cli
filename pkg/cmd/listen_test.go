package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsThinEvent(t *testing.T) {
	tests := []struct {
		eventType string
		want      bool
	}{
		{"v1.billing.meter.no_meter_found", true},
		{"v2.core.account.created", true},
		{"v1.some.event", true},
		{"charge.captured", false},
		{"customer.created", false},
		{"payment_intent.succeeded", false},
		{"*", false},
		{"", false},
		{"v1", false},
		{"v1.", true},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			got := isThinEvent(tt.eventType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSplitEventsByType(t *testing.T) {
	tests := []struct {
		name         string
		events       []string
		wantSnapshot []string
		wantThin     []string
	}{
		{
			name:         "wildcard splits to both",
			events:       []string{"*"},
			wantSnapshot: []string{"*"},
			wantThin:     []string{"*"},
		},
		{
			name:         "snapshot events only",
			events:       []string{"charge.captured", "customer.created"},
			wantSnapshot: []string{"charge.captured", "customer.created"},
			wantThin:     nil,
		},
		{
			name:         "thin events only",
			events:       []string{"v1.billing.meter.no_meter_found", "v2.core.account.created"},
			wantSnapshot: nil,
			wantThin:     []string{"v1.billing.meter.no_meter_found", "v2.core.account.created"},
		},
		{
			name:         "mixed events",
			events:       []string{"charge.captured", "v1.billing.meter.no_meter_found"},
			wantSnapshot: []string{"charge.captured"},
			wantThin:     []string{"v1.billing.meter.no_meter_found"},
		},
		{
			name:         "empty events returns nil",
			events:       []string{},
			wantSnapshot: nil,
			wantThin:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, thin := splitEventsByType(tt.events)
			assert.Equal(t, tt.wantSnapshot, snapshot)
			assert.Equal(t, tt.wantThin, thin)
		})
	}
}

func TestGetFeatures(t *testing.T) {
	tests := []struct {
		name   string
		events []string
		want   []string
	}{
		{
			name:   "wildcard opens both channels",
			events: []string{"*"},
			want:   []string{webhooksWebSocketFeature, destinationsWebSocketFeature},
		},
		{
			name:   "snapshot events only opens webhooks",
			events: []string{"charge.captured"},
			want:   []string{webhooksWebSocketFeature},
		},
		{
			name:   "thin events only opens v2_events",
			events: []string{"v1.billing.meter.no_meter_found"},
			want:   []string{destinationsWebSocketFeature},
		},
		{
			name:   "mixed events opens both",
			events: []string{"charge.captured", "v1.billing.meter.no_meter_found"},
			want:   []string{webhooksWebSocketFeature, destinationsWebSocketFeature},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := &listenCmd{events: tt.events}
			got := lc.getFeatures()
			assert.Equal(t, tt.want, got)
		})
	}
}
