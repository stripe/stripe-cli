package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsConnect(t *testing.T) {
	evt1 := &StripeEvent{ID: "evt_123", Type: "customer.created"}
	require.False(t, evt1.IsConnect())

	evt2 := &StripeEvent{ID: "evt_123", Type: "customer.created", Account: "acct_123"}
	require.True(t, evt2.IsConnect())
}

func TestUrlForEventID(t *testing.T) {
	evt := &StripeEvent{ID: "evt_123", Livemode: false, Type: "customer.created"}
	require.Equal(t, "https://dashboard.stripe.com/test/events/evt_123", evt.URLForEventID())

	evt = &StripeEvent{ID: "evt_123", Livemode: false, Type: "customer.created", Account: "acct_123"}
	require.Equal(t, "https://dashboard.stripe.com/acct_123/test/events/evt_123", evt.URLForEventID())

	evt = &StripeEvent{ID: "evt_123", Livemode: true, Type: "customer.created"}
	require.Equal(t, "https://dashboard.stripe.com/events/evt_123", evt.URLForEventID())

	evt = &StripeEvent{ID: "evt_123", Livemode: true, Type: "customer.created", Account: "acct_123"}
	require.Equal(t, "https://dashboard.stripe.com/acct_123/events/evt_123", evt.URLForEventID())
}

func TestURLForEventType(t *testing.T) {
	evt := &StripeEvent{ID: "evt_123", Livemode: false, Type: "customer.created"}
	require.Equal(t, "https://dashboard.stripe.com/test/events?type=customer.created", evt.URLForEventType())

	evt = &StripeEvent{ID: "evt_123", Livemode: false, Type: "customer.created", Account: "acct_123"}
	require.Equal(t, "https://dashboard.stripe.com/acct_123/test/events?type=customer.created", evt.URLForEventType())

	evt = &StripeEvent{ID: "evt_123", Livemode: true, Type: "customer.created"}
	require.Equal(t, "https://dashboard.stripe.com/events?type=customer.created", evt.URLForEventType())

	evt = &StripeEvent{ID: "evt_123", Livemode: true, Type: "customer.created", Account: "acct_123"}
	require.Equal(t, "https://dashboard.stripe.com/acct_123/events?type=customer.created", evt.URLForEventType())
}
