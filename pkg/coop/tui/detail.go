package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stripe/stripe-cli/pkg/coop"
)

var detailSections = []string{"Summary", "Files", "Checks", "Reference"}

func (m Model) renderDetail() string {
	if m.session == nil {
		return ""
	}
	if m.selected.kind == navigationStep {
		return m.renderStepDetail(m.selected.stepIndex)
	}
	nodeIndex, ok := m.selectedNodeIndex()
	if !ok {
		return ""
	}
	node, err := m.session.NodeByNumber(nodeIndex + 1)
	if err != nil {
		return ""
	}

	w, innerW := m.detailWidths()

	var md strings.Builder
	section := detailSections[m.detailTab%len(detailSections)]
	currentSnippet := m.sdkSnippetNode == nodeIndex && m.sdkSnippet != ""

	switch section {
	case "Summary":
		m.writeSummaryDetail(&md, node)
		m.writeStepSDKSnippetDetail(&md, node, currentSnippet)
	case "Files":
		m.writeImplementationDetail(&md, node, false)
	case "Checks":
		m.writeReviewCommandDetail(&md, node)
		m.writeAsyncHandlerCheckDetail(&md, node)
		m.writeVerificationDetail(&md, node)
	case "Reference":
		m.writeSDKReferenceDetail(&md, node, currentSnippet)
		m.writeAsyncHandlerReferenceDetail(&md, node)
	}

	if node.State == coop.NodeSkipped && node.Activity != "" {
		md.WriteString("*Skipped: " + node.Activity + "*\n\n")
	}

	content := strings.TrimSpace(md.String())
	if content == "" {
		return ""
	}

	var parts []string
	if header := m.renderDetailHeader(section, innerW); header != "" {
		parts = append(parts, header)
	}
	if content != "" {
		parts = append(parts, clampLines(m.renderMarkdown(content, innerW), innerW))
	}
	body := clampLines(strings.Join(parts, "\n"), innerW)
	box := m.theme.DetailBoxStyle.Width(w).Render(body)
	return indentBlock(box, detailIndent)
}

func (m Model) renderStepDetail(stepIndex int) string {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return ""
	}
	w, innerW := m.detailWidths()

	var md strings.Builder
	section := detailSections[m.detailTab%len(detailSections)]
	ch := &m.session.Steps[stepIndex]
	switch section {
	case "Summary":
		m.writeStepSummaryDetail(&md, ch, innerW)
	case "Files":
		m.writeStepFilesDetail(&md, ch)
	case "Checks":
		m.writeStepChecksDetail(&md, ch)
	case "Reference":
		m.writeStepReferenceDetail(&md, ch)
	}

	content := strings.TrimSpace(md.String())
	suffix := ""
	if _, ok := m.selectedReviewTarget(); ok {
		suffix = "\n" + m.attentionWrapped("Waiting for you: c confirm step · r changes", innerW)
	}
	if content == "" && suffix == "" {
		return ""
	}

	var parts []string
	if header := m.renderDetailHeader(section, innerW); header != "" {
		parts = append(parts, header)
	}
	if content != "" {
		if section == "Summary" {
			parts = append(parts, clampLines(content, innerW))
		} else {
			parts = append(parts, clampLines(m.renderMarkdown(content, innerW), innerW))
		}
	}
	if suffix != "" {
		parts = append(parts, suffix)
	}
	body := clampLines(strings.Join(parts, "\n"), innerW)
	box := m.theme.DetailBoxStyle.Width(w).Render(body)
	return indentBlock(box, detailIndent)
}

// renderDetailHeader draws the tab strip.
//
// The detail box has always had four sections cycled with tab, but nothing on
// screen said so: the section you land on rendered no header at all, and the
// others printed a bare word with no hint that it was one of several or that
// tab moved between them. Listing all four, with the active one marked, makes
// the control visible from the tab you start on.
func (m Model) renderDetailHeader(section string, width int) string {
	active := lipgloss.NewStyle().Foreground(m.theme.Purple400).Bold(true)
	separator := m.theme.DimmedStyle.Render(" · ")

	var parts []string
	for _, name := range detailSections {
		if name == section {
			parts = append(parts, active.Render(name))
			continue
		}
		parts = append(parts, m.theme.DimmedStyle.Render(name))
	}
	strip := strings.Join(parts, separator)

	// Too narrow for the full strip — say where you are and that tab moves.
	if width > 0 && lipgloss.Width(ansi.Strip(strip)) > width {
		position := 1
		for i, name := range detailSections {
			if name == section {
				position = i + 1
			}
		}
		return active.Render(section) + m.theme.DimmedStyle.Render(
			fmt.Sprintf(" %d/%d · tab", position, len(detailSections)))
	}
	return strip
}

