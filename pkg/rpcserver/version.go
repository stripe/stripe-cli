package rpcserver

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/version"
	"github.com/stripe/stripe-cli/rpc"
)

// Version returns the version of the Stripe CLI
func (s *RPCServer) Version(ctx context.Context, req *rpc.VersionRequest) (*rpc.VersionResponse, error) {
	return &rpc.VersionResponse{Version: version.Version}, nil
}
