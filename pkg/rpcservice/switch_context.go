package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var switchContext = login.SwitchContext

// SwitchContext switches to a different authorized account context, like `stripe switch
// context`. Unlike the CLI command, an account ID is always required since there is no
// interactive picker to fall back on.
func (srv *RPCService) SwitchContext(ctx context.Context, req *rpc.SwitchContextRequest) (*rpc.SwitchContextResponse, error) {
	if req.AccountId == "" {
		return nil, status.Error(codes.InvalidArgument, "Account ID is required")
	}

	result, err := switchContext(ctx, login.DefaultAccessBaseURL, srv.cfg.UserCfg, req.AccountId, req.Live)
	if err != nil {
		return nil, err
	}

	return &rpc.SwitchContextResponse{
		AccountId:   result.Account.ID,
		DisplayName: result.Account.Name,
		Live:        result.Mode == "live",
	}, nil
}
