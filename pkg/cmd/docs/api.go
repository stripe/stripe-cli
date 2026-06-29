package docs

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/docs/ui"
)

// errAPINotFound indicates a failed API reference lookup.
type errAPINotFound struct {
	query string
	cause error
}

func (e *errAPINotFound) Error() string {
	return fmt.Sprintf("API reference not found for %q: %v", e.query, e.cause)
}

func (e *errAPINotFound) Unwrap() error {
	return e.cause
}

func (r *RootCommand) newAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api <method|resource|event>",
		Short: "Look up Stripe API reference documentation",
		Long: `Look up Stripe API reference documentation by resource name, HTTP method and path, or event type.

Look up by resource name:

  stripe docs api product
  stripe docs api customer

Look up by HTTP method and path:

  stripe docs api GET /v1/products
  stripe docs api POST /v1/products/{id}

Look up by event type:

  stripe docs api charge.succeeded
  stripe docs api payment_intent.created`,
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
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s Unable to locate API Reference documentation for %q\n", s.Error.Render("✗"), q)
		if r.logger != nil {
			r.logger.Debug("api reference lookup failed", "query", q, "error", err)
		}
		cmd.SilenceErrors = true
		return &errAPINotFound{query: q, cause: err}
	}

	return r.show(cmd, &page)
}
