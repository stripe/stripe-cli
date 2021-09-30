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

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
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

func TestUpdateContextWithTelemetry(t *testing.T) {
	telemetryClient := &stripe.NoOpTelemetryClient{}
	config := &Config{
		UserCfg: &config.Config{
			Profile: config.Profile{
				AccountID: "acct_xxx",
			},
		}}
	rpcService := New(config, telemetryClient)

	// Add grpc Metadata to context
	md := metadata.Pairs("user-agent", "unit_test")

	ctx := metadata.NewIncomingContext(context.Background(), md)

	newCtx := updateContextWithTelemetry(ctx, "method", rpcService)

	eventMetadata := stripe.GetEventMetadata(newCtx)
	assert.NotNil(t, eventMetadata)
	assert.Equal(t, eventMetadata.Merchant, "acct_xxx")
	assert.Equal(t, eventMetadata.CommandPath, "method")
	assert.Equal(t, eventMetadata.UserAgent, "unit_test")
	assert.Equal(t, stripe.GetTelemetryClient(newCtx), telemetryClient)
}

func TestGetUserAgentFromGRPCMetadata(t *testing.T) {
	// No grpc metadata
	assert.Equal(t, getUserAgentFromGrpcMetadata(context.Background()), "")
}

func TestGetUserAgentFromGRPCMetadataWithNoUserAgent(t *testing.T) {
	// no user-agent key
	md := metadata.Pairs("hello", "world")

	ctx := metadata.NewIncomingContext(context.Background(), md)
	assert.Equal(t, getUserAgentFromGrpcMetadata(ctx), "")
}

func TestGetUserAgentFromGRPCMetadataWitMultipleUserAgents(t *testing.T) {
	// no user-agent key
	md := metadata.Pairs("user-agent", "unit_test")
	md.Append("user-agent", "hello")

	ctx := metadata.NewIncomingContext(context.Background(), md)
	assert.Equal(t, getUserAgentFromGrpcMetadata(ctx), "unit_test,hello")
}
