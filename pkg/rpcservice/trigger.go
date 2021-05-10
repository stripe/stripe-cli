package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/fixtures"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"
)

var baseURL = stripe.DefaultAPIBaseURL

// Trigger triggers a Stripe event.
func (srv *RPCService) Trigger(ctx context.Context, req *rpc.TriggerRequest) (*rpc.TriggerResponse, error) {
	requestNames, err := fixtures.Trigger(req.Event, req.StripeAccount, baseURL, srv.cfg.UserCfg)
	if err != nil {
		return nil, err
	}

	return &rpc.TriggerResponse{
		Requests: requestNames,
	}, nil
}
