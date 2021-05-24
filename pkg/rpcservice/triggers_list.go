package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/fixtures"
	"github.com/stripe/stripe-cli/rpc"
)

// TriggersList returns a list of available events for `Trigger`.
func (srv *RPCService) TriggersList(ctx context.Context, req *rpc.TriggersListRequest) (*rpc.TriggersListResponse, error) {
	return &rpc.TriggersListResponse{Events: fixtures.EventNames()}, nil
}
