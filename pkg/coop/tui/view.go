package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func (m Model) renderWaitingView() string {
	w := m.contentWidth() - 8
	if w < 25 {
		w = 25
	}

	waitingLines := strings.Split(wordWrap("Waiting for the agent to explore your codebase and pick an integration...", w), "\n")
	subtitleLines := strings.Split(wordWrap("The agent will ask what you want to build, then start a session. You'll see progress here.", w), "\n")

	var content string
	content = HeaderStyle.Render("● Stripe Co-op") + "\n\n"
	for i, line := range waitingLines {
		if i == 0 {
			content += "  " + m.spinner.View() + " " + BrandStyle.Render(line) + "\n"
		} else {
			content += "    " + BrandStyle.Render(line) + "\n"
		}
	}
	content += "\n"
	for _, line := range subtitleLines {
		content += "  " + MutedStyle.Render(line) + "\n"
	}

	footer := FooterStyle.Render("  q quit")
	return m.pinFooter(content, footer)
}

func (m Model) renderHeader() string {
	if m.session == nil {
		return HeaderStyle.Render("● Stripe Co-op")
	}

	left := HeaderStyle.Render("● Stripe Co-op")
	right := m.session.Blueprint
	if lang, ok := m.session.Settings["language"]; ok {
		right += " · " + lang
	}

	summary := m.session.StepSummary()
	done := summary[coop.StepDone]
	skipped := summary[coop.StepSkipped]
	total := m.session.TotalSteps()

	progress := fmt.Sprintf("%d/%d", done, total-skipped)
	if skipped > 0 {
		progress += fmt.Sprintf(" · %d skipped", skipped)
	}
	rightPart := MutedStyle.Render(right + " · " + progress)

	available := m.contentWidth()
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(rightPart)

	var header string
	if leftW+rightW+4 > available {
		header = left + "\n  " + rightPart
	} else {
		header = left + strings.Repeat(" ", available-leftW-rightW-2) + rightPart
	}

	if m.session.ClaimURL != "" {
		url := m.session.ClaimURL
		maxW := available - 10
		if maxW > 0 && len(url) > maxW {
			url = url[:maxW-1] + "…"
		}
		header += "\n" + DimmedStyle.Render("  ⚡ ") + BrandStyle.Render(url)
	}

	return header
}

