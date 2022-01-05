package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"
)

// IntegrationInsight returns integration insight of a given log.
func (srv *RPCService) IntegrationInsight(ctx context.Context, req *rpc.IntegrationInsightRequest) (*rpc.IntegrationInsightResponse, error) {
	userConfig := srv.cfg.UserCfg
	livemode := false

	key, err := userConfig.Profile.GetAPIKey(livemode)
	if err != nil {
		return nil, err
	}

	insightMessage := requests.IntegrationInsight(ctx, stripe.DefaultAPIBaseURL, stripe.APIVersion, key, &userConfig.Profile, req.Log)

	return &rpc.IntegrationInsightResponse{Message: insightMessage}, nil
}
