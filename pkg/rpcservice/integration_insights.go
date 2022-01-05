package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IntegrationInsight returns integration insight of a given log.
func (srv *RPCService) IntegrationInsight(ctx context.Context, req *rpc.IntegrationInsightRequest) (*rpc.IntegrationInsightResponse, error) {
	userConfig := srv.cfg.UserCfg
	livemode := false

	key, err := userConfig.Profile.GetAPIKey(livemode)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	insightMessage, err := requests.IntegrationInsight(ctx, stripe.DefaultAPIBaseURL, stripe.APIVersion, key, &userConfig.Profile, req.Log)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	return &rpc.IntegrationInsightResponse{Message: insightMessage}, nil
}
