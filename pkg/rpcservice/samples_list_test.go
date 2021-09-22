package rpcservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSamplesListReturnsList(t *testing.T) {
	fetchRawSamplesList = func() (map[string]*samples.SampleData, error) {
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

		return list, nil
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SamplesList(ctx, &rpc.SamplesListRequest{})

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

	assert.Equal(t, nil, err)
	assert.Equal(t, len(expected.Samples), len(resp.Samples))
	assert.ElementsMatch(t, expected.Samples, resp.Samples)
}

func TestSamplesListReturnsEmptyList(t *testing.T) {
	fetchRawSamplesList = func() (map[string]*samples.SampleData, error) {
		list := make(map[string]*samples.SampleData)
		return list, nil
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SamplesList(ctx, &rpc.SamplesListRequest{})

	assert.Equal(t, nil, err)
	assert.Equal(t, 0, len(resp.Samples))
}

func TestSamplesListReturnsError(t *testing.T) {
	fetchRawSamplesList = func() (map[string]*samples.SampleData, error) {
		return nil, errors.New("foo")
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	_, err = client.SamplesList(ctx, &rpc.SamplesListRequest{})
	expected := status.Errorf(codes.Internal, "Failed to fetch Stripe samples list: %v", errors.New("foo"))

	assert.Equal(t, expected.Error(), err.Error())
}
