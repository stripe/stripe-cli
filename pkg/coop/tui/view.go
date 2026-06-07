package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/stripe/stripe-cli/pkg/coop"
)

var detailSections = []string{"Summary", "Files", "Checks", "Reference"}

func (m Model) renderWaitingView() string {
	w := m.contentWidth() - 8
	if w < 25 {
		w = 25
	}

	waitingText := m.waitingMessage
	if waitingText == "" {
		waitingText = "Waiting for agent"
	}
	waitingLines := strings.Split(wordWrap(waitingText, w), "\n")
	subtitleLines := strings.Split(wordWrap("The agent is scanning the project and will start a session here. You can leave this open.", w), "\n")

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
	case node.State == coop.StepReview:
		annText = "Waiting for you to review"
		annStyle = func(s string) string { return AttentionStyle.Render(s) }
	case node.State == coop.StepActive && node.Activity != "":
		elapsed := ""
		if node.StartedAt != nil {
			dur := time.Since(*node.StartedAt).Truncate(time.Second)
			if dur >= time.Second {
				elapsed = " [" + formatDuration(dur) + "]"
			}
		}
		annText = "Agent working: " + node.Activity + elapsed
		annStyle = func(s string) string { return DimmedStyle.Render(s) }
	case node.State == coop.StepSkipped && node.Activity != "":
		annText = "— " + node.Activity
		annStyle = func(s string) string { return DimmedStyle.Render(s) }
	}

	line := fmt.Sprintf("  %s%s %s", cursor, icon, title)
	if label, style := m.stepStatusLabel(node); label != "" {
		line += "  " + style(label)
	}

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

func (m Model) stepStatusLabel(node coop.SessionNode) (string, func(string) string) {
	switch node.State {
	case coop.StepDone:
		return "Done", func(s string) string { return SuccessStyle.Render(s) }
	case coop.StepActive:
		return "Agent working", func(s string) string { return MutedStyle.Render(s) }
	case coop.StepReview:
		return "Needs review", func(s string) string { return AttentionStyle.Render(s) }
	case coop.StepSkipped:
		return "Skipped", func(s string) string { return DimmedStyle.Render(s) }
	case coop.StepPending:
		return "Pending", func(s string) string { return MutedStyle.Render(s) }
	default:
		return "", func(s string) string { return s }
	}
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

	w, innerW := m.detailWidths()

	var md strings.Builder
	section := detailSections[m.detailTab%len(detailSections)]
	md.WriteString("**Details: " + node.Title + "**\n\n")
	md.WriteString(m.renderDetailTabs(section) + "\n\n")

	switch section {
	case "Summary":
		m.writeSummaryDetail(&md, node)
	case "Files":
		m.writeImplementationDetail(&md, node, false)
	case "Checks":
		m.writeAsyncHandlerCheckDetail(&md, node)
		m.writeVerificationDetail(&md, node)
	case "Reference":
		currentSnippet := m.sdkSnippetStep == m.cursor && m.sdkSnippet != ""
		m.writeSDKReferenceDetail(&md, node, currentSnippet)
		m.writeAsyncHandlerReferenceDetail(&md, node)
	}

	if node.State == coop.StepSkipped && node.Activity != "" {
		md.WriteString("*Skipped: " + node.Activity + "*\n\n")
	}

	content := md.String()
	suffix := m.renderDetailSuffix(node)
	if content == "" && suffix == "" {
		return ""
	}

	rendered := m.renderMarkdown(content, innerW)
	return "    " + DetailBoxStyle.Width(w).Render(rendered+suffix)
}

func (m Model) renderDetailTabs(active string) string {
	var parts []string
	for _, section := range detailSections {
		if section == active {
			parts = append(parts, "["+section+"]")
		} else {
			parts = append(parts, section)
		}
	}
	return strings.Join(parts, "  ")
}

func (m Model) detailWidths() (int, int) {
	w := m.contentWidth() - 6
	if w < 30 {
		w = m.contentWidth() - 2
	}
	innerW := w - 4
	if innerW < 20 {
		innerW = 20
	}
	return w, innerW
}

func (m Model) detailLanguage() string {
	lang := m.session.Settings["language"]
	if lang == "" {
		lang = "javascript"
	}
	return lang
}

func (m Model) writeSummaryDetail(md *strings.Builder, node *coop.SessionNode) {
	if node.Description != "" {
		md.WriteString("**Summary**\n\n")
		md.WriteString(node.Description + "\n\n")
	}
	if node.ReviewPrompt != "" {
		md.WriteString("**You check**\n\n")
		md.WriteString(node.ReviewPrompt + "\n\n")
	}
	if node.Description == "" && node.ReviewPrompt == "" {
		md.WriteString("*No summary available for this step.*\n\n")
	}
}

