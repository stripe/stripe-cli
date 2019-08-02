package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type getCmd struct {
	reqs requests.Base
}

func newGetCmd() *getCmd {
	gc := &getCmd{}

	gc.reqs.Method = http.MethodGet
	gc.reqs.Profile = Config.Profile
	gc.reqs.Cmd = &cobra.Command{
		Use:   "get",
		Args:  validators.ExactArgs(1),
		Short: "Make GET requests to the Stripe API using your test mode key.",
		Long: fmt.Sprintf(`%s

Make GET requests to the Stripe API using your test mode key.

The command supports common API features like pagination and limits. Currently,
you can only get data in test mode.

For a full list of supported paths, see the API reference:
https://stripe.com/docs/api

To get a charge:

  $ stripe get /charges/ch_1EGYgUByst5pquEtjb0EkYha

To get 50 charges:

  $ stripe get --limit 50 /charges`,
			ansi.Italic("⚠️  The Stripe CLI is in beta! Have feedback? Let us know, run: 'stripe feedback'. ⚠️"),
		),

		RunE: gc.reqs.RunRequestsCmd,
	}

	gc.reqs.InitFlags()

	return gc
}
