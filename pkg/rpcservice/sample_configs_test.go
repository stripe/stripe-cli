package rpcservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
)

func TestSampleConfigsReturnsListOfIntegrations(t *testing.T) {
	fetchRawSampleIntegrations = func(req *rpc.SampleConfigsRequest) []samples.SampleConfigIntegration {
		return []samples.SampleConfigIntegration{
			{
				Name:    "using-webhooks",
				Clients: []string{"web", "android", "ios"},
				Servers: []string{"java", "node", "node-typescript", "php-slim", "php", "python", "ruby"},
			},
			{
				Name:    "without-webhooks",
				Clients: []string{"web", "android", "ios"},
				Servers: []string{"java", "node", "node-typescript", "php-slim", "php", "python", "ruby"},
			},
			{
				Name:    "decline-on-card-authentication",
				Clients: []string{"web"},
				Servers: []string{"java", "node", "node-typescript", "php-slim", "php", "python", "ruby"},
			},
		}
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SampleConfigs(ctx, &rpc.SampleConfigsRequest{SampleName: "accept-a-card-payment"})
	if err != nil {
		t.Fatalf("SampleConfigs failed: %v", err)
	}

	expected := rpc.SampleConfigsResponse{
		Integrations: []*rpc.SampleConfigsResponse_Integration{
			{
				IntegrationName: "using-webhooks",
				Clients:         []string{"web", "android", "ios"},
				Servers:         []string{"java", "node", "node-typescript", "php-slim", "php", "python", "ruby"},
			},
			{
				IntegrationName: "without-webhooks",
				Clients:         []string{"web", "android", "ios"},
				Servers:         []string{"java", "node", "node-typescript", "php-slim", "php", "python", "ruby"},
			},
			{
				IntegrationName: "decline-on-card-authentication",
				Clients:         []string{"web"},
				Servers:         []string{"java", "node", "node-typescript", "php-slim", "php", "python", "ruby"},
			},
		},
	}

	assert.EqualValues(t, expected.Integrations, resp.Integrations)
}

func TestSampleConfigsReturnsEmpty(t *testing.T) {
	fetchRawSampleIntegrations = func(req *rpc.SampleConfigsRequest) []samples.SampleConfigIntegration {
		return []samples.SampleConfigIntegration{}
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SampleConfigs(ctx, &rpc.SampleConfigsRequest{SampleName: "accept-a-card-payment"})
	if err != nil {
		t.Fatalf("SampleConfigs failed: %v", err)
	}

	expected := rpc.SampleConfigsResponse{
		Integrations: nil,
	}

	assert.EqualValues(t, expected.Integrations, resp.Integrations)
}
