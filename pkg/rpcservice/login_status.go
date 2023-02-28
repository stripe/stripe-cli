package rpcservice

import (
	"context"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login/keys"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var pollForKey = keys.PollForKey
var configureProfile = saveLoginDetails

// LoginStatus returns when login is successful, or returns an error if failure or timeout.
func (srv *RPCService) LoginStatus(ctx context.Context, req *rpc.LoginStatusRequest) (*rpc.LoginStatusResponse, error) {
	if links == nil || links.PollURL == "" {
		return nil, status.Error(codes.FailedPrecondition, "There is no login in progress.")
	}

	response, account, err := pollForKey(ctx, links.PollURL, 0, 0)
	if err != nil {
		return nil, err
	}

	err = configureProfile(srv.cfg.UserCfg, response)
	if err != nil {
		return nil, err
	}

	displayName := account.Settings.Dashboard.DisplayName
	accountID := account.ID

	return &rpc.LoginStatusResponse{
		DisplayName: displayName,
		AccountId:   accountID,
	}, nil
}

func saveLoginDetails(config *config.Config, response *keys.PollAPIKeyResponse) error {
	configurer := keys.NewRAKConfigurer(config, afero.NewOsFs())
	return configurer.SaveLoginDetails(response)
}
