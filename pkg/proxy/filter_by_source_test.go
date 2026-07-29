package proxy

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/websocket"
)

func newProcessorWithEventsFrom(eventsFrom string) *WebhookEventProcessor {
	return &WebhookEventProcessor{
		cfg: &WebhookEventProcessorConfig{
			Log:        log.StandardLogger(),
			OutCh:      make(chan websocket.IElement, 1),
			EventsFrom: eventsFrom,
		},
	}
}

func TestFilterBySource(t *testing.T) {
	tests := []struct {
		name       string
		eventsFrom string
		account    string
		wantSkip   bool
	}{
		{
			name:       "@self skips connect events",
			eventsFrom: "@self",
			account:    "acct_123",
			wantSkip:   true,
		},
		{
			name:       "@self allows self events",
			eventsFrom: "@self",
			account:    "",
			wantSkip:   false,
		},
		{
			name:       "@accounts skips self events",
			eventsFrom: "@accounts",
			account:    "",
			wantSkip:   true,
		},
		{
			name:       "@accounts allows connect events",
			eventsFrom: "@accounts",
			account:    "acct_123",
			wantSkip:   false,
		},
		{
			name:       "all allows self events",
			eventsFrom: "all",
			account:    "",
			wantSkip:   false,
		},
		{
			name:       "all allows connect events",
			eventsFrom: "all",
			account:    "acct_123",
			wantSkip:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProcessorWithEventsFrom(tt.eventsFrom)
			evt := &StripeEvent{Account: tt.account}
			assert.Equal(t, tt.wantSkip, p.filterBySource(evt))
		})
	}
}

func TestFilterByV2Source(t *testing.T) {
	tests := []struct {
		name       string
		eventsFrom string
		context    string
		wantSkip   bool
	}{
		{
			name:       "@self skips connect v2 events",
			eventsFrom: "@self",
			context:    "acct_123",
			wantSkip:   true,
		},
		{
			name:       "@self allows self v2 events",
			eventsFrom: "@self",
			context:    "",
			wantSkip:   false,
		},
		{
			name:       "@accounts skips self v2 events",
			eventsFrom: "@accounts",
			context:    "",
			wantSkip:   true,
		},
		{
			name:       "@accounts allows connect v2 events",
			eventsFrom: "@accounts",
			context:    "acct_123",
			wantSkip:   false,
		},
		{
			name:       "all allows self v2 events",
			eventsFrom: "all",
			context:    "",
			wantSkip:   false,
		},
		{
			name:       "all allows connect v2 events",
			eventsFrom: "all",
			context:    "acct_123",
			wantSkip:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProcessorWithEventsFrom(tt.eventsFrom)
			evt := &V2EventPayload{Context: tt.context}
			assert.Equal(t, tt.wantSkip, p.filterByV2Source(evt))
		})
	}
}
