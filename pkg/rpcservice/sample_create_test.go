package rpcservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
)

func TestSampleCreateSucceeds(t *testing.T) {
	getSampleConfig = func(sampleName string, forceRefresh bool) (*samples.SampleConfig, error) {
		return &samples.SampleConfig{
			Integrations: []samples.SampleConfigIntegration{
				{
					Name:    "foo",
					Clients: []string{"foo-client-1", "foo-client-2"},
					Servers: []string{"foo-server-1", "foo-server-2"},
				},
			},
		}, nil
	}

	createSample = func(
		ctx context.Context,
		config *config.Config,
		sampleName string,
		selectedConfig *samples.SelectedConfig,
		destination string,
		forceRefresh bool,
		resultChan chan<- samples.CreationResult) {
		defer close(resultChan)
		resultChan <- samples.CreationResult{
			State:       samples.Done,
			Path:        "my path",
			PostInstall: "my post install message",
		}
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SampleCreate(ctx, &rpc.SampleCreateRequest{
		SampleName:      "accept-a-card-payment",
		IntegrationName: "foo",
		Client:          "foo-client-1",
		Server:          "foo-server-1",
		Path:            "my path",
		ForceRefresh:    false,
	})

	expected := rpc.SampleCreateResponse{
		PostInstall: "my post install message",
		Path:        "my path",
	}

	assert.Nil(t, err)
	assert.Equal(t, expected.PostInstall, resp.PostInstall)
	assert.Equal(t, expected.Path, resp.Path)
}

func TestSampleCreateFailsWhenGetSampleConfigFails(t *testing.T) {
	getSampleConfig = func(sampleName string, forceRefresh bool) (*samples.SampleConfig, error) {
		return nil, errors.New("getSampleConfig failed")
	}

	createSample = func(
		ctx context.Context,
		config *config.Config,
		sampleName string,
		selectedConfig *samples.SelectedConfig,
		destination string, forceRefresh bool,
		resultChan chan<- samples.CreationResult) {
		defer close(resultChan)
		resultChan <- samples.CreationResult{
			State:       samples.Done,
			Path:        "my path",
			PostInstall: "my post install message",
		}
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SampleCreate(ctx, &rpc.SampleCreateRequest{
		SampleName:      "accept-a-card-payment",
		IntegrationName: "foo",
		Client:          "foo-client-1",
		Server:          "foo-server-1",
		Path:            "my path",
		ForceRefresh:    false,
	})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestSampleCreateFailsWhenIntegrationDoesntExist(t *testing.T) {
	getSampleConfig = func(sampleName string, forceRefresh bool) (*samples.SampleConfig, error) {
		return &samples.SampleConfig{
			Integrations: []samples.SampleConfigIntegration{
				{
					Name:    "foo",
					Clients: []string{"foo-client-1", "foo-client-2"},
					Servers: []string{"foo-server-1", "foo-server-2"},
				},
			},
		}, nil
	}

	createSample = func(
		ctx context.Context,
		config *config.Config,
		sampleName string,
		selectedConfig *samples.SelectedConfig,
		destination string, forceRefresh bool,
		resultChan chan<- samples.CreationResult) {
		defer close(resultChan)
		resultChan <- samples.CreationResult{
			State:       samples.Done,
			Path:        "my path",
			PostInstall: "my post install message",
		}
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SampleCreate(ctx, &rpc.SampleCreateRequest{
		SampleName:      "accept-a-card-payment",
		IntegrationName: "doesn't exist",
		Client:          "foo-client-1",
		Server:          "foo-server-1",
		Path:            "my path",
		ForceRefresh:    false,
	})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestSampleCreateFailsWhenCreateSampleFails(t *testing.T) {
	getSampleConfig = func(sampleName string, forceRefresh bool) (*samples.SampleConfig, error) {
		return &samples.SampleConfig{
			Integrations: []samples.SampleConfigIntegration{
				{
					Name:    "foo",
					Clients: []string{"foo-client-1", "foo-client-2"},
					Servers: []string{"foo-server-1", "foo-server-2"},
				},
			},
		}, nil
	}

	createSample = func(
		ctx context.Context,
		config *config.Config,
		sampleName string,
		selectedConfig *samples.SelectedConfig,
		destination string,
		forceRefresh bool,
		resultChan chan<- samples.CreationResult) {
		defer close(resultChan)
		resultChan <- samples.CreationResult{
			Err: errors.New("createSample failed"),
		}
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SampleCreate(ctx, &rpc.SampleCreateRequest{
		SampleName:      "accept-a-card-payment",
		IntegrationName: "doesn't exist",
		Client:          "foo-client-1",
		Server:          "foo-server-1",
		Path:            "my path",
		ForceRefresh:    false,
	})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}