func (m Model) writeAsyncHandlerCheckDetail(md *strings.Builder, node *coop.SessionNode) {
	if node.Type != coop.NodeAsyncHandler || len(node.Events) == 0 {
		return
	}
	md.WriteString("**How to verify:**\n\n")
	md.WriteString("1. `stripe listen --forward-to localhost:<port>/webhook`\n")
	md.WriteString("2. `stripe trigger " + node.Events[0] + "`\n")
	md.WriteString("3. Confirm your handler processes the event\n\n")
}

func (m Model) writeAsyncHandlerReferenceDetail(md *strings.Builder, node *coop.SessionNode) {
	if node.Type != coop.NodeAsyncHandler || len(node.Events) == 0 {
		return
	}
	md.WriteString("**Webhook trigger:**\n\n")
	md.WriteString("`stripe trigger " + node.Events[0] + "`\n\n")
}

func (m Model) writeSDKReferenceDetail(md *strings.Builder, node *coop.SessionNode, currentSnippet bool) {
	if node.Type != coop.NodeAPIRequest {
		return
	}
	if currentSnippet {
		md.WriteString("**Reference:**\n\n")
		md.WriteString("```" + m.detailLanguage() + "\n")
		md.WriteString(m.sdkSnippet + "\n")
		md.WriteString("```\n\n")
		return
	}
	if m.sdkLoading && m.sdkLoadingStep == m.cursor {
		md.WriteString("*Loading reference...*\n\n")
	}
}

func (m Model) writeImplementationDetail(md *strings.Builder, node *coop.SessionNode, currentSnippet bool) {
	if node.Implementation == nil {
		return
	}
	if currentSnippet {
		md.WriteString("---\n\n")
	}
	imp := node.Implementation
	md.WriteString("**Agent wrote:** `" + implementationFileLabel(imp) + "`\n\n")
	if imp.Snippet != "" {
		md.WriteString("```" + m.detailLanguage() + "\n")
		md.WriteString(imp.Snippet + "\n")
		md.WriteString("```\n\n")
	}
	if imp.Note != "" {
		md.WriteString("> " + imp.Note + "\n\n")
	}
}

func implementationFileLabel(imp *coop.Implementation) string {
	if imp.File == "" {
		return ""
	}
	if imp.Lines == "" {
		return imp.File
	}
	return imp.File + ":" + imp.Lines
}

func (m Model) writeVerificationDetail(md *strings.Builder, node *coop.SessionNode) {
	if len(node.Verifications) == 0 {
		return
	}
	for _, v := range node.Verifications {
		if v.Passed {
			md.WriteString("- ✓ " + v.Check + "\n")
		} else {
			md.WriteString("- ✗ " + v.Check + "\n")
		}
	}
	md.WriteString("\n")
}

func (m Model) renderDetailSuffix(node *coop.SessionNode) string {
	var suffix string
	if node.State == coop.StepReview {
		suffix = "\n" + AttentionStyle.Render("  Waiting for you: press c to confirm or r to request changes")
	}
	return suffix
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
		footer += AttentionStyle.Render("  Waiting for agent: no recent updates. Reconnect: stripe coop status") + "\n"
	}

	if m.statusMessage != "" {
		footer += AttentionStyle.Render("  "+m.statusMessage) + "\n"
	}

	if m.session != nil {
		if count := m.actionableReviewCount(); count > 0 {
			footer += AttentionStyle.Render(fmt.Sprintf("  Waiting for you: %d item(s) need review", count)) + "\n"
		}
	}

	if card := m.renderReviewCard(); card != "" {
		footer += card + "\n"
	}

	if m.rejecting {
		return footer + FooterStyle.Render("  enter send feedback  ·  esc cancel")
	}

	var parts []string
	if m.userMoved {
		parts = append(parts, "Viewing earlier steps", "f follow latest")
	} else {
		parts = append(parts, "↑↓ navigate")
	}
	parts = append(parts, "enter/e details")
	if m.expanded {
		parts = append(parts, "tab section", "esc close")
	}
	if m.session != nil && m.session.ClaimURL != "" {
		parts = append(parts, "o open claim URL")
	}
	if m.session != nil {
		if _, ok := m.selectedReviewTarget(); ok {
			parts = append(parts, SuccessStyle.Render("c confirm"))
			parts = append(parts, ErrorStyle.Render("r request changes"))
		}
	}
	parts = append(parts, "q quit")
	footer += FooterStyle.Render("  " + strings.Join(parts, "  ·  "))
	return footer
}

