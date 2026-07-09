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
		{"v1.", true},
		{"charge.captured", false},
		{"customer.created", false},
		{"payment_intent.succeeded", false},
		{"*", false},
		{"", false},
		{"v1", false},
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
		allSnapshot  bool
		allThin      bool
		wantSnapshot []string
		wantThin     []string
	}{
		{
			name:         "bare listen with no flags subscribes to everything",
			events:       []string{},
			wantSnapshot: []string{"*"},
			wantThin:     []string{"*"},
		},
		{
			name:         "all-snapshot adds wildcard for snapshot",
			allSnapshot:  true,
			events:       []string{},
			wantSnapshot: []string{"*"},
			wantThin:     nil,
		},
		{
			name:         "all-thin adds wildcard for thin",
			allThin:      true,
			events:       []string{},
			wantSnapshot: nil,
			wantThin:     []string{"*"},
		},
		{
			name:         "both all-snapshot and all-thin",
			allSnapshot:  true,
			allThin:      true,
			events:       []string{},
			wantSnapshot: []string{"*"},
			wantThin:     []string{"*"},
		},
		{
			name:         "all-snapshot with specific thin events",
			allSnapshot:  true,
			events:       []string{"v1.billing.meter.no_meter_found"},
			wantSnapshot: []string{"*"},
			wantThin:     []string{"v1.billing.meter.no_meter_found"},
		},
		{
			name:         "all-thin with specific snapshot events",
			allThin:      true,
			events:       []string{"charge.captured"},
			wantSnapshot: []string{"charge.captured"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, thin := splitEventsByType(tt.events, tt.allSnapshot, tt.allThin)
			assert.Equal(t, tt.wantSnapshot, snapshot)
			assert.Equal(t, tt.wantThin, thin)
		})
	}
}

func TestGetFeatures(t *testing.T) {
	tests := []struct {
		name        string
		events      []string
		allSnapshot bool
		allThin     bool
		want        []string
	}{
		{
			name: "bare listen opens both channels",
			want: []string{webhooksWebSocketFeature, destinationsWebSocketFeature},
		},
		{
			name:        "all-snapshot opens webhooks only",
			allSnapshot: true,
			want:        []string{webhooksWebSocketFeature},
		},
		{
			name:    "all-thin opens v2_events only",
			allThin: true,
			want:    []string{destinationsWebSocketFeature},
		},
		{
			name:        "both all flags open both channels",
			allSnapshot: true,
			allThin:     true,
			want:        []string{webhooksWebSocketFeature, destinationsWebSocketFeature},
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
			lc := &listenCmd{
				events:      tt.events,
				allSnapshot: tt.allSnapshot,
				allThin:     tt.allThin,
			}
			got := lc.getFeatures()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMergeAndSplitEvents(t *testing.T) {
	tests := []struct {
		name           string
		events         []string
		thinEvents     []string
		eventsExplicit bool
		wantSnapshot   []string
		wantThin       []string
	}{
		{
			name:           "thin-events without explicit --events: snapshot wildcard + specific thin",
			events:         []string{"*"},
			thinEvents:     []string{"v1.billing.meter.no_meter_found"},
			eventsExplicit: false,
			wantSnapshot:   []string{"*"},
			wantThin:       []string{"v1.billing.meter.no_meter_found"},
		},
		{
			name:           "thin-events with explicit --events: merged and split",
			events:         []string{"charge.captured"},
			thinEvents:     []string{"v1.billing.meter.no_meter_found"},
			eventsExplicit: true,
			wantSnapshot:   []string{"charge.captured"},
			wantThin:       []string{"v1.billing.meter.no_meter_found"},
		},
		{
			name:           "thin-events with explicit wildcard --events: all of both",
			events:         []string{"*"},
			thinEvents:     []string{"v1.billing.meter.no_meter_found"},
			eventsExplicit: true,
			wantSnapshot:   []string{"*"},
			wantThin:       []string{"*", "v1.billing.meter.no_meter_found"},
		},
		{
			name:           "no thin-events: normal split",
			events:         []string{"charge.captured"},
			thinEvents:     []string{},
			eventsExplicit: true,
			wantSnapshot:   []string{"charge.captured"},
			wantThin:       nil,
		},
		{
			name:           "no thin-events with wildcard: both channels",
			events:         []string{"*"},
			thinEvents:     []string{},
			eventsExplicit: false,
			wantSnapshot:   []string{"*"},
			wantThin:       []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, thin := mergeAndSplitEvents(tt.events, tt.thinEvents, tt.eventsExplicit, false, false)
			assert.Equal(t, tt.wantSnapshot, snapshot)
			assert.Equal(t, tt.wantThin, thin)
		})
	}
}

func TestResolveForwardURLs(t *testing.T) {
	tests := []struct {
		name           string
		eventsFrom     string
		forwardURL     string
		forwardConnect string
		wantDirect     string
		wantConnect    string
	}{
		{
			name:        "@self routes to direct only",
			eventsFrom:  "@self",
			forwardURL:  "http://localhost:3000",
			wantDirect:  "http://localhost:3000",
			wantConnect: "",
		},
		{
			name:        "@accounts routes to connect",
			eventsFrom:  "@accounts",
			forwardURL:  "http://localhost:3000",
			wantDirect:  "",
			wantConnect: "http://localhost:3000",
		},
		{
			name:           "@accounts prefers forward-connect-to if set",
			eventsFrom:     "@accounts",
			forwardURL:     "http://localhost:3000",
			forwardConnect: "http://localhost:4000",
			wantDirect:     "",
			wantConnect:    "http://localhost:4000",
		},
		{
			name:        "all routes to both using forward-to",
			eventsFrom:  "all",
			forwardURL:  "http://localhost:3000",
			wantDirect:  "http://localhost:3000",
			wantConnect: "http://localhost:3000",
		},
		{
			name:           "all uses forward-connect-to for connect if set",
			eventsFrom:     "all",
			forwardURL:     "http://localhost:3000",
			forwardConnect: "http://localhost:4000",
			wantDirect:     "http://localhost:3000",
			wantConnect:    "http://localhost:4000",
		},
		{
			name:        "default (empty) behaves like all",
			eventsFrom:  "",
			forwardURL:  "http://localhost:3000",
			wantDirect:  "http://localhost:3000",
			wantConnect: "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := &listenCmd{
				eventsFrom:        tt.eventsFrom,
				forwardURL:        tt.forwardURL,
				forwardConnectURL: tt.forwardConnect,
			}
			direct, connect := lc.resolveForwardURLs()
			assert.Equal(t, tt.wantDirect, direct)
			assert.Equal(t, tt.wantConnect, connect)
		})
	}
}
