package rpcservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
)

func TestLoginReturnsURLAndPairingCode(t *testing.T) {
	getLinks = func(ctx context.Context, baseURL string, deviceName string) (*login.Links, error) {
		return &login.Links{
			BrowserURL:       "foo",
			PollURL:          "bar",
			VerificationCode: "baz",
		}, nil
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.Login(ctx, &rpc.LoginRequest{})

	expected := &rpc.LoginResponse{
		Url:         "foo",
		PairingCode: "baz",
	}

	assert.Nil(t, err)
	assert.Equal(t, expected.Url, resp.Url)
	assert.Equal(t, expected.PairingCode, resp.PairingCode)
}

func TestLoginReturnsFailsWhenGetLinksFails(t *testing.T) {
	getLinks = func(ctx context.Context, baseURL string, deviceName string) (*login.Links, error) {
		return nil, errors.New("Failed to get links")
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.Login(ctx, &rpc.LoginRequest{})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}
