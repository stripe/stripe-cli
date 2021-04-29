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
var fetchRawSamplesList = func() (map[string]*samples.SampleData, error) {
	sample := samples.Samples{
		Fs:  afero.NewOsFs(),
		Git: gitpkg.Operations{},
	}
	return sample.GetSamples("list")
}

// SamplesList returns a list of available Stripe samples
func (srv *RPCService) SamplesList(ctx context.Context, req *rpc.SamplesListRequest) (*rpc.SamplesListResponse, error) {
	rawSamplesList, err := fetchRawSamplesList()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch Stripe samples list: %v", err)
	}

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
