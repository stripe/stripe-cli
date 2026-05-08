package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

func newDeleteCmd(isPreview bool) *requests.Base {
	reqs := &requests.Base{
		Method:           http.MethodDelete,
		Profile:          &Config.Profile,
		IsPreviewCommand: isPreview,
	}

	preview := ""
	verb := "DELETE"
	if isPreview {
		preview = "preview "
		verb = "preview DELETE"
	}

	previewNote := ""
	if isPreview {
		previewNote = "\nThe preview Stripe-Version header is set automatically on all requests.\n"
	}

	reqs.Cmd = &cobra.Command{
		Use:   "delete <path>",
		Args:  validators.ExactArgs(1),
		Short: "Make a " + verb + " request to the Stripe API",
		Long: `Make ` + verb + ` requests to the Stripe API using your test mode key.
` + previewNote + `
For a full list of supported paths, see the API reference:
https://stripe.com/docs/api
`,
		Example: `stripe ` + preview + `delete /customers/cus_FROPkgsHVRRspg`,
		RunE:    reqs.RunRequestsCmd,
	}

	reqs.InitFlags()

	return reqs
}
