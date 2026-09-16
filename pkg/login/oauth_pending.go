package login

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// oauthContinuation holds the data needed to poll for an OAuth device token.
// It is written to disk by InitiateLogin and read by PollPendingDeviceAuth.
type oauthContinuation struct {
	DeviceCode           string               `json:"device_code"`
	Interval             int                  `json:"interval"`
	ExpiresIn            int                  `json:"expires_in"`
	AccessBaseURL        string               `json:"access_base"`
	Version              int                  `json:"version,omitempty"`
	ID                   string               `json:"id,omitempty"`
	ProfileName          string               `json:"profile_name,omitempty"`
	CreatedAt            time.Time            `json:"created_at,omitempty"`
	ExpiresAt            time.Time            `json:"expires_at,omitempty"`
	BrowserURL           string               `json:"browser_url,omitempty"`
	VerificationCode     string               `json:"verification_code,omitempty"`
	NextPollAt           time.Time            `json:"next_poll_at,omitempty"`
	State                LoginHandoffState    `json:"state,omitempty"`
	PollInFlight         bool                 `json:"poll_in_flight,omitempty"`
	InitialSnapshot      oauthHandoffSnapshot `json:"initial_snapshot,omitempty"`
	CompletedSnapshot    oauthHandoffSnapshot `json:"completed_snapshot,omitempty"`
	InstallStarted       bool                 `json:"install_started,omitempty"`
	InstalledContextHash string               `json:"installed_context_hash,omitempty"`
	AccountID            string               `json:"account_id,omitempty"`
	Livemode             bool                 `json:"livemode,omitempty"`
}

func pendingDeviceAuthPath() string {
	return filepath.Join(filepath.Dir(config.CredentialsFilePath()), "oauth_pending.json")
}

func savePendingDeviceAuth(cont *oauthContinuation) error {
	path := pendingDeviceAuthPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(cont)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".oauth-pending-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		dir, err := os.Open(filepath.Dir(path))
		if err != nil {
			return err
		}
		defer dir.Close()
		return dir.Sync()
	}
	return nil
}

func readOptionalPendingDeviceAuth() (*oauthContinuation, error) {
	path := pendingDeviceAuthPath()
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
		return nil, errorcategory.New(errorcategory.Filesystem, "OAuth continuation must be a user-only regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, opened) {
		return nil, errorcategory.New(errorcategory.Filesystem, "OAuth continuation changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, 1024*1024))
	if err != nil {
		return nil, err
	}
	var cont oauthContinuation
	if err := json.Unmarshal(data, &cont); err != nil {
		return nil, errorcategory.New(errorcategory.Filesystem, "invalid pending login state")
	}
	return &cont, nil
}

func loadPendingDeviceAuth() (*oauthContinuation, error) {
	cont, err := readOptionalPendingDeviceAuth()
	if err != nil {
		return nil, err
	}
	if cont == nil {
		return nil, errorcategory.New(errorcategory.UserInput, "no pending login found; run 'stripe login --non-interactive' first")
	}
	return cont, nil
}

func removePendingDeviceAuth() error {
	err := os.Remove(pendingDeviceAuthPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
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
