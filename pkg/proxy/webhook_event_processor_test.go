package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/websocket"
)

func TestProcessV2EventFiltersEventTypes(t *testing.T) {
	tests := []struct {
		name       string
		thinEvents []string
		eventType  string
		expectSent bool
	}{
		{"wildcard sends thin events", []string{"*"}, "v2.core.account.created", true},
		{"wildcard sends preview thin events", []string{"*"}, "v2.money_management.transaction.created", true},
		{"wildcard sends v1 thin events without a snapshot counterpart", []string{"*"}, "v1.billing.meter.no_meter_found", true},
		{"wildcard skips interop events", []string{"*"}, "v1.charge.succeeded", false},
		{"explicit interop event is sent", []string{"v1.charge.succeeded"}, "v1.charge.succeeded", true},
		{"explicit interop event does not enable other interop events", []string{"v1.charge.succeeded"}, "v1.customer.created", false},
		{"unsubscribed thin event is skipped", []string{"v2.core.account.created"}, "v2.core.account.updated", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outCh := make(chan websocket.IElement, 1)
			var ackedEventIDs []string

			p := NewWebhookEventProcessor(
				func(msg *websocket.OutgoingMessage) {
					ackedEventIDs = append(ackedEventIDs, msg.EventID)
				},
				nil,
				&WebhookEventProcessorConfig{
					Log:        &log.Logger{Out: io.Discard},
					ThinEvents: test.thinEvents,
					OutCh:      outCh,
				},
			)

			p.processV2Event(&websocket.StripeV2Event{
				Payload:            fmt.Sprintf(`{"id":"evt_123","type":%q}`, test.eventType),
				EventDestinationID: "we_123",
			})

			// events are acknowledged whether or not they're forwarded, so
			// Stripe doesn't retry the ones we filter out
			require.Equal(t, []string{"evt_123"}, ackedEventIDs)

			if !test.expectSent {
				require.Empty(t, outCh)
				return
			}

			require.Len(t, outCh, 1)
			data, ok := (<-outCh).(websocket.DataElement)
			require.True(t, ok)
			evt, ok := data.Data.(V2EventPayload)
			require.True(t, ok)
			require.Equal(t, test.eventType, evt.Type)
		})
	}
}

func TestProcessV2EventDoesNotForwardInteropEvents(t *testing.T) {
	forwarded := make(chan string, 2)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		forwarded <- string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// give the endpoint response handler room so its writes don't block
	outCh := make(chan websocket.IElement, 8)

	p := NewWebhookEventProcessor(
		func(msg *websocket.OutgoingMessage) {},
		[]EndpointRoute{{
			URL:                ts.URL,
			EventTypes:         []string{"*"},
			IsEventDestination: true,
		}},
		&WebhookEventProcessorConfig{
			Log:        &log.Logger{Out: io.Discard},
			ThinEvents: []string{"*"},
			OutCh:      outCh,
			Timeout:    5,
		},
	)

	p.processV2Event(&websocket.StripeV2Event{Payload: `{"id":"evt_interop","type":"v1.charge.succeeded"}`})
	p.processV2Event(&websocket.StripeV2Event{Payload: `{"id":"evt_thin","type":"v2.core.account.created"}`})

	select {
	case body := <-forwarded:
		// the interop event was processed first, so the thin event arriving
		// first means the interop event was never forwarded
		require.Contains(t, body, "evt_thin")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the thin event to be forwarded")
	}

	require.Empty(t, forwarded)
}
