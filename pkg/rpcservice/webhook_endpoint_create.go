package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"
)

// WebhookEndpointCreate create a new webhook endpoint
func (srv *RPCService) WebhookEndpointCreate(ctx context.Context, req *rpc.WebhookEndpointCreateRequest) (*rpc.WebhookEndpointCreateResponse, error) {
	userConfig := srv.cfg.UserCfg
	livemode := false

	key, err := userConfig.Profile.GetAPIKey(livemode)
	if err != nil {
		return nil, err
	}

	err = requests.WebhookEndpointCreate(
		ctx,
		stripe.DefaultAPIBaseURL,
		stripe.APIVersion,
		key,
		req.Url,
		req.Description,
		req.Connect,
		&userConfig.Profile,
	)
	if err != nil {
		return nil, err
	}

	return &rpc.WebhookEndpointCreateResponse{}, nil
}
