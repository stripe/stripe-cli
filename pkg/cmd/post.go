package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

func newPostCmd(isPreview bool) *requests.Base {
	reqs := &requests.Base{
		Method:           http.MethodPost,
		Profile:          &Config.Profile,
		IsPreviewCommand: isPreview,
	}

	preview := ""
	verb := "POST"
	if isPreview {
		preview = "preview "
		verb = "preview POST"
	}

	previewNote := ""
	if isPreview {
		previewNote = "\nThe preview Stripe-Version header is set automatically on all requests.\n"
	}

	reqs.Cmd = &cobra.Command{
		Use:   "post <path>",
		Args:  validators.ExactArgs(1),
		Short: "Make a " + verb + " request to the Stripe API",
		Long: `Make ` + verb + ` requests to the Stripe API using your test mode key.

The post command supports API features like idempotency keys and expand flags.
` + previewNote + `
For a full list of supported paths, see the API reference:
https://stripe.com/docs/api
`,
		Example: `stripe ` + preview + `post /payment_intents \
    -d amount=2000 \
    -d currency=usd \
    -d "payment_method_types[]=card"`,
		RunE: reqs.RunRequestsCmd,
	}

	reqs.InitFlags()

	return reqs
}
