package login

import (
	"context"

	"github.com/spf13/afero"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login/configurer"
)

func Login(ctx context.Context, baseURL string, config *config.Config) error {
	links, err := GetLinks(ctx, baseURL, config.Profile.DeviceName)
	if err != nil {
		return err
	}

	configurer := configurer.NewConfigurer(config, afero.NewOsFs())
	auth := NewAuthenticator(configurer)
	return auth.Login(ctx, links)
}