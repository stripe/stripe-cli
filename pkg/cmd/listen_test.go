package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, tt.want, isThinEvent(tt.eventType))
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
			assert.Equal(t, tt.want, lc.getFeatures())
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
			eventsFrom:  eventsFromSelf,
			forwardURL:  "http://localhost:3000",
			wantDirect:  "http://localhost:3000",
			wantConnect: "",
		},
		{
			name:        "@accounts routes to connect",
			eventsFrom:  eventsFromAccounts,
			forwardURL:  "http://localhost:3000",
			wantDirect:  "",
			wantConnect: "http://localhost:3000",
		},
		{
			name:           "@accounts prefers forward-connect-to if set",
			eventsFrom:     eventsFromAccounts,
			forwardConnect: "http://localhost:4000",
			wantDirect:     "",
			wantConnect:    "http://localhost:4000",
		},
		{
			name:        "all routes to both using forward-to",
			eventsFrom:  eventsFromAll,
			forwardURL:  "http://localhost:3000",
			wantDirect:  "http://localhost:3000",
			wantConnect: "http://localhost:3000",
		},
		{
			name:           "all uses forward-connect-to for connect if set",
			eventsFrom:     eventsFromAll,
			forwardURL:     "http://localhost:3000",
			forwardConnect: "http://localhost:4000",
			wantDirect:     "http://localhost:3000",
			wantConnect:    "http://localhost:4000",
		},
		{
			name:        "empty events-from behaves like all",
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
		name    string
		lc      listenCmd
		wantErr string
	}{
		{
			name: "not forwarding: no error even without events",
			lc:   listenCmd{eventsFrom: eventsFromAll},
		},
		{
			name:    "forwarding without specifying events",
			lc:      listenCmd{eventsFrom: eventsFromAll, forwardURL: "http://localhost:3000"},
			wantErr: "must specify events to forward using --events, --all-snapshot, or --all-thin",
		},
		{
			name:    "forwarding to connect without specifying events",
			lc:      listenCmd{eventsFrom: eventsFromAll, forwardConnectURL: "http://localhost:3000"},
			wantErr: "must specify events to forward using --events, --all-snapshot, or --all-thin",
		},
		{
			name: "forwarding with --all-snapshot",
			lc:   listenCmd{eventsFrom: eventsFromAll, forwardURL: "http://localhost:3000", allSnapshot: true},
		},
		{
			name: "forwarding with --all-thin",
			lc:   listenCmd{eventsFrom: eventsFromAll, forwardURL: "http://localhost:3000", allThin: true},
		},
		{
			name: "forwarding specific snapshot events",
			lc: listenCmd{
				eventsFrom: eventsFromAll,
				forwardURL: "http://localhost:3000",
				events:     []string{"charge.captured"},
			},
		},
		{
			name: "forwarding specific thin events",
			lc: listenCmd{
				eventsFrom: eventsFromAll,
				forwardURL: "http://localhost:3000",
				events:     []string{"v1.billing.meter.no_meter_found"},
			},
		},
		{
			name: "snapshot and thin to the same destination",
			lc: listenCmd{
				eventsFrom:  eventsFromAll,
				forwardURL:  "http://localhost:3000",
				allSnapshot: true,
				allThin:     true,
			},
			wantErr: "cannot forward both snapshot and thin events to the same destination",
		},
		{
			name: "mixed specific events to the same destination",
			lc: listenCmd{
				eventsFrom: eventsFromAll,
				forwardURL: "http://localhost:3000",
				events:     []string{"charge.captured", "v1.billing.meter.no_meter_found"},
			},
			wantErr: "cannot forward both snapshot and thin events to the same destination",
		},
		{
			name: "snapshot and thin to the same connect destination",
			lc: listenCmd{
				eventsFrom:        eventsFromAll,
				forwardConnectURL: "http://localhost:3000",
				allSnapshot:       true,
				allThin:           true,
			},
			wantErr: "cannot forward both snapshot and thin events to the same connect destination",
		},
		{
			name: "--forward-connect-to with --events-from @self",
			lc: listenCmd{
				eventsFrom:        eventsFromSelf,
				forwardURL:        "http://localhost:3000",
				forwardConnectURL: "http://localhost:4000",
				allSnapshot:       true,
			},
			wantErr: "--forward-connect-to cannot be used with --events-from @self",
		},
		{
			name: "conflicting destinations with --events-from @accounts",
			lc: listenCmd{
				eventsFrom:        eventsFromAccounts,
				forwardURL:        "http://localhost:3000",
				forwardConnectURL: "http://localhost:4000",
				allSnapshot:       true,
			},
			wantErr: "name the same destination when --events-from is @accounts",
		},
		{
			name: "identical destinations with --events-from @accounts",
			lc: listenCmd{
				eventsFrom:        eventsFromAccounts,
				forwardURL:        "http://localhost:3000",
				forwardConnectURL: "http://localhost:3000",
				allSnapshot:       true,
			},
		},
		{
			name: "--forward-connect-to alone with --events-from @accounts",
			lc: listenCmd{
				eventsFrom:        eventsFromAccounts,
				forwardConnectURL: "http://localhost:4000",
				allThin:           true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lc.validateForwardingConfig()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEvents(t *testing.T) {
	tests := []struct {
		name    string
		events  []string
		wantErr bool
	}{
		{name: "no events is valid", events: []string{}},
		{name: "snapshot event is valid", events: []string{"charge.captured"}},
		{name: "thin event is valid", events: []string{"v1.billing.meter.no_meter_found"}},
		{name: "bare wildcard is rejected", events: []string{"*"}, wantErr: true},
		{name: "wildcard among other events is rejected", events: []string{"charge.captured", "*"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := &listenCmd{events: tt.events}
			err := lc.validateEvents()
			if tt.wantErr {
				assert.ErrorContains(t, err, "--events '*' is no longer supported. Use --all-snapshot")
			} else {
				assert.NoError(t, err)
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
		{name: "@self is valid", value: "@self"},
		{name: "@accounts is valid", value: "@accounts"},
		{name: "all is valid", value: "all"},
		{name: "@everyone is invalid", value: "@everyone", wantErr: true},
		{name: "empty string is invalid", value: "", wantErr: true},
		{name: "self without @ is invalid", value: "self", wantErr: true},
		{name: "accounts without @ is invalid", value: "accounts", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := &listenCmd{eventsFrom: tt.value}
			err := lc.validateEventsFrom()
			if tt.wantErr {
				assert.ErrorContains(t, err, "invalid --events-from value")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEventsFromDefaultsToAll(t *testing.T) {
	lc := newListenCmd()
	require.NoError(t, lc.cmd.ParseFlags([]string{}))
	assert.Equal(t, eventsFromAll, lc.eventsFrom)
	assert.NoError(t, lc.validateEventsFrom())
}

// The deprecated flags keep working: published commands that use them must
// resolve to the same subscription and destinations they did before the redesign.
func TestDeprecatedThinFlagsStillWork(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantSnapshot    []string
		wantThin        []string
		wantThinURL     string
		wantThinConnect string
		wantFeatures    []string
	}{
		{
			// The docs' most common thin invocation.
			name:            "--thin-events wildcard forwards all thin events",
			args:            []string{"--thin-events", "*", "--forward-thin-to", "http://localhost:3000"},
			wantSnapshot:    []string{"*"},
			wantThin:        []string{"*"},
			wantThinURL:     "http://localhost:3000",
			wantThinConnect: "http://localhost:3000",
			wantFeatures:    []string{webhooksWebSocketFeature, destinationsWebSocketFeature},
		},
		{
			name:            "specific thin events",
			args:            []string{"--thin-events", "v1.billing.meter.no_meter_found", "--forward-thin-to", "http://localhost:3000"},
			wantSnapshot:    []string{"*"},
			wantThin:        []string{"v1.billing.meter.no_meter_found"},
			wantThinURL:     "http://localhost:3000",
			wantThinConnect: "http://localhost:3000",
			wantFeatures:    []string{webhooksWebSocketFeature, destinationsWebSocketFeature},
		},
		{
			// --forward-to can't express this, which is why the flags stay.
			name:            "a separate destination per payload style",
			args:            []string{"--events", "charge.succeeded", "--forward-to", "http://a", "--thin-events", "v1.billing.meter.no_meter_found", "--forward-thin-to", "http://b"},
			wantSnapshot:    []string{"charge.succeeded"},
			wantThin:        []string{"v1.billing.meter.no_meter_found"},
			wantThinURL:     "http://b",
			wantThinConnect: "http://b",
			wantFeatures:    []string{webhooksWebSocketFeature, destinationsWebSocketFeature},
		},
		{
			name:            "--forward-thin-connect-to routes only connect thin events",
			args:            []string{"--thin-events", "*", "--forward-thin-connect-to", "http://c"},
			wantSnapshot:    []string{"*"},
			wantThin:        []string{"*"},
			wantThinURL:     "",
			wantThinConnect: "http://c",
			wantFeatures:    []string{webhooksWebSocketFeature, destinationsWebSocketFeature},
		},
		{
			// Before the redesign this forwarded snapshot events and only printed
			// thin ones, since --forward-thin-to was what built a thin route. It
			// keeps doing that instead of erroring on a destination the user never
			// pointed thin events at.
			name:            "--thin-events with a bare --forward-to still forwards only snapshot events",
			args:            []string{"--thin-events", "*", "--forward-to", "http://a"},
			wantSnapshot:    []string{"*"},
			wantThin:        []string{"*"},
			wantThinURL:     "",
			wantThinConnect: "",
			wantFeatures:    []string{webhooksWebSocketFeature, destinationsWebSocketFeature},
		},
		{
			// Values are thin by declaration, so shape doesn't decide.
			name:            "a thin event that doesn't look versioned",
			args:            []string{"--thin-events", "unusual.event.shape"},
			wantSnapshot:    []string{"*"},
			wantThin:        []string{"unusual.event.shape"},
			wantThinURL:     "",
			wantThinConnect: "",
			wantFeatures:    []string{webhooksWebSocketFeature, destinationsWebSocketFeature},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := newListenCmd()
			require.NoError(t, lc.cmd.ParseFlags(tt.args))
			require.NoError(t, lc.validateFlags())

			snapshotEvents, thinEvents := lc.resolveEvents()
			assert.Equal(t, tt.wantSnapshot, snapshotEvents)
			assert.Equal(t, tt.wantThin, thinEvents)

			directURL, connectURL := lc.resolveForwardURLs()
			thinURL, thinConnectURL := lc.resolveThinForwardURLs(directURL, connectURL)
			assert.Equal(t, tt.wantThinURL, thinURL)
			assert.Equal(t, tt.wantThinConnect, thinConnectURL)

			assert.Equal(t, tt.wantFeatures, lc.getFeatures())
		})
	}
}

// Without the deprecated flags, thin events follow --forward-to.
func TestThinForwardURLsDefaultToForwardTo(t *testing.T) {
	lc := newListenCmd()
	require.NoError(t, lc.cmd.ParseFlags([]string{"--all-thin", "--forward-to", "http://a"}))

	directURL, connectURL := lc.resolveForwardURLs()
	thinURL, thinConnectURL := lc.resolveThinForwardURLs(directURL, connectURL)
	assert.Equal(t, "http://a", thinURL)
	assert.Equal(t, "http://a", thinConnectURL)
}

// MarkDeprecated warns on use and keeps the flags out of help, but they still work.
func TestDeprecatedFlagsAreMarkedDeprecated(t *testing.T) {
	for _, name := range []string{"thin-events", "forward-thin-to", "forward-thin-connect-to"} {
		t.Run(name, func(t *testing.T) {
			flag := newListenCmd().cmd.Flags().Lookup(name)
			require.NotNil(t, flag, "deprecated flag should stay registered")
			assert.NotEmpty(t, flag.Deprecated, "flag should warn when used")
			assert.True(t, flag.Hidden, "deprecated flag should not appear in help")
		})
	}
}

func TestValidateFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name: "bare listen",
			args: []string{},
		},
		{
			name: "snapshot events to a destination",
			args: []string{"--events", "charge.captured", "--forward-to", "http://localhost:3000"},
		},
		{
			name: "thin events to a destination",
			args: []string{"--events", "v1.billing.meter.no_meter_found", "--forward-to", "http://localhost:3000"},
		},
		{
			name: "connect events to a destination",
			args: []string{"--events", "customer.created", "--events-from", "@accounts", "--forward-to", "http://localhost:3000"},
		},
		{
			name: "deprecated thin flags satisfy the subscription requirement",
			args: []string{"--thin-events", "*", "--forward-thin-to", "http://localhost:3000"},
		},
		{
			name: "deprecated flags name a separate destination per payload style",
			args: []string{"--events", "charge.succeeded", "--forward-to", "http://a", "--thin-events", "v1.billing.meter.no_meter_found", "--forward-thin-to", "http://b"},
		},
		{
			name:    "deprecated flags pointing both payload styles at one destination",
			args:    []string{"--events", "charge.succeeded", "--forward-to", "http://a", "--thin-events", "v1.billing.meter.no_meter_found", "--forward-thin-to", "http://a"},
			wantErr: "cannot forward both snapshot and thin events to the same destination",
		},
		{
			// The mixed-destination rule applies to what --events subscribes to,
			// not to a deprecated thin subscription that reaches no destination.
			name: "deprecated --thin-events alongside a bare --forward-to",
			args: []string{"--thin-events", "*", "--forward-to", "http://a"},
		},
		{
			name:    "wildcard events",
			args:    []string{"--events", "*", "--forward-to", "http://localhost:3000"},
			wantErr: "--events '*' is no longer supported",
		},
		{
			name:    "invalid events-from",
			args:    []string{"--events-from", "everyone"},
			wantErr: "invalid --events-from value",
		},
		{
			name:    "forwarding without events",
			args:    []string{"--forward-to", "http://localhost:3000"},
			wantErr: "must specify events to forward",
		},
		{
			name:    "mixed payload styles to one destination",
			args:    []string{"--all-snapshot", "--all-thin", "--forward-to", "http://localhost:3000"},
			wantErr: "cannot forward both snapshot and thin events to the same destination",
		},
		{
			name: "--print-secret is exempt from the forwarding rules",
			args: []string{"--print-secret", "--forward-to", "http://localhost:3000"},
		},
		{
			name: "--print-secret with deprecated flags",
			args: []string{"--print-secret", "--forward-thin-to", "http://localhost:3000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := newListenCmd()
			require.NoError(t, lc.cmd.ParseFlags(tt.args))
			err := lc.validateFlags()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
