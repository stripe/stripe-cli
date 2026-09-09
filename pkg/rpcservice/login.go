package rpcservice

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"
)

var links *login.Links
var getLinks = login.GetLinks

// Login returns a URL and pairing code to complete the login for the Stripe CLI
func (srv *RPCService) Login(ctx context.Context, req *rpc.LoginRequest) (*rpc.LoginResponse, error) {
	var err error

	var useOAuth bool
	links, useOAuth, err = getLinks(ctx, stripe.DefaultDashboardBaseURL, srv.cfg.UserCfg.Profile.DeviceName, srv.cfg.UserCfg.GetMachineUUID())
	if err != nil {
		return nil, err
	}
	if useOAuth {
		return nil, errorcategory.Errorf(errorcategory.Auth, "OAuth login required; use 'stripe login' in a terminal to complete browser authorization")
	}

	return &rpc.LoginResponse{
		Url:         links.BrowserURL,
		PairingCode: links.VerificationCode,
	}, nil
}
