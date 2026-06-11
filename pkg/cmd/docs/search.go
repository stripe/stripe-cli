package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli-docs-plugin/internal/agent"
	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
	"github.com/stripe/stripe-cli-docs-plugin/internal/pager"
	"github.com/stripe/stripe-cli-docs-plugin/internal/spinner"
	"github.com/stripe/stripe-cli-docs-plugin/internal/tui"
	"github.com/stripe/stripe-cli-docs-plugin/internal/ui"
	"golang.org/x/term"
)

func isStdoutTTY(cmd *cobra.Command) bool {
	f, ok := cmd.OutOrStdout().(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func (r *RootCommand) newSearchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search docs.stripe.com from the terminal",
		Long: `Search docs.stripe.com from the terminal.

Search by keyword or phrase:

  docs search "payment methods"
  docs search "API keys"
  docs search "dispute evidence"`,
		Args: cobra.ArbitraryArgs,
		RunE: r.runSearch,
	}
}

func (r *RootCommand) runSearch(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("search: missing search query argument")
	}

	query := strings.Join(args, " ")

	if r.useTUI(cmd) {
		return r.show(cmd, nil, tui.WithPaletteInput(query))
	}

	styles := ui.DefaultStyles()

	checkmark := styles.SuccessText.Render("✓")
	disabled := agent.Detect() != agent.NotDetected || !isStdoutTTY(cmd)

	var response *docs.SearchResponse
	err := spinner.New().
		WithLabel("Searching Stripe documentation...").
		WithFinalMsg(checkmark + " Searching Stripe documentation...\n").
		WithOutput(cmd.ErrOrStderr()).
		WithDisabled(disabled).
		Run(func() error {
			var searchErr error
			response, searchErr = r.client.Search(cmd.Context(), query)
			if searchErr != nil {
				return fmt.Errorf("search: %w", searchErr)
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	w := pager.New(cmd.OutOrStdout(), !r.noPager)
	defer func() { _ = w.Close() }()
	return r.renderSearch(w, response)
}

func (r *RootCommand) renderSearch(w io.Writer, response *docs.SearchResponse) error {
	styles := ui.DefaultStyles()
	colorOff := r.color() == colorValueOff

	l := list.New().
		Enumerator(list.Bullet).
		ItemStyle(lipgloss.NewStyle().MarginBottom(1))
	for _, hit := range response.Hits {
		route := strings.TrimPrefix(hit.URL, "https://docs.stripe.com")
		cmdStr := "stripe docs " + route
		if colorOff {
			l.Item(hit.Title + "\n" + cmdStr)
		} else {
			title := styles.Title.Render(hit.Title)
			routeStr := styles.Muted.Render(cmdStr)
			l.Item(title + "\n" + routeStr)
		}
	}

	if _, err := fmt.Fprintln(w, l.String()); err != nil {
		return fmt.Errorf("search: writing output: %w", err)
	}
	return nil
}
