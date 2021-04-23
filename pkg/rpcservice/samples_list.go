package rpcservice

import (
	"context"

	"github.com/spf13/afero"

	gitpkg "github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/rpc"
)

// Make overridable for tests
var fetchRawSamplesList = func() map[string]*samples.SampleData {
	var sample = samples.Samples{
		Fs:  afero.NewOsFs(),
		Git: gitpkg.Operations{},
	}
	return sample.GetSamples("list")
}

// SamplesList returns a list of available Stripe samples
func (srv *RPCService) SamplesList(ctx context.Context, req *rpc.SamplesListRequest) (*rpc.SamplesListResponse, error) {
	rawSamplesList := fetchRawSamplesList()

	formattedSamplesList := make([]*rpc.SamplesListResponse_SampleData, 0, len(rawSamplesList))
	for _, v := range rawSamplesList {
		formattedSamplesList = append(formattedSamplesList, &rpc.SamplesListResponse_SampleData{
			Name:        v.Name,
			Description: v.Description,
			Url:         v.URL,
		})
	}

	return &rpc.SamplesListResponse{
		Samples: formattedSamplesList,
	}, nil
}
