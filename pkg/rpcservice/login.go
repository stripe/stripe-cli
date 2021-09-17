package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"
)

var links *login.Links
var getLinks = login.GetLinks

// Login returns a URL and pairing code to complete the login for the Stripe CLI
func (srv *RPCService) Login(ctx context.Context, req *rpc.LoginRequest) (*rpc.LoginResponse, error) {
	var err error

	links, err = getLinks(ctx, stripe.DefaultDashboardBaseURL, srv.cfg.UserCfg.Profile.DeviceName)
	if err != nil {
		return nil, err
	}

	return &rpc.LoginResponse{
		Url:         links.BrowserURL,
		PairingCode: links.VerificationCode,
	}, nil
}
