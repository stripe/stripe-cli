package login

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login/keys"
)

// DefaultLoginTimeout bounds how long LoginAndWait waits for the user to complete
// authentication in the browser when the caller doesn't specify a timeout.
const DefaultLoginTimeout = 10 * time.Minute

// LoginResult describes the account that became active after a LoginAndWait call.
type LoginResult struct {
	AccountID   string
	AccountName string
	// Livemode is always false for the legacy RAK flow, which doesn't have separate
	// live/sandbox contexts the way OAuth does.
	Livemode bool
}

// LoginAndWait starts a Stripe CLI login, the same way `stripe login` does when run
// interactively: it requests a browser URL, opens the browser automatically when possible, and
// waits for the user to complete authentication -- using the OAuth device-code flow or the
// legacy RAK flow depending on what the server returns. Unlike `stripe login`, it always starts
// a new login attempt regardless of any credential already stored, so it works even if that
// credential is expired or revoked.
//
// If timeout elapses (or ctx is canceled) before the user completes authentication, it returns a
// nil result and a nil error, the same way a nil result from SwitchContext means the user
// canceled -- the caller can call LoginAndWait again to keep waiting.
func LoginAndWait(ctx context.Context, dashboardBaseURL, accessBaseURL string, cfg *config.Config, timeout time.Duration) (*LoginResult, error) {
	links, useOAuth, err := GetLinks(ctx, dashboardBaseURL, cfg.Profile.DeviceName, cfg.GetMachineUUID())
	if err != nil {
		return nil, err
	}

	// Clear all stale credentials before saving new ones, so this succeeds even if a
	// previously stored credential is expired or revoked.
	_ = cfg.RemoveAuthFields(cfg.Profile.ProfileName)

	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if useOAuth {
		return loginAndWaitOAuth(ctx, pollCtx, accessBaseURL, cfg)
	}
	return loginAndWaitRAK(pollCtx, links, cfg)
}

func loginAndWaitOAuth(ctx, pollCtx context.Context, accessBaseURL string, cfg *config.Config) (*LoginResult, error) {
	authResp, clientID, err := RequestDeviceCodeForAccessBase(ctx, accessBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}
	_ = OpenBrowserIfPossible(authResp.VerificationURI)

	interval := max(time.Duration(authResp.Interval)*time.Second, 5*time.Second)
	result, err := PollAndSaveDeviceCredentials(pollCtx, accessBaseURL, clientID, authResp.DeviceCode, interval, cfg)
	if err != nil {
		if pollCtx.Err() != nil {
			return nil, nil
		}
		return nil, err
	}

	warnIfInsecureStorage()
	return &LoginResult{
		AccountID:   result.ActiveAccountID,
		AccountName: result.ActiveDisplayName,
		Livemode:    result.ActiveLivemode,
	}, nil
}

func loginAndWaitRAK(pollCtx context.Context, links *Links, cfg *config.Config) (*LoginResult, error) {
	_ = OpenBrowserIfPossible(links.BrowserURL)

	response, account, err := keys.PollForKey(pollCtx, links.PollURL, 0, 0)
	if err != nil {
		if pollCtx.Err() != nil {
			return nil, nil
		}
		return nil, err
	}

	configurer := keys.NewRAKConfigurer(cfg, afero.NewOsFs())
	if err := configurer.SaveLoginDetails(response); err != nil {
		return nil, err
	}

	warnIfInsecureStorage()
	return &LoginResult{
		AccountID:   account.ID,
		AccountName: account.Settings.Dashboard.DisplayName,
	}, nil
}
