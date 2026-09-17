package login

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

// OAuthResolutionState is a closed set of outcomes for OAuth-only callers.
type OAuthResolutionState string

const (
	OAuthResolved           OAuthResolutionState = "resolved"
	OAuthLoginRequired      OAuthResolutionState = "login_required"
	OAuthCompletionRequired OAuthResolutionState = "completion_required"
	OAuthContextRequired    OAuthResolutionState = "context_required"
	OAuthModeMismatch       OAuthResolutionState = "mode_mismatch"
	OAuthContextChanged     OAuthResolutionState = "context_changed"
	OAuthStorageError       OAuthResolutionState = "storage_error"
	OAuthRefreshError       OAuthResolutionState = "refresh_error"
	OAuthRefreshRequired    OAuthResolutionState = "refresh_required"
)

// OAuthResolution contains credentials only when State is OAuthResolved. It is
// for the private host/plugin channel, never user output or telemetry.
type OAuthResolution struct {
	State         OAuthResolutionState
	Token         string
	StripeContext string
	Livemode      bool
}

// OAuthResolutionOptions binds refresh and subsequent reads to the command's
// selected context. AllowRefresh=false makes resolution read-only.
type OAuthResolutionOptions struct {
	Livemode        bool
	ExpectedContext string
	AllowRefresh    bool
}

type oauthLoginRequiredError struct{}

func (*oauthLoginRequiredError) Error() string {
	return "session expired; run 'stripe login' to re-authenticate"
}

func (*oauthLoginRequiredError) ErrorCategory() errorcategory.Category { return errorcategory.Auth }

// ResolveOAuthCredentials bypasses API-key overrides and legacy fallback. The
// lock binds the credential read, optional refresh, and selected context to one
// host operation; browser login is never initiated here.
func ResolveOAuthCredentials(ctx context.Context, accessBaseURL string, cfg *config.Config, options OAuthResolutionOptions) (*OAuthResolution, error) {
	ctx, cancel := context.WithTimeout(ctx, handoffOperationTimeout)
	defer cancel()
	unlock, err := lockOAuthHandoff(ctx)
	if err != nil {
		return nil, &HandoffError{Reason: "continuation_busy_or_unavailable"}
	}
	defer unlock()
	if err := validateHandoffContext(accessBaseURL, cfg); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	state := func(s OAuthResolutionState) (*OAuthResolution, error) { return &OAuthResolution{State: s}, nil }
	snapshot, token, err := handoffSnapshot(cfg)
	if err != nil {
		return state(OAuthStorageError)
	}
	cont, err := readOptionalPendingDeviceAuth()
	if err != nil {
		return state(OAuthStorageError)
	}
	completionRequired, err := ownedOAuthCompletion(cont, accessBaseURL, cfg, snapshot, token)
	if err != nil {
		return state(OAuthStorageError)
	}
	if completionRequired {
		return state(OAuthCompletionRequired)
	}
	if !strings.HasPrefix(token, "oak_") {
		return state(OAuthLoginRequired)
	}
	active, err := config.GetActiveContext()
	if err != nil {
		return state(OAuthStorageError)
	}
	if active == nil || strings.TrimSpace(active.AccountID) == "" {
		return state(OAuthContextRequired)
	}
	if options.ExpectedContext != "" && active.AccountID != options.ExpectedContext {
		return state(OAuthContextChanged)
	}
	if active.Livemode != options.Livemode {
		return state(OAuthModeMismatch)
	}
	expires, err := config.GetUATExpiresAt()
	if err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return state(OAuthStorageError)
	}
	if err != nil || time.Until(expires) < time.Minute {
		if !options.AllowRefresh {
			return state(OAuthRefreshRequired)
		}
		return resolveRenewedOAuth(ctx, accessBaseURL, cfg, cont, active)
	}
	return &OAuthResolution{State: OAuthResolved, Token: token, StripeContext: active.AccountID, Livemode: active.Livemode}, nil
}

// A completion checkpoint owns partially installed credentials until the same
// handoff finishes installing its account context.
func ownedOAuthCompletion(cont *oauthContinuation, accessBaseURL string, cfg *config.Config, snapshot oauthHandoffSnapshot, token string) (bool, error) {
	if cont != nil && cont.Version == 1 && cont.State == LoginHandoffPending &&
		cont.AccessBaseURL == accessBaseURL && cont.ProfileName == cfg.Profile.ProfileName {
		if err := validateContinuation(cont, accessBaseURL, cfg); err != nil {
			return false, err
		}
		completion, err := readHandoffCompletion(cont.ID)
		if err != nil {
			return false, err
		}
		if completion != nil && canInstallHandoffCompletion(cont, completion, snapshot, token) {
			return true, nil
		}
	}
	return false, nil
}

func resolveRenewedOAuth(ctx context.Context, accessBaseURL string, cfg *config.Config, cont *oauthContinuation, active *config.ActiveContext) (*OAuthResolution, error) {
	state := func(s OAuthResolutionState) (*OAuthResolution, error) { return &OAuthResolution{State: s}, nil }
	profile := cfg.Profile
	profile.OAuthAccessBaseURL = accessBaseURL
	if err := refreshOAuthTokenLocked(ctx, &profile); err != nil {
		var loginRequired *oauthLoginRequiredError
		if errors.As(err, &loginRequired) {
			if err := clearOAuthCredentialsForProfile(&profile); err != nil {
				return state(OAuthStorageError)
			}
			// Completed handoffs are no longer useful after terminal renewal
			// failure. Preserve pending or incompatible continuations.
			if cont != nil && cont.Version == 1 && cont.State == LoginHandoffAuthenticated &&
				cont.ProfileName == cfg.Profile.ProfileName && cont.AccessBaseURL == accessBaseURL {
				if err := forgetPendingLoginLocked(); err != nil {
					return state(OAuthStorageError)
				}
			}
			return state(OAuthLoginRequired)
		}
		if category, ok := errorcategory.Get(err); ok && category == errorcategory.Filesystem {
			return state(OAuthStorageError)
		}
		return state(OAuthRefreshError)
	}
	token, err := cfg.Profile.GetUAT()
	if err != nil {
		return state(OAuthStorageError)
	}
	current, err := config.GetActiveContext()
	if err != nil {
		return state(OAuthStorageError)
	}
	if current == nil || *current != *active {
		return state(OAuthContextChanged)
	}
	expires, err := config.GetUATExpiresAt()
	if err != nil || !strings.HasPrefix(token, "oak_") || !time.Now().Before(expires) {
		return state(OAuthRefreshError)
	}
	return &OAuthResolution{State: OAuthResolved, Token: token, StripeContext: active.AccountID, Livemode: active.Livemode}, nil
}
