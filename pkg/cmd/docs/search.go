package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli-docs-plugin/internal/pager"
)

func (r *RootCommand) newSearchCommand() *cobra.Command {
	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search on docs.stripe.com from the terminal",
		Example: `  stripe docs search "Payment methods"
  stripe docs search "API keys"`,
		Args: cobra.ArbitraryArgs,
		RunE: r.runSearch,
	}
	return searchCmd
}

func (r *RootCommand) runSearch(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("search: missing search query argument")
	}

	query := args[0]
	noPager, err := cmd.Flags().GetBool("no-pager")
	if err != nil {
		return fmt.Errorf("search: getting no-pager flag: %w", err)
	}
	w := pager.New(cmd.OutOrStdout(), !noPager)
	defer func() { _ = w.Close() }()
	return r.search(cmd.Context(), w, query)
}

func (r *RootCommand) search(ctx context.Context, w io.Writer, query string) error {
	if r.client == nil {
		return fmt.Errorf("search: docs client not initialized")
	}

	response, err := r.client.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9D97FF"))
	routeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4A9EFF"))
	for _, hit := range response.Hits {
		route := strings.TrimPrefix(hit.URL, "https://docs.stripe.com")
		lineOutput := fmt.Sprintf("%s\n  %s\n\n", titleStyle.Render(hit.Title), routeStyle.Render("stripe docs "+route))

		if _, err := fmt.Fprint(w, lineOutput); err != nil {
			return fmt.Errorf("search: writing output: %w", err)
		}
	}

	return nil
}
