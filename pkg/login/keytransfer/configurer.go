package keytransfer

import (
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// Configurer saves login details into the filesystem after the user has gone through the login flow
type Configurer struct {
	cfg *config.Config
	fs  afero.Fs
}

// NewConfigurer returns a new configurer
func NewConfigurer(cfg *config.Config, fs afero.Fs) *Configurer {
	return &Configurer{
		cfg: cfg,
		fs:  fs,
	}
}

// SaveLoginDetails function sets config for this profile.
func (c *Configurer) SaveLoginDetails(response *PollAPIKeyResponse) error {
	validateErr := validators.APIKey(response.TestModeAPIKey)
	if validateErr != nil {
		return validateErr
	}

	c.cfg.Profile.LiveModeAPIKey = response.LiveModeAPIKey
	c.cfg.Profile.LiveModePublishableKey = response.LiveModePublishableKey
	c.cfg.Profile.TestModeAPIKey = response.TestModeAPIKey
	c.cfg.Profile.TestModePublishableKey = response.TestModePublishableKey
	c.cfg.Profile.DisplayName = response.AccountDisplayName
	c.cfg.Profile.AccountID = response.AccountID

	profileErr := c.cfg.Profile.CreateProfile()
	if profileErr != nil {
		return profileErr
	}

	return nil
}