func (m Model) renderReviewCard() string {
	target, ok := m.selectedReviewTarget()
	if !ok {
		return ""
	}
	w := min(m.contentWidth()-4, 84)
	if w < 30 {
		w = m.contentWidth()
	}

	var lines []string
	prefix := "Review"
	if target.kind == "chapter" {
		prefix = "Review chapter"
	}
	lines = append(lines, AttentionStyle.Render(prefix+": ")+target.title)
	if changed := m.reviewChangedLabel(target.steps); changed != "" {
		lines = append(lines, MutedStyle.Render("Agent changed: ")+changed)
	}
	if verified := m.reviewVerificationLabel(target.steps); verified != "" {
		lines = append(lines, MutedStyle.Render("Agent verified: ")+verified)
	}
	check := m.reviewPromptLabel(target.steps)
	if check != "" {
		lines = append(lines, MutedStyle.Render("You check: ")+check)
	}
	if m.rejecting {
		input := m.rejectionInput
		if input == "" {
			input = DimmedStyle.Render("Describe what should change")
		}
		lines = append(lines, ErrorStyle.Render("Request changes: ")+input)
		if m.rejectionError != "" {
			lines = append(lines, ErrorStyle.Render(m.rejectionError))
		}
	}

	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, strings.Split(wordWrap(line, w-4), "\n")...)
	}
	return ReviewCardStyle.Width(w).Render(strings.Join(wrapped, "\n"))
}

func (m Model) reviewChangedLabel(steps []int) string {
	var labels []string
	seen := map[string]bool{}
	for _, step := range steps {
		node, err := m.session.NodeByNumber(step)
		if err != nil || node.Implementation == nil || node.Implementation.File == "" {
			continue
		}
		label := implementationFileLabel(node.Implementation)
		if !seen[label] {
			seen[label] = true
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return ""
	}
	if len(labels) > 3 {
		return strings.Join(labels[:3], ", ") + fmt.Sprintf(" +%d more", len(labels)-3)
	}
	return strings.Join(labels, ", ")
}

func (m Model) reviewVerificationLabel(steps []int) string {
	passed := 0
	total := 0
	for _, step := range steps {
		node, err := m.session.NodeByNumber(step)
		if err != nil {
			continue
		}
		for _, v := range node.Verifications {
			total++
			if v.Passed {
				passed++
			}
		}
	}
	if total == 0 {
		return ""
	}
	if passed == total {
		return fmt.Sprintf("%d check(s) passed", passed)
	}
	return fmt.Sprintf("%d/%d check(s) passed", passed, total)
}

func (m Model) reviewPromptLabel(steps []int) string {
	var prompts []string
	seen := map[string]bool{}
	for _, step := range steps {
		node, err := m.session.NodeByNumber(step)
		if err != nil || node.ReviewPrompt == "" || seen[node.ReviewPrompt] {
			continue
		}
		seen[node.ReviewPrompt] = true
		prompts = append(prompts, node.ReviewPrompt)
	}
	if len(prompts) == 0 {
		return "Confirm the completed work matches this step and its verification evidence."
	}
	if len(prompts) > 2 {
		return strings.Join(prompts[:2], " ") + " Open details for the remaining checks."
	}
	return strings.Join(prompts, " ")
}

func (m Model) actionableReviewCount() int {
	if m.session == nil {
		return 0
	}
	count := 0
	countedChapters := map[int]bool{}
	step := 0
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			step++
			if m.session.Chapters[i].Nodes[j].State != coop.StepReview || !m.reviewIsActionable(step) {
				continue
			}
			if m.session.ReviewGranularityForStep(step) == coop.ReviewGranularityChapter {
				if !countedChapters[i] {
					count++
					countedChapters[i] = true
				}
				continue
			}
			count++
		}
	}
	return count
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
	header := m.renderHeader()
	footer := m.renderCompletionFooter()
	if !m.ready {
		return header + "\n" + m.pinFooter(m.renderCompletionBody(), footer)
	}
	return header + "\n" + m.viewport.View() + "\n" + footer
}

func (m Model) renderCompletionBody() string {
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

	if m.statusMessage != "" {
		content += "\n" + AttentionStyle.Render("  "+m.statusMessage)
	}

	if m.session.ClaimURL != "" {
		content += "\n" + DimmedStyle.Render("  ⚡ Claim your sandbox: ") + BrandStyle.Render(m.session.ClaimURL)
		content += "\n" + DimmedStyle.Render("    Press o to open in browser")
	}

	content += "\n\n" + ChapterTitleStyle.Render("  Next steps")
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

	return content
}

func (m Model) renderCompletionFooter() string {
	return FooterStyle.Render("  ↑↓ navigate  ·  enter select  ·  q quit")
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
	suggestions = append(suggestions, completionSuggestion{id: "done", title: "Finish", desc: "Close this session"})

	return suggestions
}
