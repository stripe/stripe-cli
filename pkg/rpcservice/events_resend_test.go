package rpcservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockStripeReq struct {
}

var makeRequest func(apiKey string, path string, params *requests.RequestParameters, errOnStatus bool) ([]byte, error)

func (m *mockStripeReq) MakeRequest(apiKey string, path string, params *requests.RequestParameters, errOnStatus bool) ([]byte, error) {
	return makeRequest(apiKey, path, params, errOnStatus)
}

func TestEventsResendReturnsEventPayload(t *testing.T) {
	getStripeReq = func() IStripeReq {
		makeRequest = func(apiKey, path string, params *requests.RequestParameters, errOnStatus bool) ([]byte, error) {
			return []byte("event payload"), nil
		}
		return &mockStripeReq{}
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	eventsResendReq := rpc.EventsResendRequest{}

	resp, err := client.EventsResend(ctx, &eventsResendReq)

	assert.Nil(t, err)
	assert.Equal(t, "event payload", resp.Payload)
}

func TestEventsResendSucceedsWithAllArgs(t *testing.T) {
	getStripeReq = func() IStripeReq {
		makeRequest = func(apiKey, path string, params *requests.RequestParameters, errOnStatus bool) ([]byte, error) {
			return []byte("event payload"), nil
		}
		return &mockStripeReq{}
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	eventsResendReq := rpc.EventsResendRequest{
		Account:         "acct_12345",
		Data:            []string{"foo=bar"},
		EventId:         "evt_12345",
		Expand:          []string{},
		Idempotency:     "foo",
		Live:            false,
		ShowHeaders:     false,
		StripeAccount:   "acct_12345",
		Version:         "foo",
		WebhookEndpoint: "foo",
	}

	resp, err := client.EventsResend(ctx, &eventsResendReq)

	assert.Nil(t, err)
	assert.Equal(t, "event payload", resp.Payload)
}

func TestEventsResendReturnsGenericError(t *testing.T) {
	getStripeReq = func() IStripeReq {
		makeRequest = func(apiKey, path string, params *requests.RequestParameters, errOnStatus bool) ([]byte, error) {
			return nil, errors.New("my error")
		}
		return &mockStripeReq{}
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	eventsResendReq := rpc.EventsResendRequest{}

	resp, err := client.EventsResend(ctx, &eventsResendReq)

	assert.Equal(t, status.Error(codes.Unknown, "my error").Error(), err.Error())
	assert.Nil(t, resp)
}

func TestEventsResendFailsWithMalformedData(t *testing.T) {
	getStripeReq = func() IStripeReq {
		makeRequest = func(apiKey, path string, params *requests.RequestParameters, errOnStatus bool) ([]byte, error) {
			return nil, errors.New("my error")
		}
		return &mockStripeReq{}
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	eventsResendReq := rpc.EventsResendRequest{
		Data: []string{"malformed"},
	}

	resp, err := client.EventsResend(ctx, &eventsResendReq)

	assert.Equal(t, status.Error(codes.InvalidArgument, "Invalid data argument: malformed").Error(), err.Error())
	assert.Nil(t, resp)
}
