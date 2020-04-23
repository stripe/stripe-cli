package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type postCmd struct {
	reqs requests.Base
}

func newPostCmd() *postCmd {
	gc := &postCmd{}

	gc.reqs.Method = http.MethodPost
	gc.reqs.Profile = &Config.Profile
	gc.reqs.Cmd = &cobra.Command{
		Use:   "post <path>",
		Args:  validators.ExactArgs(1),
		Short: "Make a POST request to the Stripe API",
		Long: `Make POST requests to the Stripe API using your test mode key.

The post command supports API features like idempotency keys and expand flags.

For a full list of supported paths, see the API reference:
https://stripe.com/docs/api
`,
		Example: `stripe post /payment_intents \
    -d amount=2000 \
    -d currency=usd \
    -d "payment_method_types[]=card"`,
		RunE: gc.reqs.RunRequestsCmd,
	}

	gc.reqs.InitFlags()

	return gc
}
