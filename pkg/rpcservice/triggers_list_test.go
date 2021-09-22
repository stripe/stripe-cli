package rpcservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/fixtures"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
)

func TestTriggersListReturnsEvents(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.TriggersList(ctx, &rpc.TriggersListRequest{})

	expected := rpc.TriggersListResponse{
		Events: fixtures.EventNames(),
	}

	assert.Nil(t, err)
	assert.Equal(t, expected.Events, resp.Events)
}
