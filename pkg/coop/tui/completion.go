package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func (m Model) renderCompletionView() string {
	header := m.renderHeader()
	footer := m.renderCompletionFooter()
	if !m.ready {
		return header + "\n" + m.pinFooter(m.renderCompletionBody(), footer)
	}
	return m.renderPinnedViewport(header, footer)
}

func (m Model) renderCompletionBody() string {
	w := m.contentWidth() - 4

	summary := m.session.StepSummary()
	done := summary[coop.StepDone]
	total := m.session.TotalSteps()

	box := m.theme.SuccessStyle.Render(fmt.Sprintf("✓ Integration complete: %s", m.session.Blueprint)) +
		"\n" + m.theme.MutedStyle.Render(fmt.Sprintf("All %d steps done.", done))
	if total != done {
		box += m.theme.MutedStyle.Render(fmt.Sprintf(" (%d skipped)", total-done))
	}
	content := m.theme.DetailBoxStyle.Width(min(w, 70)).Render(box)

	if m.statusMessage != "" {
		content += "\n" + m.theme.AttentionStyle.Render("  "+m.statusMessage)
	}

	if m.session.ClaimURL != "" {
		content += "\n" + m.theme.DimmedStyle.Render("  ⚡ Claim your sandbox: ") + m.theme.BrandStyle.Hyperlink(m.session.ClaimURL).Render(m.session.ClaimURL)
		content += "\n" + m.theme.DimmedStyle.Render("    Press o to open in browser")
	}

	if receipt := m.renderCompletionReceipt(w); receipt != "" {
		content += "\n\n" + receipt
	}

	content += "\n\n" + m.theme.ChapterTitleStyle.Render("  Next steps")
	ruleWidth := min(w-4, 50)
	if ruleWidth < 0 {
		ruleWidth = 0
	}
	content += "\n  " + m.theme.ChapterRuleStyle.Render(strings.Repeat("─", ruleWidth))

	suggestions := m.getCompletionSuggestions()
	completed := m.getCompletedSuggestionIDs()

	for i, s := range suggestions {
		cur := "  "
		if i == m.cursor {
			cur = m.theme.BrandStyle.Render(cursorMarker)
		}
		isDone := completed[s.id]
		icon := m.theme.MutedStyle.Render("○")
		if isDone {
			icon = m.theme.SuccessStyle.Render("✓")
		}
		title := s.title
		if i == m.cursor {
			title = lipgloss.NewStyle().Bold(true).Render(title)
		} else if isDone {
			title = m.theme.DimmedStyle.Render(title)
		}
		content += "\n" + fmt.Sprintf("  %s%s %s", cur, icon, title)
		if s.desc != "" && !isDone {
			descW := min(w-10, 55)
			for _, dl := range wrapPlainText(s.desc, descW) {
				content += "\n      " + m.theme.DimmedStyle.Render(dl)
			}
		}
	}

	return content
}

func (m Model) renderCompletionReceipt(width int) string {
	if m.session == nil {
		return ""
	}

	var content strings.Builder
	built := m.completionBuiltItems()
	if len(built) > 0 {
		content.WriteString(m.theme.ChapterTitleStyle.Render("  Built") + "\n")
		builtW := min(width-4, 76)
		if builtW < 20 {
			builtW = 20
		}
		for i, line := range strings.Split(wordWrap(strings.Join(built, " · "), builtW), "\n") {
			prefix := "  " + m.theme.SuccessStyle.Render("✓") + " "
			if i > 0 {
				prefix = "    "
			}
			content.WriteString(prefix + line + "\n")
		}
	}

	checks := m.completionImportantChecks()
	if len(checks) > 0 {
		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString(m.theme.ChapterTitleStyle.Render("  Important checks") + "\n")
		checkW := min(width-8, 72)
		if checkW < 20 {
			checkW = 20
		}
		for _, check := range checks {
			wrapped := wrapPlainText(check, checkW)
			for i, line := range wrapped {
				prefix := "  - "
				if i > 0 {
					prefix = "    "
				}
				content.WriteString(prefix + line + "\n")
			}
		}
	}

	return strings.TrimRight(content.String(), "\n")
}

