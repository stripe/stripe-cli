package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"
)

// WebhookEndpointsList returns a list of webhook endpoints.
func (srv *RPCService) WebhookEndpointsList(ctx context.Context, req *rpc.WebhookEndpointsListRequest) (*rpc.WebhookEndpointsListResponse, error) {
	userConfig := srv.cfg.UserCfg
	livemode := false

	key, err := userConfig.Profile.GetAPIKey(livemode)
	if err != nil {
		return nil, err
	}

	endpoints := requests.WebhookEndpointsList(ctx, stripe.DefaultAPIBaseURL, stripe.APIVersion, key, &userConfig.Profile)

	formattedEndpoints := make([]*rpc.WebhookEndpointsListResponse_WebhookEndpointData, 0, len(endpoints.Data))
	for _, v := range endpoints.Data {
		formattedEndpoints = append(formattedEndpoints, &rpc.WebhookEndpointsListResponse_WebhookEndpointData{
			Application:   v.Application,
			EnabledEvents: v.EnabledEvents,
			Url:           v.URL,
			Status:        v.Status,
		})
	}

	return &rpc.WebhookEndpointsListResponse{Endpoints: formattedEndpoints}, nil
}
