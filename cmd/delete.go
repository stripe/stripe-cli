package cmd

import (
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/requests"
	"github.com/stripe/stripe-cli/validators"
)

type deleteCmd struct {
	reqs requests.Base
}

func newDeleteCmd() *deleteCmd {
	gc := &deleteCmd{}

	gc.reqs.Method = "DELETE"
	gc.reqs.Profile = Profile
	gc.reqs.Cmd = &cobra.Command{
		Use:   "delete",
		Args:  validators.ExactArgs(1),
		Short: "Make DELETE requests to the Stripe API using your test mode key.",
		Long: `Make DELETE requests to the Stripe API using your test mode key.

You can only delete data in test mode, the delete command does not work for
live mode.

For a full list of supported paths, see the API reference: https://stripe.com/docs/api

DELETE a charge:
$ stripe delete /charges/ch_1EGYgUByst5pquEtjb0EkYha`,
		RunE: gc.reqs.RunRequestsCmd,
	}

	gc.reqs.InitFlags()

	return gc
}