func (m Model) detailWidths() (int, int) {
	frameW, _ := m.theme.DetailBoxStyle.GetFrameSize()
	w := m.outlineRuleWidth()
	if w < 12 {
		w = 12
	}
	innerW := w - frameW
	if innerW < 8 {
		innerW = 8
	}
	return w, innerW
}

func indentBlock(s string, spaces int) string {
	if spaces <= 0 || s == "" {
		return s
	}
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
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
		md.WriteString(node.Description + "\n\n")
	}
	if node.ReviewPrompt != "" {
		md.WriteString("**To confirm:** " + node.ReviewPrompt + "\n\n")
	}
	if node.Description == "" && node.ReviewPrompt == "" {
		md.WriteString("*No summary available for this task.*\n\n")
	}
}

func (m Model) writeStepSDKSnippetDetail(md *strings.Builder, node *coop.SessionNode, currentSnippet bool) {
	if node.Type != coop.NodeAPIRequest || node.Request == nil {
		return
	}
	if currentSnippet {
		md.WriteString("**SDK example**\n")
		md.WriteString("```" + m.detailLanguage() + "\n")
		md.WriteString(m.sdkSnippet + "\n")
		md.WriteString("```\n\n")
		return
	}
	if nodeIndex, ok := m.selectedNodeIndex(); ok && m.sdkLoading && m.sdkLoadingNode == nodeIndex {
		md.WriteString("*Loading SDK example...*\n\n")
	}
}

// writeStepSummaryDetail renders the expanded step.
//
// It mirrors the review card deliberately: same headings, same hues, same
// flush-left layout. The two used to diverge — the card had structure while
// this view emitted unstyled headings, hardcoded indents, and every task's
// prose concatenated into one paragraph — so opening a step took you from a
// readable summary to a wall of text.
func (m Model) writeStepSummaryDetail(md *strings.Builder, ch *coop.SessionStep, width int) {
	wrapWidth := width - 2
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	if goal := strings.TrimSpace(ch.DescriptionText()); goal != "" {
		writeWrapped(md, m.theme.MutedStyle.Render("Goal ")+goal, wrapWidth)
		md.WriteString("\n")
	}

	md.WriteString(evidenceLabel(m.theme, "Tasks") + "\n")
	for _, node := range ch.Nodes {
		label, style := m.nodeStatusLabel(node)
		line := m.nodeIcon(node) + " " + node.Title
		if label != "" {
			line += "  " + style(label)
		}
		writeWrapped(md, line, wrapWidth)
		if node.Implementation != nil && node.Implementation.File != "" {
			writeWrapped(md, m.theme.FileAnnotationStyle.Render(
				truncatePath(implementationFileLabel(node.Implementation), wrapWidth)), wrapWidth)
		}
		if note := taskEvidence(node); note != "" {
			writeWrapped(md, m.theme.DimmedStyle.Render(note), wrapWidth)
		}
	}
	md.WriteString("\n")

	if prompts := stepReviewPrompts(ch); len(prompts) > 0 {
		md.WriteString(actionLabel(m.theme, "Do this") + "\n")
		for _, prompt := range prompts {
			writeWrapped(md, prompt, wrapWidth)
		}
		md.WriteString("\n")
	}

	failed, passed := stepCheckResults(ch)
	if len(failed) > 0 || passed > 0 {
		md.WriteString(evidenceLabel(m.theme, "Checks") + "\n")
		for _, check := range failed {
			writeWrapped(md, m.theme.ErrorStyle.Render("✗ ")+check, wrapWidth)
		}
		if passed > 0 {
			md.WriteString(m.theme.SuccessStyle.Render("✓ ") +
				m.theme.MutedStyle.Render(pluralChecks(passed)+" passed") + "\n")
		}
		md.WriteString("\n")
	}
}

// writeWrapped wraps to the available width with no indentation: hierarchy in
// this view comes from weight and hue, as it does in the card.
func writeWrapped(md *strings.Builder, text string, width int) {
	for _, line := range strings.Split(wordWrap(text, width), "\n") {
		md.WriteString(line + "\n")
	}
}

