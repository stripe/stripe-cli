package login

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

var handoffNow = time.Now

// LoginHandoffState describes authentication, not whether a plugin command succeeded.
type LoginHandoffState string

const (
	LoginHandoffPending          LoginHandoffState = "pending"
	LoginHandoffCompleting       LoginHandoffState = "completing"
	LoginHandoffAuthenticated    LoginHandoffState = "authenticated"
	LoginHandoffSessionPresent   LoginHandoffState = "session_present"
	LoginHandoffExpired          LoginHandoffState = "expired"
	LoginHandoffDenied           LoginHandoffState = "denied"
	LoginHandoffSuperseded       LoginHandoffState = "superseded"
	LoginHandoffRecoveryRequired LoginHandoffState = "recovery_required"
	handoffOperationTimeout                        = 5 * time.Second
)

// LoginHandoff contains only information safe to return to the caller. In
// particular, the reference is not the OAuth device code or a credential.
type LoginHandoff struct {
	State             LoginHandoffState `json:"state"`
	ID                string            `json:"handoff_id,omitempty"`
	BrowserURL        string            `json:"browser_url,omitempty"`
	VerificationCode  string            `json:"verification_code,omitempty"`
	ExpiresAt         time.Time         `json:"expires_at"`
	CheckAfterSeconds int               `json:"check_after_seconds,omitempty"`
	Reused            bool              `json:"reused"`
	AccountID         string            `json:"account_id,omitempty"`
	Livemode          bool              `json:"livemode"`
}

// HandoffError exposes a bounded reason without echoing endpoint bodies, URLs,
// device codes, or keychain contents through the plugin RPC error channel.
type HandoffError struct{ Reason string }

func (e *HandoffError) Error() string                         { return "OAuth login: " + e.Reason }
func (e *HandoffError) ErrorCategory() errorcategory.Category { return errorcategory.Auth }

type oauthHandoffSnapshot struct {
	TokenHash   string `json:"token_hash"`
	ContextHash string `json:"context_hash"`
}

type oauthHandoffCompletion struct {
	Token      OAuthTokenResponse `json:"token"`
	ReceivedAt time.Time          `json:"received_at"`
}

func handoffCompletionKey(id string) string { return "oauth_login_completion." + id }

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func handoffSnapshot(cfg *config.Config) (oauthHandoffSnapshot, string, error) {
	if config.KeyRing == nil {
		return oauthHandoffSnapshot{}, "", &HandoffError{Reason: "credential_store_unavailable"}
	}
	token, err := cfg.Profile.GetUAT()
	if err != nil {
		return oauthHandoffSnapshot{}, "", &HandoffError{Reason: "credential_store_unreadable"}
	}
	active, err := config.GetActiveContext()
	if err != nil {
		return oauthHandoffSnapshot{}, "", &HandoffError{Reason: "active_context_unreadable"}
	}
	data, _ := json.Marshal(active)
	return oauthHandoffSnapshot{TokenHash: digest([]byte(token)), ContextHash: digest(data)}, token, nil
}

