package cmd

import (
	"context"
	"net/http"

	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/logout"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type logoutCmd struct {
	cmd        *cobra.Command
	all        bool
	apiBaseURL string
}

func newLogoutCmd() *logoutCmd {
	lc := &logoutCmd{}

	lc.cmd = &cobra.Command{
		Use:   "logout",
		Args:  validators.NoArgs,
		Short: "Logout of your Stripe account",
		Long:  `Logout of your Stripe account from the CLI`,
		RunE:  lc.runLogoutCmd,
	}

	lc.cmd.Flags().BoolVarP(&lc.all, "all", "a", false, "Clear credentials for all projects you are currently logged into.")

	// Hidden configuration flags, useful for dev/debugging
	lc.cmd.Flags().StringVar(&lc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	lc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return lc
}

func (lc *logoutCmd) runLogoutCmd(cmd *cobra.Command, args []string) error {
	experimentalFields := Config.Profile.GetExperimentalFields()
	if experimentalFields.StripeHeaders != "" {
		serverLogout(lc.cmd.Context(), &Config.Profile, lc.apiBaseURL, "")
	}

	if lc.all {
		return logout.All(&Config)
	}

	return logout.Logout(&Config)
}

func serverLogout(ctx context.Context, profile *config.Profile, baseURL string, apiKey string) {
	base := &requests.Base{
		Profile:        profile,
		Method:         http.MethodPost,
		SuppressOutput: true,
		APIBaseURL:     baseURL,
	}

	// Fail open to allow the logout command to complete
	resp, err := base.MakeRequest(ctx, apiKey, "/v1/stripecli/logout", &requests.RequestParameters{}, true)
	if err == nil {
		log.WithFields(log.Fields{
			"prefix": "cmd.Logout.serverLogout",
		}).Debugf("Logout response: resp=%s", resp)
	} else {
		log.WithFields(log.Fields{
			"prefix": "cmd.Logout.serverLogout",
		}).Debugf("Logout error: err=%+v", err)
	}
}
