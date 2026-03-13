package keys

import (
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// Configurer is an interface for saving login details
type Configurer interface {
	SaveLoginDetails(response *PollAPIKeyResponse) error
}

// RAKConfigurer saves login details into the filesystem after the user has gone through the login flow
type RAKConfigurer struct {
	cfg *config.Config
	fs  afero.Fs
}

// NewRAKConfigurer returns a new RAKConfigurer
func NewRAKConfigurer(cfg *config.Config, fs afero.Fs) *RAKConfigurer {
	return &RAKConfigurer{
		cfg: cfg,
		fs:  fs,
	}
}

// SaveLoginDetails function sets config for this profile.
func (c *RAKConfigurer) SaveLoginDetails(response *PollAPIKeyResponse) error {
	validateErr := validators.APIKey(response.TestModeAPIKey)
	if validateErr != nil {
		return validateErr
	}

	// First, back up the current profile before "default" is overwritten
	c.cfg.CopyProfile(c.cfg.Profile.ProfileName, c.cfg.Profile.GetDisplayName())

	c.cfg.Profile.LiveModeAPIKey = response.LiveModeAPIKey
	c.cfg.Profile.LiveModePublishableKey = response.LiveModePublishableKey
	c.cfg.Profile.TestModeAPIKey = response.TestModeAPIKey
	c.cfg.Profile.TestModePublishableKey = response.TestModePublishableKey
	// TODO: AccountDisplayName appears to be empty for test mode accounts; is there a better default?
	if response.AccountDisplayName != "" {
		c.cfg.Profile.DisplayName = response.AccountDisplayName
	} else {
		c.cfg.Profile.DisplayName = response.AccountID
	}
	c.cfg.Profile.AccountID = response.AccountID

	profileErr := c.cfg.Profile.CreateProfile()
	if profileErr != nil {
		return profileErr
	}

	return nil
}