// BeginOrResumeLogin never revokes or replaces a session. A SESSION_PRESENT
// result tells the caller to resolve/validate OAuth using its normal resolver;
// it is not an assertion that an unverified stored token is usable.
func BeginOrResumeLogin(ctx context.Context, accessBaseURL string, cfg *config.Config) (*LoginHandoff, error) {
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
	cont, err := readOptionalPendingDeviceAuth()
	if err != nil {
		return nil, &HandoffError{Reason: "continuation_unreadable"}
	}
	if cont != nil {
		if err := validateContinuation(cont, accessBaseURL, cfg); err != nil {
			return nil, err
		}
		completion, err := readHandoffCompletion(cont.ID)
		if err != nil {
			return nil, err
		}
		if completion != nil && cont.State != LoginHandoffAuthenticated {
			return &LoginHandoff{State: LoginHandoffCompleting, ID: cont.ID, ExpiresAt: cont.ExpiresAt, Reused: true}, nil
		}
		result, err := inspectHandoff(cont, cfg)
		if err != nil {
			return nil, err
		}
		if result.State == LoginHandoffSuperseded {
			_, token, err := handoffSnapshot(cfg)
			if err != nil {
				return nil, err
			}
			if strings.HasPrefix(token, "oak_") {
				return &LoginHandoff{State: LoginHandoffSessionPresent}, nil
			}
		}
		return result, nil
	}
	snapshot, token, err := handoffSnapshot(cfg)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(token, "oak_") {
		return &LoginHandoff{State: LoginHandoffSessionPresent}, nil
	}
	auth, err := RequestDeviceCode(ctx, accessBaseURL, clientIDForAccessBaseURL(accessBaseURL))
	if err != nil {
		return nil, &HandoffError{Reason: "authorization_request_failed"}
	}
	u, parseErr := url.Parse(auth.VerificationURI)
	if auth.DeviceCode == "" || auth.UserCode == "" || auth.ExpiresIn <= 0 || auth.ExpiresIn > 86400 ||
		auth.Interval < 0 || auth.Interval > auth.ExpiresIn || parseErr != nil || u.User != nil ||
		validateBrowserURL(auth.VerificationURI, accessBaseURL) != nil {
		return nil, &HandoffError{Reason: "invalid_authorization_response"}
	}
	now := handoffNow().UTC()
	cont = &oauthContinuation{
		Version: 1, ID: uuid.NewString(), ProfileName: cfg.Profile.ProfileName,
		DeviceCode: auth.DeviceCode, Interval: max(auth.Interval, 5), ExpiresIn: auth.ExpiresIn,
		AccessBaseURL: accessBaseURL, CreatedAt: now, ExpiresAt: now.Add(time.Duration(auth.ExpiresIn) * time.Second),
		BrowserURL: auth.VerificationURI, VerificationCode: auth.UserCode,
		State: LoginHandoffPending, InitialSnapshot: snapshot,
	}
	if err := savePendingDeviceAuth(cont); err != nil {
		return nil, &HandoffError{Reason: "continuation_save_failed"}
	}
	return publicHandoff(cont, false), nil
}

// CheckLogin performs at most one token request, respecting the persisted polling
// interval. Termination or a transient error leaves the same handoff available.
func CheckLogin(ctx context.Context, accessBaseURL string, cfg *config.Config, id string) (*LoginHandoff, error) {
	ctx, cancel := context.WithTimeout(ctx, handoffOperationTimeout)
	defer cancel()
	if _, err := uuid.Parse(id); err != nil {
		return nil, &HandoffError{Reason: "invalid_handoff_id"}
	}
	unlock, err := lockOAuthHandoff(ctx)
	if err != nil {
		return nil, &HandoffError{Reason: "continuation_busy_or_unavailable"}
	}
	defer unlock()
	if err := validateHandoffContext(accessBaseURL, cfg); err != nil {
		return nil, err
	}
	cont, err := readOptionalPendingDeviceAuth()
	if err != nil {
		return nil, &HandoffError{Reason: "continuation_unreadable"}
	}
	if cont == nil || cont.ID != id {
		return &LoginHandoff{State: LoginHandoffSuperseded, ID: id}, nil
	}
	if err := validateContinuation(cont, accessBaseURL, cfg); err != nil {
		return nil, err
	}
	completion, err := readHandoffCompletion(id)
	if err != nil {
		return nil, err
	}
	if completion != nil && cont.State != LoginHandoffAuthenticated {
		return installHandoffCompletion(ctx, cfg, cont, completion)
	}
	result, err := inspectHandoff(cont, cfg)
	if err == nil && result.State == LoginHandoffAuthenticated && completion != nil {
		// Installation is already durable; retry only journal cleanup. Reinstalling
		// here could reset a context selected after the original completion.
		if cleanupErr := config.KeyRing.Remove(handoffCompletionKey(id)); cleanupErr != nil && !errors.Is(cleanupErr, keyring.ErrKeyNotFound) {
			return nil, &HandoffError{Reason: "completion_checkpoint_cleanup_failed"}
		}
	}
	if err != nil || result.State != LoginHandoffPending || handoffNow().Before(cont.NextPollAt) {
		return result, err
	}
	return pollLoginHandoff(ctx, accessBaseURL, cfg, cont)
}

