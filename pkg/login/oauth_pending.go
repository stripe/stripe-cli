package login

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// oauthContinuation holds the data needed to poll for an OAuth device token.
// It is written to disk by InitiateLogin and read by PollPendingDeviceAuth.
type oauthContinuation struct {
	DeviceCode    string `json:"device_code"`
	Interval      int    `json:"interval"`
	ExpiresIn     int    `json:"expires_in"`
	AccessBaseURL string `json:"access_base"`
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
	return os.WriteFile(path, data, 0600)
}

func loadPendingDeviceAuth() (*oauthContinuation, error) {
	data, err := os.ReadFile(pendingDeviceAuthPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errorcategory.New(errorcategory.UserInput, "no pending OAuth login found; run 'stripe login --non-interactive' first")
		}
		return nil, err
	}
	var cont oauthContinuation
	if err := json.Unmarshal(data, &cont); err != nil {
		return nil, fmt.Errorf("invalid pending OAuth state: %w", err)
	}
	return &cont, nil
}

func clearPendingDeviceAuth() {
	_ = os.Remove(pendingDeviceAuthPath())
}
