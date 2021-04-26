package rpcservice

import (
	"context"

	"github.com/spf13/afero"

	gitpkg "github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/rpc"
)

// Make overridable for tests
var fetchRawSampleIntegrations = func(req *rpc.SampleConfigsRequest) []samples.SampleConfigIntegration {
	var sample = samples.Samples{
		Fs:  afero.NewOsFs(),
		Git: gitpkg.Operations{},
	}
	sample.Initialize(req.SampleName)
	return sample.SampleConfig.Integrations
}

// SampleConfigs returns a list of available configs for a given Stripe sample.
func (srv *RPCService) SampleConfigs(ctx context.Context, req *rpc.SampleConfigsRequest) (*rpc.SampleConfigsResponse, error) {
	rawSampleIntegrations := fetchRawSampleIntegrations(req)

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
