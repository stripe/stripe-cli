package rpcservice

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/rpc"
)

// Make overridable for tests
var fetchRawSampleIntegrations = func(req *rpc.SampleConfigsRequest) ([]samples.SampleConfigIntegration, error) {
	sampleManager, err := samples.NewSampleManager(nil)
	if err != nil {
		return nil, err
	}
	err = sampleManager.Initialize(req.SampleName)
	if err != nil {
		return nil, err
	}
	return sampleManager.SampleConfig.Integrations, nil
}

// SampleConfigs returns a list of available configs for a given Stripe sample.
func (srv *RPCService) SampleConfigs(ctx context.Context, req *rpc.SampleConfigsRequest) (*rpc.SampleConfigsResponse, error) {
	rawSampleIntegrations, err := fetchRawSampleIntegrations(req)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch configs for sample %s: %v", req.SampleName, err)
	}

	formattedSampleIntegrations := make([]*rpc.SampleConfigsResponse_Integration, len(rawSampleIntegrations))
	for i, s := range rawSampleIntegrations {
		formattedSampleIntegrations[i] = &rpc.SampleConfigsResponse_Integration{
			IntegrationName: s.Name,
			Clients:         s.Clients,
			Servers:         s.Servers,
		}
	}

	return &rpc.SampleConfigsResponse{
		Integrations: formattedSampleIntegrations,
	}, nil
}
