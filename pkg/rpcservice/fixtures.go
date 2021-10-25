package rpcservice

import (
	"context"
	"os"

	"github.com/stripe/stripe-cli/pkg/fixtures"
	"github.com/stripe/stripe-cli/rpc"
)

// Fixture returns the default fixture of given event in string format
func (srv *RPCService) Fixture(ctx context.Context, req *rpc.FixtureRequest) (*rpc.FixtureResponse, error) {
	fixtureFilename := fixtures.Events[req.Event]
	data, err := os.ReadFile(fixtureFilename)

	defaultFixture := ""
	if err == nil {
		defaultFixture = string(data)
	}

	return &rpc.FixtureResponse{Fixture: defaultFixture}, nil
}