// taskEvidence is the agent's own note about a task, condensed to one line.
func taskEvidence(node coop.SessionNode) string {
	for _, verification := range node.Verifications {
		if check := summarizeCheck(verification.Check, failedCheckBudget); check != "" {
			return check
		}
	}
	return ""
}

func stepReviewPrompts(ch *coop.SessionStep) []string {
	var prompts []string
	seen := map[string]bool{}
	for _, node := range ch.Nodes {
		if node.ReviewPrompt == "" || seen[node.ReviewPrompt] {
			continue
		}
		seen[node.ReviewPrompt] = true
		prompts = append(prompts, node.ReviewPrompt)
	}
	return prompts
}

// stepCheckResults names failures and counts passes, as the card does.
func stepCheckResults(ch *coop.SessionStep) ([]string, int) {
	var failed []string
	passed := 0
	for _, node := range ch.Nodes {
		for _, verification := range node.Verifications {
			if verification.Passed {
				passed++
				continue
			}
			if check := summarizeCheck(verification.Check, failedCheckBudget); check != "" {
				failed = append(failed, node.Title+": "+check)
			}
		}
	}
	return failed, passed
}

func (m Model) writeStepFilesDetail(md *strings.Builder, ch *coop.SessionStep) {
	wrote := false
	for _, node := range ch.Nodes {
		if node.Implementation == nil || node.Implementation.File == "" {
			continue
		}
		md.WriteString("- `" + implementationFileLabel(node.Implementation) + "` — " + node.Title + "\n")
		wrote = true
	}
	if wrote {
		md.WriteString("\n")
		return
	}
	md.WriteString("*No files reported for this step yet.*\n\n")
}

func (m Model) writeStepChecksDetail(md *strings.Builder, ch *coop.SessionStep) {
	wrote := false
	for _, node := range ch.Nodes {
		if node.ReviewPrompt != "" {
			md.WriteString("- " + node.Title + ": " + node.ReviewPrompt + "\n")
			wrote = true
		}
		for _, verification := range node.Verifications {
			prefix := "✗"
			if verification.Passed {
				prefix = "✓"
			}
			md.WriteString("- " + prefix + " " + node.Title + ": " + verification.Check + "\n")
			wrote = true
		}
		if command := reviewCommandForNode(&node); command != "" {
			md.WriteString("- `" + strings.ReplaceAll(command, "`", "'") + "`\n")
			wrote = true
		}
	}
	if wrote {
		md.WriteString("\n")
		return
	}
	md.WriteString("*No confirmation checks reported for this step yet.*\n\n")
}
func (m Model) writeStepReferenceDetail(md *strings.Builder, ch *coop.SessionStep) {
	wrote := false
	for _, node := range ch.Nodes {
		if node.Type == coop.NodeAsyncHandler && len(node.Events) > 0 {
			md.WriteString("- `" + node.Events[0] + "` webhook trigger for " + node.Title + "\n")
			wrote = true
		}
		if node.Type == coop.NodeAPIRequest && node.Request != nil {
			md.WriteString("- `" + strings.ToUpper(node.Request.Method) + " " + node.Request.Path + "` for " + node.Title + "\n")
			wrote = true
		}
	}
	if wrote {
		md.WriteString("\n")
		return
	}
	md.WriteString("*No reference metadata for this step yet.*\n\n")
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

func (m Model) writeReviewCommandDetail(md *strings.Builder, node *coop.SessionNode) {
	command := reviewCommandForNode(node)
	if command == "" {
		return
	}
	md.WriteString("**Review command:**\n\n")
	md.WriteString("`" + strings.ReplaceAll(command, "`", "'") + "`\n\n")
}

func (m Model) writeSDKReferenceDetail(md *strings.Builder, node *coop.SessionNode, currentSnippet bool) {
	if node.Type != coop.NodeAPIRequest {
		return
	}
	if currentSnippet {
		md.WriteString("**SDK example**\n")
		md.WriteString("```" + m.detailLanguage() + "\n")
		md.WriteString(m.sdkSnippet + "\n")
		md.WriteString("```\n\n")
		return
	}
	if nodeIndex, ok := m.selectedNodeIndex(); ok && m.sdkLoading && m.sdkLoadingNode == nodeIndex {
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

func (m Model) attentionWrapped(text string, width int) string {
	if width < 1 {
		width = 1
	}
	lines := strings.Split(wordWrap(text, width), "\n")
	for i, line := range lines {
		lines[i] = m.theme.AttentionStyle.Render(line)
	}
	return strings.Join(lines, "\n")
}
