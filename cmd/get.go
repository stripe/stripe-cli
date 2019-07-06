package cmd

import (
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/requests"
	"github.com/stripe/stripe-cli/validators"
)

type getCmd struct {
	reqs requests.Base
}

func newGetCmd() *getCmd {
	gc := &getCmd{}

	gc.reqs.Method = "GET"
	gc.reqs.Profile = Profile
	gc.reqs.Cmd = &cobra.Command{
		Use:   "get",
		Args:  validators.ExactArgs(1),
		Short: "Make GET requests to the Stripe API using your test mode key.",
		Long: `Make GET requests to the Stripe API using your test mode key.

You can only get data in test mode, the get command does not work for live mode.
The command also supports common API features like pagination and limits.

For a full list of supported paths, see the API reference: https://stripe.com/docs/api

GET a charge:
$ stripe get /charges/ch_1EGYgUByst5pquEtjb0EkYha

GET 50 charges:
$ stripe get --limit 50 /charges`,

		RunE: gc.reqs.RunRequestsCmd,
	}

	gc.reqs.InitFlags()

	return gc
}