func pollLoginHandoff(ctx context.Context, accessBaseURL string, cfg *config.Config, cont *oauthContinuation) (*LoginHandoff, error) {
	wasInFlight := cont.PollInFlight
	cont.PollInFlight = true
	cont.NextPollAt = handoffNow().UTC().Add(time.Duration(cont.Interval) * time.Second)
	if err := savePendingDeviceAuth(cont); err != nil {
		return nil, &HandoffError{Reason: "continuation_save_failed"}
	}
	token, err := pollDeviceTokenOnce(ctx, accessBaseURL, clientIDForAccessBaseURL(accessBaseURL), cont.DeviceCode)
	if err != nil {
		var oauthErr *OAuthError
		if !errors.As(err, &oauthErr) {
			return nil, &HandoffError{Reason: "token_request_failed_resume_same_handoff"}
		}
		switch oauthErr.Code {
		case "authorization_pending":
			cont.PollInFlight = false
		case "slow_down":
			cont.PollInFlight = false
			cont.Interval += 5
			cont.NextPollAt = handoffNow().UTC().Add(time.Duration(cont.Interval) * time.Second)
		case "access_denied":
			cont.State = LoginHandoffDenied
		case "expired_token", "invalid_grant":
			cont.State = LoginHandoffExpired
			if wasInFlight {
				cont.State = LoginHandoffRecoveryRequired
			}
		default:
			return nil, &HandoffError{Reason: "token_request_rejected"}
		}
		if err := savePendingDeviceAuth(cont); err != nil {
			return nil, &HandoffError{Reason: "continuation_save_failed"}
		}
		return publicHandoff(cont, true), nil
	}
	if (token.AccessToken == "" || token.RefreshToken == "") || !strings.EqualFold(token.TokenType, "Bearer") || token.ExpiresIn <= 0 || token.ExpiresIn > 86400 {
		return nil, &HandoffError{Reason: "invalid_token_response"}
	}
	completion := &oauthHandoffCompletion{Token: *token, ReceivedAt: handoffNow().UTC()}
	data, _ := json.Marshal(completion)
	if err := config.KeyRing.Set(handoffCompletionKey(cont.ID), data, "Stripe CLI pending OAuth completion"); err != nil {
		return nil, &HandoffError{Reason: "completion_checkpoint_failed"}
	}
	return installHandoffCompletion(ctx, cfg, cont, completion)
}

func validateHandoffContext(accessBaseURL string, cfg *config.Config) error {
	if err := ValidateAccessBaseURL(accessBaseURL); err != nil {
		return err
	}
	if cfg == nil || cfg.Profile.ValidateProfileNameForWrite() != nil {
		return &HandoffError{Reason: "invalid_profile"}
	}
	return nil
}

func validateContinuation(cont *oauthContinuation, accessBaseURL string, cfg *config.Config) error {
	if cont.Version != 1 {
		return &HandoffError{Reason: "unsupported_continuation_use_original_completion_command"}
	}
	if cont.AccessBaseURL != accessBaseURL || cont.ProfileName != cfg.Profile.ProfileName {
		return &HandoffError{Reason: "continuation_context_mismatch"}
	}
	if _, err := uuid.Parse(cont.ID); err != nil || cont.ExpiresAt.IsZero() || cont.CreatedAt.IsZero() ||
		!cont.ExpiresAt.After(cont.CreatedAt) || cont.ExpiresAt.Sub(cont.CreatedAt) > 24*time.Hour || cont.Interval <= 0 || cont.Interval > 86400 {
		return &HandoffError{Reason: "invalid_continuation"}
	}
	switch cont.State {
	case LoginHandoffPending, LoginHandoffAuthenticated, LoginHandoffExpired, LoginHandoffDenied, LoginHandoffRecoveryRequired:
	default:
		return &HandoffError{Reason: "invalid_continuation"}
	}
	if cont.State == LoginHandoffPending && (cont.DeviceCode == "" || validateBrowserURL(cont.BrowserURL, accessBaseURL) != nil) {
		return &HandoffError{Reason: "invalid_continuation"}
	}
	return nil
}

