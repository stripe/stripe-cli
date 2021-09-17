package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var getSampleConfig = samples.GetSampleConfig
var createSample = samples.Create

// SampleCreate creates a sample at a given path with the selected integration, client language, and server language.
func (srv *RPCService) SampleCreate(ctx context.Context, req *rpc.SampleCreateRequest) (*rpc.SampleCreateResponse, error) {
	selectedConfig, err := getSelectedConfig(req)
	if err != nil {
		return nil, err
	}

	resultChan := make(chan samples.CreationResult)
	go createSample(
		ctx,
		srv.cfg.UserCfg,
		req.SampleName,
		selectedConfig,
		req.Path,
		req.ForceRefresh,
		resultChan,
	)

	for res := range resultChan {
		if res.Err != nil {
			return nil, res.Err
		}
		if res.State == samples.Done {
			return &rpc.SampleCreateResponse{
				Path:        res.Path,
				PostInstall: res.PostInstall,
			}, nil
		}
	}

	return nil, status.Error(codes.Internal, "An unknown error occurred")
}

func getSelectedConfig(req *rpc.SampleCreateRequest) (*samples.SelectedConfig, error) {
	// Validate the selected integration exists
	sampleConfig, err := getSampleConfig(req.SampleName, req.ForceRefresh)
	if err != nil {
		return nil, err
	}

	var selectedIntegration *samples.SampleConfigIntegration
	for i := range sampleConfig.Integrations {
		if sampleConfig.Integrations[i].Name == req.IntegrationName {
			selectedIntegration = &sampleConfig.Integrations[i]
			break
		}
	}
	if selectedIntegration == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to find the integration %s", req.IntegrationName)
	}

	// Set the sample configuration that we will create
	selectedClient := "" // Empty string means there's only one option
	if selectedIntegration.HasMultipleClients() {
		selectedClient = req.Client
	}

	selectedServer := "" // Empty string means there's only one option
	if selectedIntegration.HasMultipleServers() {
		selectedServer = req.Server
	}

	return &samples.SelectedConfig{
		Integration: selectedIntegration,
		Client:      selectedClient,
		Server:      selectedServer,
	}, nil
}
