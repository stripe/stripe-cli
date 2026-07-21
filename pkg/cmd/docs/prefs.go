package docs

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
	"github.com/spf13/cobra"

	pkgdocs "github.com/stripe/stripe-cli/pkg/docs"
	"github.com/stripe/stripe-cli/pkg/docs/pager"
	"github.com/stripe/stripe-cli/pkg/docs/ui"
)

func (r *RootCommand) newPrefsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prefs",
		Short: "Manage preferences to customize rendered documentation",
		Long:  `Manage preferences to customize rendered documentation, such as code snippet languages.`,
	}

	cmd.AddCommand(r.newPrefsListCmd())

	return cmd
}

func (r *RootCommand) newPrefsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available preferences for customizing rendered documentation",
		Long: `List available preferences for customizing rendered documentation and their allowed values.

  stripe docs prefs list`,
		Args: cobra.NoArgs,
		RunE: r.runPrefsList,
	}
}

func (r *RootCommand) runPrefsList(cmd *cobra.Command, _ []string) error {
	response, err := r.client.FetchPrefs(cmd.Context())
	if err != nil {
		return fmt.Errorf("prefs: %w", err)
	}

	w := pager.New(cmd.OutOrStdout(), !r.noPager)
	defer func() { _ = w.Close() }()
	return r.renderPrefsList(w, response)
}

func (r *RootCommand) renderPrefsList(w io.Writer, response *pkgdocs.PrefsResponse) error {
	styles := ui.DefaultStyles()
	colorOff := r.color() == colorValueOff

	l := list.New().
		Enumerator(list.Bullet).
		ItemStyle(lipgloss.NewStyle().MarginBottom(1))

	for _, pref := range response.Prefs {
		values := strings.Join(pref.Values, ", ")
		var defaultStr string
		if pref.Default != nil {
			defaultStr = fmt.Sprintf(" (default: %s)", *pref.Default)
		}
		valuesLine := "Values: " + values + defaultStr

		if colorOff {
			l.Item(pref.ID + "\n" + pref.Description + "\n" + valuesLine)
		} else {
			id := styles.Title.Render(pref.ID)
			desc := styles.Muted.Render(pref.Description)
			vals := styles.Muted.Render(valuesLine)
			l.Item(id + "\n" + desc + "\n" + vals)
		}
	}

	if _, err := fmt.Fprintln(w, l.String()); err != nil {
		return fmt.Errorf("prefs: writing output: %w", err)
	}
	return nil
}
