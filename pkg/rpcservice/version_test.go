package rpcservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
)

func TestVersionReturnsCLIVersion(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.Version(ctx, &rpc.VersionRequest{})
	if err != nil {
		t.Fatalf("Version failed: %v", err)
	}

	expected := rpc.VersionResponse{
		Version: "master",
	}

	assert.Equal(t, expected.Version, resp.Version)
}
