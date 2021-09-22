package rpcservice

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/rpc"
)

func TestAllowRequestIfHeaderPresent(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	_, err = client.Version(ctx, &rpc.VersionRequest{})
	assert.Equal(t, nil, err)
}

func TestRejectRequestIfHeaderAbsent(t *testing.T) {
	md := metadata.New(map[string]string{"foo-bar": "1"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	_, err = client.Version(ctx, &rpc.VersionRequest{})
	expected := status.Errorf(codes.Unauthenticated, fmt.Sprintf("%s header is not supplied", requiredHeader))

	assert.Equal(t, expected.Error(), err.Error())
}

func TestRejectRequestIfMetadataEmpty(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	_, err = client.Version(ctx, &rpc.VersionRequest{})
	expected := status.Errorf(codes.Unauthenticated, fmt.Sprintf("%s header is not supplied", requiredHeader))

	assert.Equal(t, expected.Error(), err.Error())
}
