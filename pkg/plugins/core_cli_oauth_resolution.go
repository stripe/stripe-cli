package plugins

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

// OAuthCredentialHelper is additive; old credential resolution keeps its API
// key precedence. Only callers of the new method opt into OAuth-only semantics.
type OAuthCredentialHelper interface {
	ResolveOAuthCredentials(context.Context, login.OAuthResolutionOptions) (*login.OAuthResolution, error)
}

var _ OAuthCredentialHelper = (*coreCLIHelper)(nil)
var _ OAuthCredentialHelper = (*CoreCLIHelperClient)(nil)

var oauthResolutionStates = map[login.OAuthResolutionState]proto.OAuthResolutionState{
	login.OAuthResolved:           proto.OAuthResolutionState_OAUTH_RESOLUTION_STATE_RESOLVED,
	login.OAuthLoginRequired:      proto.OAuthResolutionState_OAUTH_RESOLUTION_STATE_LOGIN_REQUIRED,
	login.OAuthCompletionRequired: proto.OAuthResolutionState_OAUTH_RESOLUTION_STATE_COMPLETION_REQUIRED,
	login.OAuthContextRequired:    proto.OAuthResolutionState_OAUTH_RESOLUTION_STATE_CONTEXT_REQUIRED,
	login.OAuthModeMismatch:       proto.OAuthResolutionState_OAUTH_RESOLUTION_STATE_MODE_MISMATCH,
	login.OAuthContextChanged:     proto.OAuthResolutionState_OAUTH_RESOLUTION_STATE_CONTEXT_CHANGED,
	login.OAuthStorageError:       proto.OAuthResolutionState_OAUTH_RESOLUTION_STATE_STORAGE_ERROR,
	login.OAuthRefreshError:       proto.OAuthResolutionState_OAUTH_RESOLUTION_STATE_REFRESH_ERROR,
	login.OAuthRefreshRequired:    proto.OAuthResolutionState_OAUTH_RESOLUTION_STATE_REFRESH_REQUIRED,
}

func (s *CoreCLIHelperServer) ResolveOAuthCredentials(ctx context.Context, req *proto.ResolveOAuthCredentialsRequest) (*proto.ResolveOAuthCredentialsResponse, error) {
	h, ok := s.Impl.(OAuthCredentialHelper)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "OAuth-only resolution is unavailable")
	}
	r, err := h.ResolveOAuthCredentials(ctx, login.OAuthResolutionOptions{Livemode: req.Livemode, ExpectedContext: req.ExpectedContext, AllowRefresh: req.AllowRefresh})
	if err != nil {
		return nil, handoffRPCError(err)
	}
	return &proto.ResolveOAuthCredentialsResponse{State: oauthResolutionStates[r.State], Token: r.Token, StripeContext: r.StripeContext, Livemode: r.Livemode}, nil
}

func (h *coreCLIHelper) ResolveOAuthCredentials(ctx context.Context, options login.OAuthResolutionOptions) (*login.OAuthResolution, error) {
	cfg, ok := h.config.(*config.Config)
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "login configuration unavailable")
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(h.ctx, cancel)
	defer stop()
	defer cancel()
	return login.ResolveOAuthCredentials(ctx, h.loginAccessBase(), cfg, options)
}

func (c *CoreCLIHelperClient) ResolveOAuthCredentials(ctx context.Context, options login.OAuthResolutionOptions) (*login.OAuthResolution, error) {
	r, err := c.client.ResolveOAuthCredentials(ctx, &proto.ResolveOAuthCredentialsRequest{Livemode: options.Livemode, ExpectedContext: options.ExpectedContext, AllowRefresh: options.AllowRefresh})
	if err != nil {
		return nil, err
	}
	for state, wire := range oauthResolutionStates {
		if r.State == wire {
			return &login.OAuthResolution{State: state, Token: r.Token, StripeContext: r.StripeContext, Livemode: r.Livemode}, nil
		}
	}
	return nil, status.Error(codes.FailedPrecondition, "unsupported OAuth resolution state")
}
