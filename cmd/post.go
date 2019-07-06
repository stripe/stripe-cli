package cmd

import (
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/requests"
	"github.com/stripe/stripe-cli/validators"
)

type postCmd struct {
	reqs requests.Base
}

func newPostCmd() *postCmd {
	gc := &postCmd{}

	gc.reqs.Method = "POST"
	gc.reqs.Profile = Profile
	gc.reqs.Cmd = &cobra.Command{
		Use:   "post",
		Args:  validators.ExactArgs(1),
		Short: "Make POST requests to the Stripe API using your test mode key.",
		Long: `Make POST requests to the Stripe API using your test mode key.

You can only POST data in test mode, the post command does not work for
live mode. The post command supports API features like idempotency keys and
expand flags.

For a full list of supported paths, see the API reference: https://stripe.com/docs/api

Example:
$ stripe post /payment_intents -d amount=2000 -d currency=usd -d payment_method_types[]=card`,
		RunE: gc.reqs.RunRequestsCmd,
	}


	gc.reqs.InitFlags()

	return gc
}