func wrapPlainText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	line := words[0]
	for _, word := range words[1:] {
		if lipgloss.Width(line)+1+lipgloss.Width(word) <= width {
			line += " " + word
			continue
		}
		lines = append(lines, line)
		line = word
	}
	lines = append(lines, line)
	return lines
}

func (m Model) completionBuiltItems() []string {
	var items []string
	for _, ch := range m.session.Chapters {
		if ch.Key == "context-chapter" {
			continue
		}
		done := 0
		relevant := 0
		for _, node := range ch.Nodes {
			if node.State == coop.StepSkipped {
				continue
			}
			relevant++
			if node.State == coop.StepDone {
				done++
			}
		}
		if relevant > 0 && done == relevant {
			items = append(items, ch.Title)
		}
	}
	return items
}

func (m Model) completionImportantChecks() []string {
	var checks []string
	seen := map[string]bool{}
	for _, ch := range m.session.Chapters {
		for _, node := range ch.Nodes {
			if node.State != coop.StepDone || node.ReviewPrompt == "" || seen[node.ReviewPrompt] {
				continue
			}
			seen[node.ReviewPrompt] = true
			checks = append(checks, node.ReviewPrompt)
			if len(checks) == 2 {
				return checks
			}
		}
	}
	return checks
}

func (m Model) renderCompletionFooter() string {
	h := m.help
	h.SetWidth(m.width)
	h.ShortSeparator = " · "
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "navigate")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		m.keys.Quit,
	}
	return m.theme.FooterStyle.Render("  " + h.ShortHelpView(bindings))
}

func (m Model) getCompletedSuggestionIDs() map[string]bool {
	result := make(map[string]bool)
	if m.session == nil || m.session.NextSteps == nil {
		return result
	}
	for _, id := range m.session.NextSteps.Completed {
		result[id] = true
	}
	return result
}

type completionSuggestion struct {
	id    string
	title string
	desc  string
}

func (m Model) getCompletionSuggestions() []completionSuggestion {
	if m.session != nil && m.session.NextSteps != nil && len(m.session.NextSteps.Suggestions) > 0 {
		var suggestions []completionSuggestion
		for _, s := range m.session.NextSteps.Suggestions {
			desc := s.Description
			if s.Reason != "" {
				desc = s.Reason
			}
			suggestions = append(suggestions, completionSuggestion{id: s.ID, title: s.Title, desc: desc})
		}
		return suggestions
	}
	completed := m.getCompletedSuggestionIDs()

	var suggestions []completionSuggestion

	if completed["summarize"] {
		suggestions = append(suggestions, completionSuggestion{id: "summarize", title: "Regenerate STRIPE.md", desc: "Update the summary with latest changes"})
	} else {
		suggestions = append(suggestions, completionSuggestion{id: "summarize", title: "Write a STRIPE.md summary", desc: "Generate a summary of what was built, API keys used, endpoints created, and how to run it"})
	}

	if completed["deploy"] || completed["deploy-update"] {
		suggestions = append(suggestions, completionSuggestion{id: "deploy", title: "Redeploy", desc: "Push latest changes to production"})
	} else {
		suggestions = append(suggestions, completionSuggestion{id: "deploy", title: "Deploy with Stripe Projects", desc: "Set up hosting, CI/CD, and environment management"})
	}

	suggestions = append(suggestions, completionSuggestion{id: "add-integration", title: "Add another Stripe feature", desc: "Subscriptions, Connect, billing portal, and more"})
	suggestions = append(suggestions, completionSuggestion{id: "done", title: "Finish", desc: "Close this session"})

	return suggestions
}
