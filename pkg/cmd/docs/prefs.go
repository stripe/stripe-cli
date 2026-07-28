package docs

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	pkgdocs "github.com/stripe/stripe-cli/pkg/docs"
	"github.com/stripe/stripe-cli/pkg/docs/pager"
	"github.com/stripe/stripe-cli/pkg/docs/ui"
)

const docsPrefsConfigKey = "docs_prefs"

func (r *RootCommand) newPrefsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prefs",
		Short: "Manage preferences to customize rendered documentation",
		Long:  `Manage preferences to customize rendered documentation, such as code snippet languages.`,
	}

	cmd.AddCommand(r.newPrefsListCmd())
	cmd.AddCommand(r.newPrefsSetCmd())
	cmd.AddCommand(r.newPrefsUnsetCmd())

	return cmd
}

func (r *RootCommand) newPrefsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List available preferences for customizing rendered documentation",
		Long:    `List available preferences for customizing rendered documentation and their allowed values.`,
		Example: `  stripe docs prefs list`,
		Args:    cobra.NoArgs,
		RunE:    r.runPrefsList,
	}
}

func (r *RootCommand) newPrefsSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "set <id> <value>",
		Short:   "Set a documentation preference",
		Long:    `Set a documentation preference to a specific value.`,
		Example: `  stripe docs prefs set server go`,
		Args:    cobra.ExactArgs(2),
		RunE:    r.runPrefsSet,
	}
}

func (r *RootCommand) newPrefsUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "unset <id>",
		Short:   "Unset a documentation preference, reverting to the default",
		Long:    `Remove a previously set documentation preference, reverting to the default.`,
		Example: `  stripe docs prefs unset server`,
		Args:    cobra.ExactArgs(1),
		RunE:    r.runPrefsUnset,
	}
}

func (r *RootCommand) runPrefsList(cmd *cobra.Command, _ []string) error {
	response, err := r.client.FetchPrefs(cmd.Context())
	if err != nil {
		return fmt.Errorf("prefs: %w", err)
	}

	current := make(map[string]string)
	for _, pref := range response.Prefs {
		if v := r.getDocsPref(pref.ID); v != "" {
			current[pref.ID] = v
		}
	}

	w := pager.New(cmd.OutOrStdout(), !r.noPager)
	defer func() { _ = w.Close() }()
	return r.renderPrefsList(w, response, current)
}

func (r *RootCommand) runPrefsSet(cmd *cobra.Command, args []string) error {
	id, value := args[0], args[1]

	response, err := r.client.FetchPrefs(cmd.Context())
	if err != nil {
		return fmt.Errorf("prefs: %w", err)
	}

	var found *pkgdocs.Pref
	for i := range response.Prefs {
		if response.Prefs[i].ID == id {
			found = &response.Prefs[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("prefs: unknown preference %q", id)
	}

	if len(found.Values) > 0 {
		valid := false
		for _, v := range found.Values {
			if v == value {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("prefs: invalid value %q for %q; allowed: %s", value, id, strings.Join(found.Values, ", "))
		}
	}

	if err := r.writeDocsPref(id, value); err != nil {
		return fmt.Errorf("prefs: writing config: %w", err)
	}

	check := "✓"
	prefID := id
	prefValue := value
	if r.color() != colorValueOff {
		styles := ui.DefaultStyles()
		check = styles.SuccessText.Render(check)
		prefID = styles.Title.Render(id)
		prefValue = styles.SuccessText.Render(value)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s Preference %s set to %s\n", check, prefID, prefValue)
	return nil
}

func (r *RootCommand) runPrefsUnset(cmd *cobra.Command, args []string) error {
	id := args[0]

	if err := r.deleteDocsPref(id); err != nil {
		return fmt.Errorf("prefs: removing config: %w", err)
	}

	check := "✓"
	prefID := id
	if r.color() != colorValueOff {
		styles := ui.DefaultStyles()
		check = styles.SuccessText.Render(check)
		prefID = styles.Title.Render(id)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s Preference %s unset\n", check, prefID)
	return nil
}

func (r *RootCommand) renderPrefsList(w io.Writer, response *pkgdocs.PrefsResponse, current map[string]string) error {
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

		if currentVal, ok := current[pref.ID]; ok {
			valuesLine += fmt.Sprintf(" [current: %s]", currentVal)
		}

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

// loadDocsPrefMap returns all stored docs preferences as a map suitable for passing to the client.
func (r *RootCommand) loadDocsPrefMap() map[string]string {
	if r.cfg == nil {
		return nil
	}
	raw := viper.GetStringMapString(r.cfg.Profile.GetConfigField(docsPrefsConfigKey))
	if len(raw) == 0 {
		return nil
	}
	return raw
}

func (r *RootCommand) getDocsPref(id string) string {
	if r.cfg == nil {
		return ""
	}
	return viper.GetString(r.cfg.Profile.GetConfigField(docsPrefsConfigKey + "." + id))
}

func (r *RootCommand) writeDocsPref(id, value string) error {
	if r.cfg == nil {
		return fmt.Errorf("no configuration available")
	}
	return r.cfg.Profile.WriteConfigField(docsPrefsConfigKey+"."+id, value)
}

func (r *RootCommand) deleteDocsPref(id string) error {
	if r.cfg == nil {
		return fmt.Errorf("no configuration available")
	}
	return r.cfg.Profile.DeleteConfigField(docsPrefsConfigKey + "." + id)
}
