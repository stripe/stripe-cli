package rpcservice

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
)

const customerPath = "/v1/customers"
const customerWithIDPath = "/v1/customers/cust_12345"
const customerPayload = `{"id": "cust_12345", "foo": "bar"}`
const planPath = "/v1/plans"
const subscriptionPath = "/v1/subscriptions"

func TestTriggerSucceedsWithSupportedEvent(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case customerPath:
			res.Write([]byte(customerPayload))
		case customerWithIDPath:
			// Do nothing, we just want to verify this request came in
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	baseURL = ts.URL

	resp, err := client.Trigger(ctx, &rpc.TriggerRequest{
		Event: "customer.deleted",
	})

	expected := rpc.TriggerResponse{
		Requests: []string{"customer", "customer_deleted"},
	}

	assert.Nil(t, err)
	assert.Equal(t, expected.Requests, resp.Requests)
}

func TestTriggerSucceedsWithStripeAccount(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case customerPath:
			res.Write([]byte(customerPayload))
		case customerWithIDPath:
			// Do nothing, we just want to verify this request came in
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	baseURL = ts.URL

	resp, err := client.Trigger(ctx, &rpc.TriggerRequest{
		Event:         "customer.deleted",
		StripeAccount: "acct_123",
	})

	expected := rpc.TriggerResponse{
		Requests: []string{"customer", "customer_deleted"},
	}

	assert.Nil(t, err)
	assert.Equal(t, expected.Requests, resp.Requests)
}

func TestTriggerFailsWithUnsupportedEvent(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	baseURL = "foo"

	resp, err := client.Trigger(ctx, &rpc.TriggerRequest{
		Event: "bar",
	})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestTriggerFailsWithEmptyEvent(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	baseURL = "foo"

	resp, err := client.Trigger(ctx, &rpc.TriggerRequest{})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestTriggerSucceedsWithFixtureFlags(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		switch url := req.URL.String(); url {
		case customerPath:
			require.True(t, strings.Contains(string(body), "name=testUser"))
			require.True(t, strings.Contains(string(body), "email=testEmail"))
		case planPath:
			require.True(t, strings.Contains(string(body), "amount=500"))
		case subscriptionPath:
			require.False(t, strings.Contains(string(body), "description"))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	baseURL = ts.URL

	resp, err := client.Trigger(ctx, &rpc.TriggerRequest{
		Event:         "customer.subscription.created",
		StripeAccount: "acct_123",
		Skip:          []string{},
		Override:      []string{"customer:name=testUser", "plan:amount=500"},
		Add:           []string{"customer:email=testEmail"},
		Remove:        []string{"customer:description"},
	})

	expected := rpc.TriggerResponse{
		Requests: []string{"customer", "plan", "subscription"},
	}

	assert.Nil(t, err)
	assert.Equal(t, expected.Requests, resp.Requests)
}
