package rpcservice

import (
	"context"

	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	gitpkg "github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/rpc"
)

// Make overridable for tests
var fetchRawSampleIntegrations = func(req *rpc.SampleConfigsRequest) ([]samples.SampleConfigIntegration, error) {
	sample := samples.Samples{
		Fs:  afero.NewOsFs(),
		Git: gitpkg.Operations{},
	}
	err := sample.Initialize(req.SampleName)
	if err != nil {
		return nil, err
	}
	return sample.SampleConfig.Integrations, nil
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
