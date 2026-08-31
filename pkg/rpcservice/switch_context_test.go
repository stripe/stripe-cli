package rpcservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestSwitchContextSucceeds(t *testing.T) {
	switchContext = func(ctx context.Context, accessBaseURL string, cfg *config.Config, accountID string, livemode bool) (*login.SwitchResult, error) {
		assert.Equal(t, "acct_12345", accountID)
		assert.True(t, livemode)
		return &login.SwitchResult{
			Account: config.AuthorizedAccount{ID: "acct_12345", Name: "my display name"},
			Mode:    "live",
		}, nil
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SwitchContext(ctx, &rpc.SwitchContextRequest{AccountId: "acct_12345", Live: true})

	assert.Nil(t, err)
	assert.Equal(t, "acct_12345", resp.AccountId)
	assert.Equal(t, "my display name", resp.DisplayName)
	assert.True(t, resp.Live)
}

func TestSwitchContextFailsWhenAccountIDMissing(t *testing.T) {
	switchContext = func(ctx context.Context, accessBaseURL string, cfg *config.Config, accountID string, livemode bool) (*login.SwitchResult, error) {
		t.Fatal("switchContext should not be called when account_id is missing")
		return nil, nil
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SwitchContext(ctx, &rpc.SwitchContextRequest{})

	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestSwitchContextFailsWhenSwitchContextFails(t *testing.T) {
	switchContext = func(ctx context.Context, accessBaseURL string, cfg *config.Config, accountID string, livemode bool) (*login.SwitchResult, error) {
		return nil, errors.New("switchContext failed")
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.SwitchContext(ctx, &rpc.SwitchContextRequest{AccountId: "acct_12345"})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}