func inspectHandoff(cont *oauthContinuation, cfg *config.Config) (*LoginHandoff, error) {
	snapshot, _, err := handoffSnapshot(cfg)
	if err != nil {
		return nil, err
	}
	if cont.State == LoginHandoffAuthenticated {
		if snapshot != cont.CompletedSnapshot {
			return &LoginHandoff{State: LoginHandoffSuperseded, ID: cont.ID}, nil
		}
	} else if !cont.InstallStarted && snapshot != cont.InitialSnapshot {
		return &LoginHandoff{State: LoginHandoffSuperseded, ID: cont.ID}, nil
	}
	if cont.State == LoginHandoffPending && !handoffNow().Before(cont.ExpiresAt) {
		cont.State = LoginHandoffExpired
		if cont.PollInFlight {
			cont.State = LoginHandoffRecoveryRequired
		}
		if err := savePendingDeviceAuth(cont); err != nil {
			return nil, &HandoffError{Reason: "continuation_save_failed"}
		}
	}
	return publicHandoff(cont, true), nil
}

func publicHandoff(cont *oauthContinuation, reused bool) *LoginHandoff {
	r := &LoginHandoff{State: cont.State, ID: cont.ID, ExpiresAt: cont.ExpiresAt, Reused: reused}
	if cont.State == LoginHandoffPending {
		r.BrowserURL, r.VerificationCode = cont.BrowserURL, cont.VerificationCode
		r.CheckAfterSeconds = max(0, int(cont.NextPollAt.Sub(handoffNow()).Seconds()+1))
	}
	if cont.State == LoginHandoffAuthenticated {
		r.AccountID, r.Livemode = cont.AccountID, cont.Livemode
	}
	return r
}

func readHandoffCompletion(id string) (*oauthHandoffCompletion, error) {
	if config.KeyRing == nil {
		return nil, &HandoffError{Reason: "credential_store_unavailable"}
	}
	data, err := config.KeyRing.Get(handoffCompletionKey(id))
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, &HandoffError{Reason: "completion_checkpoint_unreadable"}
	}
	var completion oauthHandoffCompletion
	if json.Unmarshal(data, &completion) != nil || completion.Token.AccessToken == "" || completion.ReceivedAt.IsZero() {
		return nil, &HandoffError{Reason: "invalid_completion_checkpoint"}
	}
	return &completion, nil
}

