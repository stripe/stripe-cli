package rpcservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/login/acct"
	"github.com/stripe/stripe-cli/pkg/login/keys"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestLoginStatusSucceeds(t *testing.T) {
	links = &login.Links{
		BrowserURL:       "foo",
		PollURL:          "bar",
		VerificationCode: "baz",
	}

	pollForKey = func(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int) (*keys.PollAPIKeyResponse, *acct.Account, error) {
		return &keys.PollAPIKeyResponse{}, &acct.Account{
			ID: "acct_12345",
			Settings: acct.Settings{
				Dashboard: acct.Dashboard{
					DisplayName: "my display name",
				},
			},
		}, nil
	}

	configureProfile = func(config *config.Config, response *keys.PollAPIKeyResponse) error {
		return nil
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.LoginStatus(ctx, &rpc.LoginStatusRequest{})

	expected := &rpc.LoginStatusResponse{
		AccountId:   "acct_12345",
		DisplayName: "my display name",
	}

	assert.Nil(t, err)
	assert.Equal(t, expected.AccountId, resp.AccountId)
	assert.Equal(t, expected.DisplayName, resp.DisplayName)
}

func TestLoginStatusFailsWhenLinksEmpty(t *testing.T) {
	links = &login.Links{}

	pollForKey = func(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int) (*keys.PollAPIKeyResponse, *acct.Account, error) {
		return &keys.PollAPIKeyResponse{}, &acct.Account{
			ID: "acct_12345",
			Settings: acct.Settings{
				Dashboard: acct.Dashboard{
					DisplayName: "my display name",
				},
			},
		}, nil
	}

	configureProfile = func(config *config.Config, response *keys.PollAPIKeyResponse) error {
		return nil
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.LoginStatus(ctx, &rpc.LoginStatusRequest{})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestLoginStatusFailsWhenLinksNil(t *testing.T) {
	links = nil

	pollForKey = func(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int) (*keys.PollAPIKeyResponse, *acct.Account, error) {
		return &keys.PollAPIKeyResponse{}, &acct.Account{
			ID: "acct_12345",
			Settings: acct.Settings{
				Dashboard: acct.Dashboard{
					DisplayName: "my display name",
				},
			},
		}, nil
	}

	configureProfile = func(config *config.Config, response *keys.PollAPIKeyResponse) error {
		return nil
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.LoginStatus(ctx, &rpc.LoginStatusRequest{})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestLoginStatusFailsWhenPollFails(t *testing.T) {
	links = nil

	pollForKey = func(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int) (*keys.PollAPIKeyResponse, *acct.Account, error) {
		return nil, nil, errors.New("pollForKey failed")
	}

	configureProfile = func(config *config.Config, response *keys.PollAPIKeyResponse) error {
		return nil
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.LoginStatus(ctx, &rpc.LoginStatusRequest{})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestLoginStatusFailsWhenConfigureProfileFails(t *testing.T) {
	links = nil

	pollForKey = func(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int) (*keys.PollAPIKeyResponse, *acct.Account, error) {
		return &keys.PollAPIKeyResponse{}, &acct.Account{
			ID: "acct_12345",
			Settings: acct.Settings{
				Dashboard: acct.Dashboard{
					DisplayName: "my display name",
				},
			},
		}, nil
	}

	configureProfile = func(config *config.Config, response *keys.PollAPIKeyResponse) error {
		return errors.New("configureProfile failed")
	}

	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.LoginStatus(ctx, &rpc.LoginStatusRequest{})

	assert.NotNil(t, err)
	assert.Nil(t, resp)
}
