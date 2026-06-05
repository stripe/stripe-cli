package cmd

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli-docs-plugin/internal/agent"
	"github.com/stripe/stripe-cli-docs-plugin/internal/agentskills"
	"github.com/stripe/stripe-cli-docs-plugin/internal/pager"
	"github.com/stripe/stripe-cli-docs-plugin/internal/spinner"
	"github.com/stripe/stripe-cli-docs-plugin/internal/ui"
)

func (r *RootCommand) newSkillsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage agent skills for docs.stripe.com",
	}
	cmd.AddCommand(r.newSkillsListCommand())
	return cmd
}

func (r *RootCommand) newSkillsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available agent skills from docs.stripe.com",
		Args:  cobra.NoArgs,
		RunE:  r.runSkillsList,
	}
}

func (r *RootCommand) runSkillsList(cmd *cobra.Command, _ []string) error {
	styles := ui.DefaultStyles()
	checkmark := styles.SuccessText.Render("✓")
	disabled := agent.Detect() || !isStdoutTTY(cmd)

	var index *agentskills.Index
	err := spinner.New().
		WithLabel("Fetching agent skills...").
		WithFinalMsg(checkmark + " Fetching agent skills...\n").
		WithOutput(cmd.ErrOrStderr()).
		WithDisabled(disabled).
		Run(func() error {
			var fetchErr error
			index, fetchErr = r.client.FetchSkills(cmd.Context())
			if fetchErr != nil {
				return fmt.Errorf("skills list: %w", fetchErr)
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("skills list: %w", err)
	}

	w := pager.New(cmd.OutOrStdout(), !r.noPager)
	defer func() { _ = w.Close() }()
	return r.renderSkillsList(w, index)
}

func (r *RootCommand) renderSkillsList(w io.Writer, index *agentskills.Index) error {
	styles := ui.DefaultStyles()
	colorOff := r.color() == colorValueOff

	var heading, body string
	if colorOff {
		heading = "Install agent skills"
		body = wordWrap("Agent skills are instructions that agents can follow to create more accurate integrations. Stripe has a catalog of skills that instruct agents on best Stripe integration practices. They're available from different marketplaces, as well as hosted at https://docs.stripe.com/.well-known/skills/index.json.")
	} else {
		heading = styles.Title.Render("Install agent skills")
		body = styles.Description.Render(wordWrap("Agent skills are instructions that agents can follow to create more accurate integrations. Stripe has a catalog of skills that instruct agents on best Stripe integration practices. They're available from different marketplaces, as well as hosted at https://docs.stripe.com/.well-known/skills/index.json."))
	}

	if _, err := fmt.Fprintf(w, "%s\n\n%s\n\n", heading, body); err != nil {
		return fmt.Errorf("skills list: writing output: %w", err)
	}

	l := list.New().
		Enumerator(list.Bullet).
		ItemStyle(lipgloss.NewStyle().MarginBottom(1))
	for _, skill := range index.Skills {
		if colorOff {
			l.Item(skill.Name + "\n" + wordWrap(skill.Description))
		} else {
			name := styles.Title.Render(skill.Name)
			desc := styles.Muted.Render(wordWrap(skill.Description))
			l.Item(name + "\n" + desc)
		}
	}

	if _, err := fmt.Fprintln(w, l.String()); err != nil {
		return fmt.Errorf("skills list: writing output: %w", err)
	}
	return nil
}

// wordWrap breaks s into lines of at most 120 characters, splitting on word
// boundaries. Words longer than 120 are placed on their own line unbroken.
func wordWrap(s string) string {
	const width = 120
	var sb strings.Builder
	col := 0
	for i, word := range strings.Fields(s) {
		if i > 0 {
			if col+1+len(word) > width {
				sb.WriteByte('\n')
				col = 0
			} else {
				sb.WriteByte(' ')
				col++
			}
		}
		sb.WriteString(word)
		col += len(word)
	}
	return sb.String()
}
