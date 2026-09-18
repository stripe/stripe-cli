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

func TestValidateForwardingConfig(t *testing.T) {
	tests := []struct {
		name        string
		lc          listenCmd
		snapshot    []string
		thin        []string
		directURL   string
		thinURL     string
		connectURL  string
		thinConnURL string
		wantErr     string
	}{
		{
			name:    "no forwarding: no error even without events",
			lc:      listenCmd{},
			wantErr: "",
		},
		{
			name:    "forwarding without specifying events: error",
			lc:      listenCmd{forwardURL: "http://localhost:3000"},
			wantErr: "must specify events to forward",
		},
		{
			name:      "forwarding with --all-snapshot: no error",
			lc:        listenCmd{forwardURL: "http://localhost:3000", allSnapshot: true},
			snapshot:  []string{"*"},
			directURL: "http://localhost:3000",
			thinURL:   "",
			wantErr:   "",
		},
		{
			name:      "forwarding with --all-thin: no error",
			lc:        listenCmd{forwardURL: "http://localhost:3000", allThin: true},
			thin:      []string{"*"},
			directURL: "",
			thinURL:   "http://localhost:3000",
			wantErr:   "",
		},
		{
			name:      "forwarding with --events: no error",
			lc:        listenCmd{forwardURL: "http://localhost:3000", events: []string{"charge.captured"}},
			snapshot:  []string{"charge.captured"},
			directURL: "http://localhost:3000",
			thinURL:   "http://localhost:3000",
			wantErr:   "",
		},
		{
			name:      "snapshot and thin to same destination: error",
			lc:        listenCmd{forwardURL: "http://localhost:3000", allSnapshot: true, allThin: true},
			snapshot:  []string{"*"},
			thin:      []string{"*"},
			directURL: "http://localhost:3000",
			thinURL:   "http://localhost:3000",
			wantErr:   "cannot forward both snapshot and thin events to the same destination",
		},
		{
			name:      "snapshot and thin to different destinations: no error",
			lc:        listenCmd{forwardURL: "http://localhost:3000", allSnapshot: true, allThin: true},
			snapshot:  []string{"*"},
			thin:      []string{"*"},
			directURL: "http://localhost:3000",
			thinURL:   "http://localhost:4000",
			wantErr:   "",
		},
		{
			name:        "connect: snapshot and thin to same destination: error",
			lc:          listenCmd{forwardConnectURL: "http://localhost:3000", allSnapshot: true, allThin: true},
			snapshot:    []string{"*"},
			thin:        []string{"*"},
			connectURL:  "http://localhost:3000",
			thinConnURL: "http://localhost:3000",
			wantErr:     "cannot forward both snapshot and thin events to the same connect destination",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lc.validateForwardingConfig(tt.snapshot, tt.thin, tt.directURL, tt.thinURL, tt.connectURL, tt.thinConnURL)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEventsFrom(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "@self is valid", value: "@self", wantErr: false},
		{name: "@accounts is valid", value: "@accounts", wantErr: false},
		{name: "all is valid", value: "all", wantErr: false},
		{name: "@everyone is invalid", value: "@everyone", wantErr: true},
		{name: "empty string is invalid", value: "", wantErr: true},
		{name: "self without @ is invalid", value: "self", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := newListenCmd()
			lc.eventsFrom = tt.value
			err := lc.validateEventsFrom()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid --events-from value")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckRemovedFlags(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		value   string
		wantErr string
	}{
		{
			name:    "--thin-events returns helpful error",
			flag:    "thin-events",
			value:   "v1.billing.meter.no_meter_found",
			wantErr: "--thin-events has been replaced by --events",
		},
		{
			name:    "--forward-thin-to returns helpful error",
			flag:    "forward-thin-to",
			value:   "http://localhost:3000",
			wantErr: "--forward-thin-to has been replaced by --forward-to",
		},
		{
			name:    "--forward-thin-connect-to returns helpful error",
			flag:    "forward-thin-connect-to",
			value:   "http://localhost:3000",
			wantErr: "--forward-thin-connect-to has been replaced by --forward-connect-to",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := newListenCmd()
			lc.cmd.Flags().Set(tt.flag, tt.value)
			err := lc.checkRemovedFlags()
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
