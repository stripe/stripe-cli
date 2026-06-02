package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func (r *RootCommand) newAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api <method|resource|event>",
		Short: "Look up Stripe API reference documentation",
		Long: `Look up Stripe API reference documentation by HTTP method+path, resource name, or event type.

Examples:
  stripe docs api GET /v1/products
  stripe docs api product
  stripe docs api product.created`,
		Args: cobra.MinimumNArgs(1),
		RunE: r.runAPI,
	}
}

func (r *RootCommand) runAPI(cmd *cobra.Command, args []string) error {
	q := strings.Join(args, " ")

	ref := &url.URL{
		Path:     "/_endpoint/api-reference-locate",
		RawQuery: url.Values{"q": {q}}.Encode(),
	}

	page, err := r.client.FetchPage(cmd.Context(), ref)
	if err != nil {
		return fmt.Errorf("looking up API reference: %w", err)
	}

	return r.show(cmd, &page)
}
