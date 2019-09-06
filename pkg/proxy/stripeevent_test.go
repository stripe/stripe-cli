package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsConnect(t *testing.T) {
	evt1 := &stripeEvent{ID: "evt_123", Type: "customer.created"}
	require.False(t, evt1.isConnect())

	evt2 := &stripeEvent{ID: "evt_123", Type: "customer.created", Account: "acct_123"}
	require.True(t, evt2.isConnect())
}

func TestURLForEventID(t *testing.T) {
	evt := &stripeEvent{ID: "evt_123", Livemode: false, Type: "customer.created"}
	require.Equal(t, "https://dashboard.stripe.com/test/events/evt_123", evt.urlForEventID())

	evt = &stripeEvent{ID: "evt_123", Livemode: false, Type: "customer.created", Account: "acct_123"}
	require.Equal(t, "https://dashboard.stripe.com/acct_123/test/events/evt_123", evt.urlForEventID())

	evt = &stripeEvent{ID: "evt_123", Livemode: true, Type: "customer.created"}
	require.Equal(t, "https://dashboard.stripe.com/events/evt_123", evt.urlForEventID())

	evt = &stripeEvent{ID: "evt_123", Livemode: true, Type: "customer.created", Account: "acct_123"}
	require.Equal(t, "https://dashboard.stripe.com/acct_123/events/evt_123", evt.urlForEventID())
}

func TestURLForEventType(t *testing.T) {
	evt := &stripeEvent{ID: "evt_123", Livemode: false, Type: "customer.created"}
	require.Equal(t, "https://dashboard.stripe.com/test/events?type=customer.created", evt.urlForEventType())

	evt = &stripeEvent{ID: "evt_123", Livemode: false, Type: "customer.created", Account: "acct_123"}
	require.Equal(t, "https://dashboard.stripe.com/acct_123/test/events?type=customer.created", evt.urlForEventType())

	evt = &stripeEvent{ID: "evt_123", Livemode: true, Type: "customer.created"}
	require.Equal(t, "https://dashboard.stripe.com/events?type=customer.created", evt.urlForEventType())

	evt = &stripeEvent{ID: "evt_123", Livemode: true, Type: "customer.created", Account: "acct_123"}
	require.Equal(t, "https://dashboard.stripe.com/acct_123/events?type=customer.created", evt.urlForEventType())
}
