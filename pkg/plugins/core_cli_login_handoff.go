package plugins

import (
	"context"
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

// ResumableLoginHelper is additive: existing helper implementations and old
// plugins retain the interactive Login contract unchanged.
type ResumableLoginHelper interface {
	BeginOrResumeLogin(context.Context) (*login.LoginHandoff, error)
	CheckLogin(context.Context, string) (*login.LoginHandoff, error)
}

var _ ResumableLoginHelper = (*coreCLIHelper)(nil)

func (s *CoreCLIHelperServer) BeginOrResumeLogin(ctx context.Context, req *proto.BeginOrResumeLoginRequest) (*proto.LoginHandoffResponse, error) {
	h, ok := s.Impl.(ResumableLoginHelper)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "resumable login is unavailable")
	}
	result, err := h.BeginOrResumeLogin(ctx)
	if err != nil {
		return nil, handoffRPCError(err)
	}
	return loginHandoffResponse(result), nil
}

func (s *CoreCLIHelperServer) CheckLogin(ctx context.Context, req *proto.CheckLoginRequest) (*proto.LoginHandoffResponse, error) {
	h, ok := s.Impl.(ResumableLoginHelper)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "resumable login is unavailable")
	}
	result, err := h.CheckLogin(ctx, req.HandoffId)
	if err != nil {
		return nil, handoffRPCError(err)
	}
	return loginHandoffResponse(result), nil
}

func (h *coreCLIHelper) BeginOrResumeLogin(ctx context.Context) (*login.LoginHandoff, error) {
	cfg, ok := h.config.(*config.Config)
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "login configuration unavailable")
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(h.ctx, cancel)
	defer stop()
	defer cancel()
	return login.BeginOrResumeLogin(ctx, h.loginAccessBase(), cfg)
}

func (h *coreCLIHelper) CheckLogin(ctx context.Context, id string) (*login.LoginHandoff, error) {
	cfg, ok := h.config.(*config.Config)
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "login configuration unavailable")
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(h.ctx, cancel)
	defer stop()
	defer cancel()
	return login.CheckLogin(ctx, h.loginAccessBase(), cfg, id)
}

func (h *coreCLIHelper) loginAccessBase() string {
	if h.accessBaseURL != "" {
		return h.accessBaseURL
	}
	return login.DefaultAccessBaseURL
}

func loginHandoffResponse(result *login.LoginHandoff) *proto.LoginHandoffResponse {
	states := map[login.LoginHandoffState]proto.LoginHandoffState{
		login.LoginHandoffPending:          proto.LoginHandoffState_LOGIN_HANDOFF_STATE_PENDING,
		login.LoginHandoffCompleting:       proto.LoginHandoffState_LOGIN_HANDOFF_STATE_COMPLETING,
		login.LoginHandoffAuthenticated:    proto.LoginHandoffState_LOGIN_HANDOFF_STATE_AUTHENTICATED,
		login.LoginHandoffSessionPresent:   proto.LoginHandoffState_LOGIN_HANDOFF_STATE_SESSION_PRESENT,
		login.LoginHandoffExpired:          proto.LoginHandoffState_LOGIN_HANDOFF_STATE_EXPIRED,
		login.LoginHandoffDenied:           proto.LoginHandoffState_LOGIN_HANDOFF_STATE_DENIED,
		login.LoginHandoffSuperseded:       proto.LoginHandoffState_LOGIN_HANDOFF_STATE_SUPERSEDED,
		login.LoginHandoffRecoveryRequired: proto.LoginHandoffState_LOGIN_HANDOFF_STATE_RECOVERY_REQUIRED,
	}
	response := &proto.LoginHandoffResponse{
		State: states[result.State], HandoffId: result.ID, BrowserUrl: result.BrowserURL,
		VerificationCode: result.VerificationCode, CheckAfterSeconds: int32(result.CheckAfterSeconds),
		Reused: result.Reused, AccountId: result.AccountID, Livemode: result.Livemode,
	}
	if !result.ExpiresAt.IsZero() {
		response.ExpiresAt = timestamppb.New(result.ExpiresAt)
	}
	return response
}

func handoffRPCError(err error) error {
	var failure *login.HandoffError
	if !errors.As(err, &failure) {
		return status.Error(codes.FailedPrecondition, "OAuth login could not proceed")
	}
	code := codes.FailedPrecondition
	switch failure.Reason {
	case "invalid_handoff_id", "invalid_profile":
		code = codes.InvalidArgument
	case "authorization_request_failed", "token_request_failed_resume_same_handoff", "account_lookup_failed_resume_same_handoff", "continuation_busy_or_unavailable":
		code = codes.Unavailable
	}
	s, detailErr := status.New(code, "OAuth login could not proceed").WithDetails(&errdetails.ErrorInfo{Reason: failure.Reason, Domain: "stripe.cli.login"})
	if detailErr != nil {
		return status.Error(codes.Internal, "OAuth login status unavailable")
	}
	return s.Err()
}

var _ ResumableLoginHelper = (*CoreCLIHelperClient)(nil)

func (c *CoreCLIHelperClient) BeginOrResumeLogin(ctx context.Context) (*login.LoginHandoff, error) {
	response, err := c.client.BeginOrResumeLogin(ctx, &proto.BeginOrResumeLoginRequest{})
	if err != nil {
		return nil, err
	}
	return loginHandoffFromResponse(response)
}

func (c *CoreCLIHelperClient) CheckLogin(ctx context.Context, id string) (*login.LoginHandoff, error) {
	response, err := c.client.CheckLogin(ctx, &proto.CheckLoginRequest{HandoffId: id})
	if err != nil {
		return nil, err
	}
	return loginHandoffFromResponse(response)
}

func loginHandoffFromResponse(r *proto.LoginHandoffResponse) (*login.LoginHandoff, error) {
	states := map[proto.LoginHandoffState]login.LoginHandoffState{
		proto.LoginHandoffState_LOGIN_HANDOFF_STATE_PENDING:           login.LoginHandoffPending,
		proto.LoginHandoffState_LOGIN_HANDOFF_STATE_COMPLETING:        login.LoginHandoffCompleting,
		proto.LoginHandoffState_LOGIN_HANDOFF_STATE_AUTHENTICATED:     login.LoginHandoffAuthenticated,
		proto.LoginHandoffState_LOGIN_HANDOFF_STATE_SESSION_PRESENT:   login.LoginHandoffSessionPresent,
		proto.LoginHandoffState_LOGIN_HANDOFF_STATE_EXPIRED:           login.LoginHandoffExpired,
		proto.LoginHandoffState_LOGIN_HANDOFF_STATE_DENIED:            login.LoginHandoffDenied,
		proto.LoginHandoffState_LOGIN_HANDOFF_STATE_SUPERSEDED:        login.LoginHandoffSuperseded,
		proto.LoginHandoffState_LOGIN_HANDOFF_STATE_RECOVERY_REQUIRED: login.LoginHandoffRecoveryRequired,
	}
	state, ok := states[r.State]
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "unsupported login handoff state")
	}
	result := &login.LoginHandoff{State: state, ID: r.HandoffId, BrowserURL: r.BrowserUrl, VerificationCode: r.VerificationCode,
		CheckAfterSeconds: int(r.CheckAfterSeconds), Reused: r.Reused, AccountID: r.AccountId, Livemode: r.Livemode}
	if r.ExpiresAt != nil {
		if err := r.ExpiresAt.CheckValid(); err != nil {
			return nil, status.Error(codes.FailedPrecondition, "invalid login handoff expiry")
		}
		result.ExpiresAt = r.ExpiresAt.AsTime()
	}
	return result, nil
}
