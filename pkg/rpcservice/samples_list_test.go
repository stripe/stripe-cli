package rpcservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
)

func TestSamplesListReturnsList(t *testing.T) {
	fetchRawSamplesList = func() map[string]*samples.SampleData {
		list := make(map[string]*samples.SampleData)

		list["accept-a-card-payment"] = &samples.SampleData{
			Name:        "accept-a-card-payment",
			Description: "Learn how to accept a basic card payment",
			URL:         "https://github.com/stripe-samples/accept-a-card-payment",
		}

		list["accept-a-payment"] = &samples.SampleData{
			Name:        "accept-a-payment",
			Description: "Learn how to accept a payment",
			URL:         "https://github.com/stripe-samples/accept-a-payment",
		}

		return list
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SamplesList(ctx, &rpc.SamplesListRequest{})
	if err != nil {
		t.Fatalf("SamplesList failed: %v", err)
	}

	expected := rpc.SamplesListResponse{
		Samples: []*rpc.SamplesListResponse_SampleData{
			{
				Name:        "accept-a-card-payment",
				Description: "Learn how to accept a basic card payment",
				Url:         "https://github.com/stripe-samples/accept-a-card-payment",
			},
			{
				Name:        "accept-a-payment",
				Description: "Learn how to accept a payment",
				Url:         "https://github.com/stripe-samples/accept-a-payment",
			},
		},
	}

	assert.EqualValues(t, expected.Samples, resp.Samples)
}

func TestSamplesListReturnsEmptyList(t *testing.T) {
	fetchRawSamplesList = func() map[string]*samples.SampleData {
		list := make(map[string]*samples.SampleData)
		return list
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SamplesList(ctx, &rpc.SamplesListRequest{})
	if err != nil {
		t.Fatalf("SamplesList failed: %v", err)
	}

	assert.Equal(t, 0, len(resp.Samples))
}
