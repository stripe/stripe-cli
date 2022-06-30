package rpcservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
)

func TestFixturesReturnsData(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.Fixture(ctx, &rpc.FixtureRequest{Event: "customer.created"})

	expected := rpc.FixtureResponse{
		Fixture: `{
  "_meta": {
    "template_version": 0,
    "exclude_metadata": false
  },
  "fixtures": [
    {
      "name": "customer",
      "expected_error_type": "",
      "path": "/v1/customers",
      "method": "post",
      "params": {
        "description": "(created by Stripe CLI)"
      }
    }
  ],
  "env": null
}`,
	}

	assert.Nil(t, err)
	assert.Equal(t, expected.Fixture, resp.Fixture)
}
