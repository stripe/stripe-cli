package login

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// oauthContinuation holds the data needed to poll for an OAuth device token, plus enough of
// the original device-authorization response to resume (rather than restart) a login attempt.
// It is written to disk by InitiateLogin/InitiateOAuthLogin and read by PollPendingDeviceAuth/
// PollPendingOAuthLogin.
type oauthContinuation struct {
	DeviceCode      string    `json:"device_code"`
	Interval        int       `json:"interval"`
	ExpiresIn       int       `json:"expires_in"`
	AccessBaseURL   string    `json:"access_base"`
	VerificationURI string    `json:"verification_uri"`
	UserCode        string    `json:"user_code"`
	IssuedAt        time.Time `json:"issued_at"`
}

// deadline returns the absolute time at which this device code expires. It's computed from
// the persisted IssuedAt rather than "now", so it stays correct across a login attempt that's
// resumed (via InitiateOAuthLogin) or polled (via PollPendingOAuthLogin) well after it started.
func (c *oauthContinuation) deadline() time.Time {
	return c.IssuedAt.Add(max(time.Duration(c.ExpiresIn)*time.Second, 10*time.Minute))
}

func pendingDeviceAuthPath() string {
	return filepath.Join(filepath.Dir(config.CredentialsFilePath()), "oauth_pending.json")
}

// savePendingDeviceAuth writes cont via a temp file + rename, so a concurrent read (or a
// concurrent write from another process racing to mint its own device code) never observes a
// partially-written file. It does not otherwise coordinate between concurrent writers: if two
// processes mint at once, the second write wins and the first process's caller ends up holding
// a browser_url/verification_code for a device code no longer on disk. That's a narrow,
// single-user-CLI race that isn't worth solving with cross-process locking.
func savePendingDeviceAuth(cont *oauthContinuation) error {
	path := pendingDeviceAuthPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(cont)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".oauth_pending-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func loadPendingDeviceAuth() (*oauthContinuation, error) {
	data, err := os.ReadFile(pendingDeviceAuthPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errorcategory.New(errorcategory.UserInput, "no pending login found; run 'stripe login --non-interactive' first")
		}
		return nil, err
	}
	var cont oauthContinuation
	if err := json.Unmarshal(data, &cont); err != nil {
		return nil, fmt.Errorf("invalid pending login state: %w", err)
	}
	return &cont, nil
}

func clearPendingDeviceAuth() {
	_ = os.Remove(pendingDeviceAuthPath())
}

func pendingReauthAccountsPath() string {
	return filepath.Join(filepath.Dir(config.CredentialsFilePath()), "pending_reauth_accounts.json")
}

// savePendingReauthAccounts writes a snapshot of authorized accounts to disk,
// so a later, separate `stripe login --complete-reauth` invocation can poll
// for a change from it.
func savePendingReauthAccounts(accounts []config.AuthorizedAccount) error {
	path := pendingReauthAccountsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(accounts)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// loadPendingReauthAccounts reads the snapshot saved by
// savePendingReauthAccounts. Returns nil, nil if none has been saved.
func loadPendingReauthAccounts() ([]config.AuthorizedAccount, error) {
	data, err := os.ReadFile(pendingReauthAccountsPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var accounts []config.AuthorizedAccount
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, fmt.Errorf("invalid pending reauth state: %w", err)
	}
	return accounts, nil
}

// clearPendingReauthAccounts removes the snapshot saved by
// savePendingReauthAccounts, once the reauthorization it was tracking has
// completed.
func clearPendingReauthAccounts() {
	_ = os.Remove(pendingReauthAccountsPath())
}
