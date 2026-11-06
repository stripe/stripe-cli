package rpcservice

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"
)

func TestAllowRequestWithToken(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	_, err = client.Version(ctx, &rpc.VersionRequest{})
	assert.Equal(t, nil, err)
}

func TestRejectRequestWithInvalidToken(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	tests := []struct {
		name   string
		values []string
	}{
		// Clients written before the token existed send "1"
		{"legacy value", []string{"1"}},
		{"wrong token", []string{"not-the-token"}},
		{"empty", []string{""}},
		{"token prefix", []string{testToken[:len(testToken)-1]}},
		{"token with suffix", []string{testToken + "x"}},
		{"token sent twice", []string{testToken, testToken}},
		{"wrong token before token", []string{"not-the-token", testToken}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := metadata.MD{}
			md.Append(requiredHeader, tt.values...)
			ctx := metadata.NewOutgoingContext(context.Background(), md)

			_, err := client.Version(ctx, &rpc.VersionRequest{})
			assert.Equal(t, codes.Unauthenticated, status.Code(err))
			assert.Contains(t, status.Convert(err).Message(), "Invalid sec-x-stripe-cli token")
		})
	}
}

func TestRejectStreamWithInvalidToken(t *testing.T) {
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(requiredHeader, "1"))

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	stream, err := client.LogsTail(ctx, &rpc.LogsTailRequest{})
	require.NoError(t, err)

	_, err = stream.Recv()
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "Invalid sec-x-stripe-cli token")
}

func TestRejectRequestIfServerHasNoToken(t *testing.T) {
	srv := &RPCService{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(requiredHeader, ""))

	assert.Equal(t, codes.Unauthenticated, status.Code(srv.authorize(ctx)))
}

func TestNewIssuesUniqueToken(t *testing.T) {
	cfg := &Config{UserCfg: &config.Config{}}
	first := New(cfg, nil)
	second := New(cfg, nil)

	// rand.Text returns at least 128 bits of randomness as 26 or more base32 characters
	assert.GreaterOrEqual(t, len(first.token), 26)
	assert.NotEqual(t, first.token, second.token)
}

func TestRejectRequestIfHeaderAbsent(t *testing.T) {
	md := metadata.New(map[string]string{"foo-bar": "1"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	_, err = client.Version(ctx, &rpc.VersionRequest{})
	expected := status.Errorf(codes.Unauthenticated, "%s", fmt.Sprintf("%s header is not supplied", requiredHeader))

	assert.Equal(t, expected.Error(), err.Error())
}

func TestRejectRequestIfMetadataEmpty(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	_, err = client.Version(ctx, &rpc.VersionRequest{})
	expected := status.Errorf(codes.Unauthenticated, "%s", fmt.Sprintf("%s header is not supplied", requiredHeader))

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
	assert.Equal(t, "acct_xxx", eventMetadata.Merchant)
	assert.Equal(t, "method", eventMetadata.CommandPath)
	assert.Equal(t, "unit_test", eventMetadata.UserAgent)
	assert.Equal(t, stripe.GetTelemetryClient(newCtx), telemetryClient)
}

func TestGetUserAgentFromGRPCMetadata(t *testing.T) {
	// No grpc metadata
	assert.Equal(t, "", getUserAgentFromGrpcMetadata(context.Background()))
}

func TestGetUserAgentFromGRPCMetadataWithNoUserAgent(t *testing.T) {
	// no user-agent key
	md := metadata.Pairs("hello", "world")

	ctx := metadata.NewIncomingContext(context.Background(), md)
	assert.Equal(t, "", getUserAgentFromGrpcMetadata(ctx))
}

func TestGetUserAgentFromGRPCMetadataWitMultipleUserAgents(t *testing.T) {
	// no user-agent key
	md := metadata.Pairs("user-agent", "unit_test")
	md.Append("user-agent", "hello")

	ctx := metadata.NewIncomingContext(context.Background(), md)
	assert.Equal(t, "unit_test,hello", getUserAgentFromGrpcMetadata(ctx))
}
