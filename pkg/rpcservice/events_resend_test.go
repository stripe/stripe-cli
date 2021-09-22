package rpcservice

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

const expectedPath = "/v1/events/evt_12345/retry"

var rawEvent = []byte(`{
	"id": "evt_12345",
	"object": "event",
	"api_version": "2020-08-27",
	"created": 1620858554,
	"data": {
	  "object": {
		"id": "cs_test_12345"
	  }
	},
	"livemode": false,
	"pending_webhooks": 1,
	"request": {
	  "id": null,
	  "idempotency_key": null
	},
	"type": "checkout.session.completed"
}`)

func TestEventsResendReturnsEventPayload(t *testing.T) {
	// Prepare mock Stripe response

	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case expectedPath:
			assert.Equal(t, http.MethodPost, req.Method)
			body := make([]byte, 20)
			n, err := req.Body.Read(body)
			if n == 0 || (err != nil && err != io.EOF) {
				t.Errorf("Failed to read request body")
			}
			assert.Equal(t, "for_stripecli=true", string(body[:n]))
			res.Write(rawEvent)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	baseURL = ts.URL

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Create expected response

	expectedData, err := structpb.NewStruct(map[string]interface{}{
		"object": map[string]interface{}{
			"id": "cs_test_12345",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create expected event data")
	}

	expected := &rpc.EventsResendResponse{
		StripeEvent: &rpc.StripeEvent{
			Id:              "evt_12345",
			ApiVersion:      "2020-08-27",
			Data:            expectedData,
			Request:         &rpc.StripeEvent_Request{},
			Type:            "checkout.session.completed",
			Created:         1620858554,
			Livemode:        false,
			PendingWebhooks: 1,
		},
	}

	// Make request

	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.EventsResend(ctx, &rpc.EventsResendRequest{
		EventId: "evt_12345",
	})

	// Assert

	assert.Nil(t, err)
	assert.Equal(t, expected.StripeEvent.Id, resp.StripeEvent.Id)
	assert.Equal(t, expected.StripeEvent.ApiVersion, resp.StripeEvent.ApiVersion)
	assert.True(t, assert.ObjectsAreEqual(expected.StripeEvent.Data, resp.StripeEvent.Data))
	assert.Equal(t, expected.StripeEvent.Request, resp.StripeEvent.Request)
	assert.Equal(t, expected.StripeEvent.Type, resp.StripeEvent.Type)
	assert.Equal(t, expected.StripeEvent.Created, resp.StripeEvent.Created)
	assert.Equal(t, expected.StripeEvent.Livemode, resp.StripeEvent.Livemode)
	assert.Equal(t, expected.StripeEvent.PendingWebhooks, resp.StripeEvent.PendingWebhooks)
}

func TestEventsResendSucceedsWithAllArgs(t *testing.T) {
	// Prepare mock Stripe response

	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case expectedPath:
			assert.Equal(t, http.MethodPost, req.Method)
			body := make([]byte, 60)
			n, err := req.Body.Read(body)
			if n == 0 || (err != nil && err != io.EOF) {
				t.Errorf("Failed to read request body")
			}
			assert.Equal(t, "foo=bar&webhook_endpoint=we_12345&account=acct_12345", string(body[:n]))
			res.Write(rawEvent)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	baseURL = ts.URL

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Create expected response

	expectedData, err := structpb.NewStruct(map[string]interface{}{
		"object": map[string]interface{}{
			"id": "cs_test_12345",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create expected event data")
	}

	expected := &rpc.EventsResendResponse{
		StripeEvent: &rpc.StripeEvent{
			Id:              "evt_12345",
			ApiVersion:      "2020-08-27",
			Data:            expectedData,
			Request:         &rpc.StripeEvent_Request{},
			Type:            "checkout.session.completed",
			Created:         1620858554,
			Livemode:        false,
			PendingWebhooks: 1,
		},
	}

	// Make request

	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.EventsResend(ctx, &rpc.EventsResendRequest{
		Account:         "acct_12345",
		Data:            []string{"foo=bar"},
		EventId:         "evt_12345",
		Expand:          []string{},
		Idempotency:     "foo",
		Live:            false,
		StripeAccount:   "acct_12345",
		Version:         "2020-08-27",
		WebhookEndpoint: "we_12345",
	})

	// Assert

	assert.Nil(t, err)
	assert.Equal(t, expected.StripeEvent.Id, resp.StripeEvent.Id)
	assert.Equal(t, expected.StripeEvent.ApiVersion, resp.StripeEvent.ApiVersion)
	assert.True(t, assert.ObjectsAreEqual(expected.StripeEvent.Data, resp.StripeEvent.Data))
	assert.Equal(t, expected.StripeEvent.Request, resp.StripeEvent.Request)
	assert.Equal(t, expected.StripeEvent.Type, resp.StripeEvent.Type)
	assert.Equal(t, expected.StripeEvent.Created, resp.StripeEvent.Created)
	assert.Equal(t, expected.StripeEvent.Livemode, resp.StripeEvent.Livemode)
	assert.Equal(t, expected.StripeEvent.PendingWebhooks, resp.StripeEvent.PendingWebhooks)
}

func TestEventsResendReturnsGenericError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case expectedPath:
			res.WriteHeader(http.StatusBadRequest)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	baseURL = ts.URL

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	eventsResendReq := rpc.EventsResendRequest{
		EventId: "evt_12345",
	}

	resp, err := client.EventsResend(ctx, &eventsResendReq)

	assert.Equal(t, status.Error(codes.FailedPrecondition, "Request failed, status=400, body=").Error(), err.Error())
	assert.Nil(t, resp)
}

func TestEventsResendFailsWithoutEventId(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case expectedPath:
			res.WriteHeader(http.StatusBadRequest)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	baseURL = ts.URL

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	eventsResendReq := rpc.EventsResendRequest{}

	resp, err := client.EventsResend(ctx, &eventsResendReq)

	assert.Equal(t, status.Error(codes.InvalidArgument, "Event ID is required").Error(), err.Error())
	assert.Nil(t, resp)
}

func TestEventsResendFailsWithMalformedData(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case expectedPath:
			res.WriteHeader(http.StatusBadRequest)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	baseURL = ts.URL

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	eventsResendReq := rpc.EventsResendRequest{
		EventId: "evt_12345",
		Data:    []string{"malformed"},
	}

	resp, err := client.EventsResend(ctx, &eventsResendReq)

	assert.Equal(t, status.Error(codes.InvalidArgument, "Invalid data argument: malformed").Error(), err.Error())
	assert.Nil(t, resp)
}
