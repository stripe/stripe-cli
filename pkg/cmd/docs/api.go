package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli-docs-plugin/internal/ui"
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
		Path:     "/_endpoint/api-reference-locator",
		RawQuery: url.Values{"q": {q}}.Encode(),
	}

	page, err := r.client.FetchPage(cmd.Context(), ref)
	if err != nil {
		s := ui.DefaultStyles()
		fmt.Fprintf(cmd.ErrOrStderr(), "%s Unable to locate API Reference documentation for %q\n", s.Error.Render("✗"), q)
		if r.logger != nil {
			r.logger.Debug("api reference lookup failed", "query", q, "error", err)
		}
		cmd.SilenceErrors = true
		return err
	}

	return r.show(cmd, &page)
}