func (m Model) renderStepList() string {
	if m.session == nil {
		return ""
	}

	w := m.contentWidth()
	var lines []string
	stepIdx := 0

	ruleWidth := w - 4
	if ruleWidth < 20 {
		ruleWidth = 20
	}
	if ruleWidth > 80 {
		ruleWidth = 80
	}

	for _, ch := range m.session.Chapters {
		lines = append(lines, "")
		lines = append(lines, "  "+ChapterTitleStyle.Render(ch.Title))
		lines = append(lines, "  "+ChapterRuleStyle.Render(strings.Repeat("─", ruleWidth)))

		for _, node := range ch.Nodes {
			lines = append(lines, m.renderStepLine(node, stepIdx))
			if m.expanded && stepIdx == m.cursor {
				if detail := m.renderDetail(); detail != "" {
					lines = append(lines, detail)
				}
			}
			stepIdx++
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderStepLine(node coop.SessionNode, idx int) string {
	icon := m.stepIcon(node)

	cursor := "  "
	if idx == m.cursor {
		cursor = BrandStyle.Render("▸ ")
	}

	title := node.Title
	if node.State == coop.StepSkipped {
		title = DimmedStyle.Render(title)
	} else if idx == m.cursor {
		title = lipgloss.NewStyle().Bold(true).Render(title)
	}

	var annText string
	var annStyle func(string) string
	switch {
	case node.Implementation != nil && node.Implementation.File != "":
		ann := node.Implementation.File
		if node.Implementation.Lines != "" {
			ann += ":" + node.Implementation.Lines
		}
		annText = ann
		annStyle = func(s string) string { return FileAnnotationStyle.Render(s) }
	case node.State == coop.StepActive && node.Activity != "":
		elapsed := ""
		if node.StartedAt != nil {
			dur := time.Since(*node.StartedAt).Truncate(time.Second)
			if dur >= time.Second {
				elapsed = " [" + formatDuration(dur) + "]"
			}
		}
		annText = node.Activity + elapsed
		annStyle = func(s string) string { return DimmedStyle.Render(s) }
	case node.State == coop.StepSkipped && node.Activity != "":
		annText = "— " + node.Activity
		annStyle = func(s string) string { return DimmedStyle.Render(s) }
	}

	line := fmt.Sprintf("  %s%s %s", cursor, icon, title)

	if annText != "" {
		wrapW := m.contentWidth() - 8
		if wrapW < 20 {
			wrapW = 20
		}
		wrapped := wordWrap(annText, wrapW)
		for _, wl := range strings.Split(wrapped, "\n") {
			line += "\n      " + annStyle(wl)
		}
	}

	return line
}

func (m Model) stepIcon(node coop.SessionNode) string {
	// All icons rendered at fixed 1-cell width for alignment
	switch node.State {
	case coop.StepDone:
		return SuccessStyle.Render("✓")
	case coop.StepActive:
		return lipgloss.NewStyle().Width(1).Render(m.spinner.View())
	case coop.StepReview:
		return AttentionStyle.Render("◆")
	case coop.StepSkipped:
		return DimmedStyle.Render("–")
	default:
		return MutedStyle.Render("○")
	}
}

func (m Model) renderDetail() string {
	if m.session == nil {
		return ""
	}
	node, err := m.session.NodeByNumber(m.cursor + 1)
	if err != nil {
		return ""
	}

	w := m.contentWidth() - 6
	if w < 30 {
		w = m.contentWidth() - 2
	}
	innerW := w - 4
	if innerW < 20 {
		innerW = 20
	}

	var md strings.Builder

	if node.Description != "" {
		md.WriteString(node.Description + "\n\n")
	}

	if node.State == coop.StepSkipped && node.Activity != "" {
		md.WriteString("*Skipped: " + node.Activity + "*\n")
		rendered := m.renderMarkdown(md.String(), innerW)
		return "    " + DetailBoxStyle.Width(w).Render(rendered)
	}

	if node.Type == coop.NodeAsyncHandler && len(node.Events) > 0 {
		md.WriteString("**How to verify:**\n\n")
		md.WriteString("1. `stripe listen --forward-to localhost:<port>/webhook`\n")
		md.WriteString("2. `stripe trigger " + node.Events[0] + "`\n")
		md.WriteString("3. Confirm your handler processes the event\n\n")
	}

	currentSnippet := m.sdkSnippetStep == m.cursor && m.sdkSnippet != ""
	if node.Type == coop.NodeAPIRequest && currentSnippet {
		lang := m.session.Settings["language"]
		if lang == "" {
			lang = "javascript"
		}
		md.WriteString("**Reference:**\n\n")
		md.WriteString("```" + lang + "\n")
		md.WriteString(m.sdkSnippet + "\n")
		md.WriteString("```\n\n")
	} else if node.Type == coop.NodeAPIRequest && m.sdkLoading && m.sdkLoadingStep == m.cursor {
		md.WriteString("*Loading reference...*\n\n")
	}

	if node.Implementation != nil {
		imp := node.Implementation
		if currentSnippet {
			md.WriteString("---\n\n")
		}
		fileLabel := ""
		if imp.File != "" {
			fileLabel = imp.File
			if imp.Lines != "" {
				fileLabel += ":" + imp.Lines
			}
		}
		md.WriteString("**Agent wrote:** `" + fileLabel + "`\n\n")
		if imp.Snippet != "" {
			lang := m.session.Settings["language"]
			if lang == "" {
				lang = "javascript"
			}
			md.WriteString("```" + lang + "\n")
			md.WriteString(imp.Snippet + "\n")
			md.WriteString("```\n\n")
		}
		if imp.Note != "" {
			md.WriteString("> " + imp.Note + "\n\n")
		}
	}

	if len(node.Verifications) > 0 {
		for _, v := range node.Verifications {
			if v.Passed {
				md.WriteString("- ✓ " + v.Check + "\n")
			} else {
				md.WriteString("- ✗ " + v.Check + "\n")
			}
		}
		md.WriteString("\n")
	}

	var suffix string
	if node.State == coop.StepReview {
		suffix = "\n" + AttentionStyle.Render("  ▶ Press c to confirm")
	}

	content := md.String()
	if content == "" && suffix == "" {
		return ""
	}

	rendered := m.renderMarkdown(content, innerW)
	return "    " + DetailBoxStyle.Width(w).Render(rendered+suffix)
}

func (m Model) renderMarkdown(content string, width int) string {
	if content == "" {
		return ""
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}
	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimRight(rendered, "\n ")
}

func (m Model) renderFooter() string {
	// Completion view has its own footer — don't render step footer
	if m.session != nil && m.session.IsComplete() {
		return ""
	}

	var footer string

	// Agent disconnected warning
	if m.agentIdle() {
		footer += AttentionStyle.Render("  ⚠ Agent appears idle. Reconnect: stripe coop status") + "\n"
	}

	if m.session != nil {
		summary := m.session.StepSummary()
		if summary[coop.StepReview] > 0 {
			footer += AttentionStyle.Render(fmt.Sprintf("  ◆ %d step(s) awaiting your confirmation", summary[coop.StepReview])) + "\n"
		}
	}

	parts := []string{"↑↓ navigate", "enter/e details"}
	if m.session != nil && m.session.ClaimURL != "" {
		parts = append(parts, "o open claim URL")
	}
	if m.session != nil {
		node, _ := m.session.NodeByNumber(m.cursor + 1)
		if node != nil && node.State == coop.StepReview {
			parts = append(parts, SuccessStyle.Render("c confirm"))
			parts = append(parts, ErrorStyle.Render("r reject"))
		}
	}
	parts = append(parts, "q quit")
	footer += FooterStyle.Render("  " + strings.Join(parts, "  ·  "))
	return footer
}

func (m Model) agentIdle() bool {
	if m.session == nil || m.store == nil {
		return false
	}
	if m.session.IsComplete() {
		return false
	}
	// Check if agent is actively polling (heartbeat file exists and is fresh)
	age := m.store.HeartbeatAge(m.sessionID)
	if age >= 0 && age < 5*time.Second {
		return false // agent is actively polling via await
	}
	// No heartbeat — check if session has been updated recently
	if m.lastUpdateTime.IsZero() {
		return false
	}
	return time.Since(m.lastUpdateTime) > 2*time.Minute
}

func (m Model) renderCompletionView() string {
	w := m.contentWidth() - 4

	summary := m.session.StepSummary()
	done := summary[coop.StepDone]
	total := m.session.TotalSteps()

	box := SuccessStyle.Render(fmt.Sprintf("  ✓ Integration complete: %s", m.session.Blueprint)) +
		"\n" + MutedStyle.Render(fmt.Sprintf("  All %d steps done.", done))
	if total != done {
		box += MutedStyle.Render(fmt.Sprintf(" (%d skipped)", total-done))
	}
	content := DetailBoxStyle.Width(min(w, 70)).Render(box)

	if m.session.ClaimURL != "" {
		content += "\n" + DimmedStyle.Render("  ⚡ Claim your sandbox: ") + BrandStyle.Render(m.session.ClaimURL)
		content += "\n" + DimmedStyle.Render("    Press o to open in browser")
	}

	content += "\n\n" + ChapterTitleStyle.Render("  What's next?")
	ruleWidth := min(w-4, 50)
	if ruleWidth < 0 {
		ruleWidth = 0
	}
	content += "\n  " + ChapterRuleStyle.Render(strings.Repeat("─", ruleWidth))

	suggestions := m.getCompletionSuggestions()
	completed := m.getCompletedSuggestionIDs()

	for i, s := range suggestions {
		cur := "  "
		if i == m.cursor {
			cur = BrandStyle.Render("▸ ")
		}
		isDone := completed[s.id]
		icon := MutedStyle.Render("○")
		if isDone {
			icon = SuccessStyle.Render("✓")
		}
		title := s.title
		if i == m.cursor {
			title = lipgloss.NewStyle().Bold(true).Render(title)
		} else if isDone {
			title = DimmedStyle.Render(title)
		}
		content += "\n" + fmt.Sprintf("  %s%s %s", cur, icon, title)
		if s.desc != "" && !isDone {
			descW := min(w-10, 55)
			for _, dl := range strings.Split(wordWrap(s.desc, descW), "\n") {
				content += "\n      " + DimmedStyle.Render(dl)
			}
		}
	}

	footer := FooterStyle.Render("  ↑↓ navigate  ·  enter select  ·  q quit")
	return m.pinFooter(content, footer)
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

	// Adapt suggestions based on what's been done
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
	suggestions = append(suggestions, completionSuggestion{id: "done", title: "I'm done", desc: "End the session"})

	return suggestions
}
