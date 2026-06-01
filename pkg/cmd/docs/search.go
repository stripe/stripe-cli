package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli-docs-plugin/internal/pager"
)

type searchCommand struct {
	cmd  *cobra.Command
	root *RootCommand
}

func (r *RootCommand) newSearchCommand() *searchCommand {
	searchCmd := &searchCommand{
		root: r,
	}

	searchCmd.cmd = &cobra.Command{
		Use:   "search <query>",
		Short: "Search on docs.stripe.com from the terminal",
		Example: `  stripe docs search "Payment methods"
  stripe docs search "API keys"`,
		Args: cobra.ArbitraryArgs,
		RunE: searchCmd.run,
	}
	return searchCmd
}

func (s *searchCommand) run(cmd *cobra.Command, args []string) error {
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
	return s.search(cmd.Context(), w, query)
}

func (s *searchCommand) search(ctx context.Context, w io.Writer, query string) error {
	if s.root.client == nil {
		return fmt.Errorf("search: docs client not initialized")
	}
	if s.root.renderer == nil {
		return fmt.Errorf("search: markdown renderer not initialized")
	}

	response, err := s.root.client.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	out, err := s.root.renderer.RenderSearchResponse(response)
	if err != nil {
		return fmt.Errorf("search: rendering search response: %w", err)
	}

	if _, err = fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("search: writing output: %w", err)
	}
	return nil
}