func installHandoffCompletion(ctx context.Context, cfg *config.Config, cont *oauthContinuation, completion *oauthHandoffCompletion) (*LoginHandoff, error) {
	snapshot, token, err := handoffSnapshot(cfg)
	if err != nil {
		return nil, err
	}
	canResumeInstallation := cont.InstallStarted && token == completion.Token.AccessToken &&
		(snapshot.ContextHash == cont.InitialSnapshot.ContextHash || snapshot.ContextHash == cont.InstalledContextHash)
	if snapshot != cont.InitialSnapshot && !canResumeInstallation {
		return &LoginHandoff{State: LoginHandoffSuperseded, ID: cont.ID}, nil
	}
	// Finish local installation even if the polling caller has just disconnected.
	// The bounded context also limits the accounts request after token redemption.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), handoffOperationTimeout)
	defer cancel()
	accounts, err := ListAuthorizedAccounts(ctx, cont.AccessBaseURL, completion.Token.AccessToken)
	if err != nil {
		return nil, &HandoffError{Reason: "account_lookup_failed_resume_same_handoff"}
	}
	accountID, livemode := pickActiveContext(accounts)
	if accountID == "" || !authorizedHandoffContext(accounts, accountID, livemode) {
		return nil, &HandoffError{Reason: "no_authorized_context"}
	}
	current, _, err := handoffSnapshot(cfg)
	if err != nil {
		return nil, err
	}
	if current != snapshot {
		return &LoginHandoff{State: LoginHandoffSuperseded, ID: cont.ID}, nil
	}
	activeJSON, _ := json.Marshal(&config.ActiveContext{AccountID: accountID, Livemode: livemode})
	cont.InstallStarted, cont.InstalledContextHash = true, digest(activeJSON)
	if err := savePendingDeviceAuth(cont); err != nil {
		return nil, &HandoffError{Reason: "continuation_save_failed"}
	}
	remaining := int(completion.ReceivedAt.Add(time.Duration(completion.Token.ExpiresIn) * time.Second).Sub(handoffNow()).Seconds())
	if remaining <= 0 {
		return nil, &HandoffError{Reason: "completion_credentials_expired"}
	}
	credentials := completion.Token
	credentials.ExpiresIn = remaining
	if err := saveOAuthCredentials(cfg, &credentials); err != nil {
		return nil, &HandoffError{Reason: "credential_install_failed_resume_same_handoff"}
	}
	if err := populateProfileFromAccounts(cfg, accounts, accountID, livemode); err != nil {
		return nil, &HandoffError{Reason: "context_install_failed_resume_same_handoff"}
	}
	cont.CompletedSnapshot, _, err = handoffSnapshot(cfg)
	if err != nil {
		return nil, err
	}
	cont.State, cont.AccountID, cont.Livemode = LoginHandoffAuthenticated, accountID, livemode
	cont.DeviceCode, cont.BrowserURL, cont.VerificationCode = "", "", ""
	if err := savePendingDeviceAuth(cont); err != nil {
		return nil, &HandoffError{Reason: "completion_state_save_failed"}
	}
	if err := config.KeyRing.Remove(handoffCompletionKey(cont.ID)); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return nil, &HandoffError{Reason: "completion_checkpoint_cleanup_failed"}
	}
	return publicHandoff(cont, true), nil
}

// ForgetPendingLogin is for explicit logout/restart, never for a stopped wait.
func ForgetPendingLogin(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, handoffOperationTimeout)
	defer cancel()
	unlock, err := lockOAuthHandoff(ctx)
	if err != nil {
		return &HandoffError{Reason: "continuation_busy_or_unavailable"}
	}
	defer unlock()
	return forgetPendingLoginLocked()
}

func forgetPendingLoginLocked() error {
	cont, err := readOptionalPendingDeviceAuth()
	if err != nil {
		return &HandoffError{Reason: "continuation_unreadable"}
	}
	if cont != nil && cont.ID != "" && config.KeyRing != nil {
		if err := config.KeyRing.Remove(handoffCompletionKey(cont.ID)); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
			return &HandoffError{Reason: "completion_checkpoint_cleanup_failed"}
		}
	}
	return removePendingDeviceAuth()
}

func handoffStateError(result *LoginHandoff) error {
	return &HandoffError{Reason: fmt.Sprintf("%s; run 'stripe login --new-session' to restart", result.State)}
}

func authorizedHandoffContext(accounts []config.AuthorizedAccount, id string, live bool) bool {
	mode := "test"
	if live {
		mode = "live"
	}
	for _, account := range accounts {
		if account.ID != id {
			continue
		}
		for _, allowed := range account.Modes {
			if allowed == mode {
				return true
			}
		}
	}
	return false
}

// MutateLoginCredentials invalidates outstanding handoffs and serializes a
// bounded credential/context mutation with handoff installation. Callbacks must
// not wait for human input or call another handoff operation.
func MutateLoginCredentials(ctx context.Context, mutate func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, handoffOperationTimeout)
	defer cancel()
	unlock, err := lockOAuthHandoff(ctx)
	if err != nil {
		return &HandoffError{Reason: "continuation_busy_or_unavailable"}
	}
	defer unlock()
	if err := forgetPendingLoginLocked(); err != nil {
		return err
	}
	return mutate(ctx)
}

// HasPendingLogin gives interrupted installation precedence over the CLI's
// ordinary existing-session reauthorization path.
func HasPendingLogin() (bool, error) {
	cont, err := readOptionalPendingDeviceAuth()
	if err != nil {
		return false, err
	}
	return cont != nil && cont.Version == 1 && cont.State == LoginHandoffPending, nil
}
