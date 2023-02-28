package login

import (
	"context"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login/keys"
)

// Login is the main entrypoint for logging in to the CLI.
func Login(ctx context.Context, baseURL string, config *config.Config) error {
	links, err := GetLinks(ctx, baseURL, config.Profile.DeviceName)
	if err != nil {
		return err
	}

	configurer := keys.NewRAKConfigurer(config, afero.NewOsFs())
	rt := keys.NewRAKTransfer(configurer)
	auth := NewAuthenticator(rt)
	return auth.Login(ctx, links)
}
