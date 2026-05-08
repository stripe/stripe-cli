package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

func newGetCmd(isPreview bool) *requests.Base {
	reqs := &requests.Base{
		Method:           http.MethodGet,
		Profile:          &Config.Profile,
		IsPreviewCommand: isPreview,
	}

	preview := ""
	verb := "GET"
	if isPreview {
		preview = "preview "
		verb = "preview GET"
	}

	previewNote := ""
	if isPreview {
		previewNote = "\nThe preview Stripe-Version header is set automatically on all requests.\n"
	}

	reqs.Cmd = &cobra.Command{
		Use:   "get <id or path>",
		Args:  validators.ExactArgs(1),
		Short: "Retrieve resources by their ID or make " + verb + " requests",
		Long: `With the get command, you can load API resources by providing just the resource
id. You can also make normal HTTP ` + verb + ` requests to the Stripe API by providing
the API path.
` + previewNote + `
For a full list of supported paths, see the API reference:
https://stripe.com/docs/api
`,
		Example: `stripe ` + preview + `get ch_1EGYgUByst5pquEtjb0EkYha
  stripe ` + preview + `get cus_G6GQwbr1dWXt9O
  stripe ` + preview + `get /v1/charges --limit 50`,
		RunE: reqs.RunRequestsCmd,
	}

	reqs.InitFlags()

	return reqs
}
