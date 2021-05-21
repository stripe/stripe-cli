package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/fixtures"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var baseURL = stripe.DefaultAPIBaseURL

// Trigger triggers a Stripe event.
func (srv *RPCService) Trigger(ctx context.Context, req *rpc.TriggerRequest) (*rpc.TriggerResponse, error) {
	apiKey, err := srv.cfg.UserCfg.Profile.GetAPIKey(false)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	requestNames, err := fixtures.Trigger(req.Event, req.StripeAccount, baseURL, apiKey)
	if err != nil {
		return nil, err
	}

	return &rpc.TriggerResponse{
		Requests: requestNames,
	}, nil
}
