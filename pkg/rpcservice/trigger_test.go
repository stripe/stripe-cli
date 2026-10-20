package rpcservice

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const customerPath = "/v1/customers"
const customerWithIDPath = "/v1/customers/cust_12345"
const customerPayload = `{"id": "cust_12345", "foo": "bar"}`
const productPath = "/v1/products"
const pricePath = "/v1/prices"
const subscriptionPath = "/v1/subscriptions"

func TestTriggerCredentialDestinationProtection(t *testing.T) {
	requests := make(chan struct{}, 4)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- struct{}{}
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()
	oldBaseURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = oldBaseURL }()
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)
	for _, destination := range []string{"https://evil.example", ts.URL} {
		for _, token := range []string{"wrong-token", testDaemonToken} {
			t.Run(destination+"/"+token, func(t *testing.T) {
				ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(requiredHeader, token))
				raw := fmt.Sprintf(`{"_meta":{"template_version":0},"fixtures":[{"name":"request","path":"/v1/products","method":"post","api_base":%q}]}`, destination)
				resp, err := client.Trigger(ctx, &rpc.TriggerRequest{Raw: raw})
				require.Nil(t, resp)
				if token != testDaemonToken {
					require.Equal(t, codes.Unauthenticated, status.Code(err))
				} else {
					require.ErrorContains(t, err, "invalid API base URL")
				}
				require.Empty(t, requests)
			})
		}
	}
}

func TestTriggerRejectsLiveOverride(t *testing.T) {
	t.Setenv("STRIPE_API_KEY", "sk_live_example")
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	resp, err := rpc.NewStripeCLIClient(conn).Trigger(withAuth(context.Background()), &rpc.TriggerRequest{Event: "product.created"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.ErrorContains(t, err, "API key mode does not match")
	require.Nil(t, resp)
}

func TestTriggerSucceedsWithSupportedEvent(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		case productPath:
			require.True(t, strings.Contains(string(body), "name=myproduct"))
		case pricePath:
			require.True(t, strings.Contains(string(body), "unit_amount=500"))
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
		Override:      []string{"customer:name=testUser", "price:unit_amount=500"},
		Add:           []string{"customer:email=testEmail"},
		Remove:        []string{"customer:description"},
	})

	expected := rpc.TriggerResponse{
		Requests: []string{"customer", "product", "price", "subscription"},
	}

	assert.Nil(t, err)
	assert.Equal(t, expected.Requests, resp.Requests)
}
