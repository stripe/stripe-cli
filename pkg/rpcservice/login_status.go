package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var pollForKey = login.PollForKey
var configureProfile = login.ConfigureProfile

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
