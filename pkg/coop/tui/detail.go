package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

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
		m.writeAsyncHandlerExampleDetail(&md, node)
	}

	if node.State == coop.NodeSkipped && node.Activity != "" {
		md.WriteString("*Skipped: " + node.Activity + "*\n\n")
	}

	content := strings.TrimSpace(md.String())
	suffix := m.renderDetailSuffix(node, innerW)
	if content == "" && suffix == "" {
		return ""
	}

	var parts []string
	if header := m.renderDetailHeader(section); header != "" {
		parts = append(parts, header)
	}
	if content != "" {
		parts = append(parts, clampLines(m.renderMarkdown(content, innerW), innerW))
	}
	if suffix != "" {
		parts = append(parts, suffix)
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
	if target, ok := m.selectedReviewTarget(); ok && target.kind == "step" {
		suffix = "\n" + m.attentionWrapped("Waiting for you: c confirm all · r request changes", innerW)
	}
	if content == "" && suffix == "" {
		return ""
	}

	var parts []string
	if header := m.renderDetailHeader(section); header != "" {
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

func (m Model) renderDetailHeader(section string) string {
	if section == "Summary" {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(m.theme.Purple400).
		Bold(true).
		Render(section)
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
		md.WriteString("**Confirmation steps:** " + node.ReviewPrompt + "\n\n")
	}
	if node.Description == "" && node.ReviewPrompt == "" {
		md.WriteString("*No summary available for this step.*\n\n")
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

func (m Model) writeStepSummaryDetail(md *strings.Builder, ch *coop.SessionStep, width int) {
	wrapWidth := width - 2
	if wrapWidth < 20 {
		wrapWidth = 20
	}
	md.WriteString("Steps\n")
	for _, node := range ch.Nodes {
		md.WriteString("  " + stepNodeStatusLabel(node) + " " + node.Title + "\n")
	}
	md.WriteString("\n")
	if changed := stepChangedFiles(ch); changed != "" {
		md.WriteString("Changed\n")
		for _, line := range strings.Split(wordWrap(changed, wrapWidth), "\n") {
			md.WriteString("  " + line + "\n")
		}
		md.WriteString("\n")
	}
	if checks := stepConfirmationNodes(ch); checks != "" {
		md.WriteString("Confirmation steps\n")
		for _, line := range strings.Split(wordWrap(checks, wrapWidth), "\n") {
			md.WriteString("  " + line + "\n")
		}
		md.WriteString("\n")
	}
	md.WriteString("Agent help\n")
	for _, line := range strings.Split(wordWrap("The agent should run relevant checks, keep any app or server available, share a local URL when useful, and create or identify test data.", wrapWidth), "\n") {
		md.WriteString("  " + line + "\n")
	}
	md.WriteString("\n")
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
			md.WriteString("- `" + strings.Join(node.Events, "`, `") + "` webhook triggers for " + node.Title + "\n")
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

func stepNodeStatusLabel(node coop.SessionNode) string {
	switch node.State {
	case coop.NodeDone:
		return "✓"
	case coop.NodeActive:
		return "●"
	case coop.NodeReview:
		return "◆"
	case coop.NodeSkipped:
		return "–"
	default:
		return "○"
	}
}

func stepChangedFiles(ch *coop.SessionStep) string {
	var files []string
	for _, node := range ch.Nodes {
		if node.Implementation != nil && node.Implementation.File != "" {
			files = append(files, implementationFileLabel(node.Implementation))
		}
	}
	return strings.Join(files, ", ")
}

func stepConfirmationNodes(ch *coop.SessionStep) string {
	var agentChecks []string
	seenAgentCheck := map[string]bool{}
	for _, node := range ch.Nodes {
		for _, verification := range node.Verifications {
			check := strings.TrimSpace(verification.Check)
			if !verification.Passed || check == "" || seenAgentCheck[check] {
				continue
			}
			seenAgentCheck[check] = true
			if node.Title != "" {
				check = node.Title + ": " + check
			}
			agentChecks = append(agentChecks, check)
		}
	}
	if len(agentChecks) > 0 {
		return strings.Join(agentChecks, " ")
	}

	var checks []string
	for _, node := range ch.Nodes {
		if node.ReviewPrompt != "" {
			checks = append(checks, node.Title+": "+node.ReviewPrompt)
		}
	}
	return strings.Join(checks, " ")
}

func (m Model) writeAsyncHandlerCheckDetail(md *strings.Builder, node *coop.SessionNode) {
	if node.Type != coop.NodeAsyncHandler || len(node.Events) == 0 {
		return
	}
	commands := asyncEventTriggerCommands(node.Events)
	if len(commands) == 0 {
		return
	}
	md.WriteString("**How to verify:**\n\n")
	md.WriteString("1. `stripe listen --forward-to localhost:<port>/webhook`\n")
	if len(commands) == 1 {
		md.WriteString("2. `" + commands[0] + "`\n")
	} else {
		md.WriteString("2. Run each required trigger:\n")
		for _, command := range commands {
			md.WriteString("   - `" + command + "`\n")
		}
	}
	md.WriteString("3. Confirm your handler processes the event\n\n")
}

func (m Model) writeAsyncHandlerReferenceDetail(md *strings.Builder, node *coop.SessionNode) {
	if node.Type != coop.NodeAsyncHandler || len(node.Events) == 0 {
		return
	}
	commands := asyncEventTriggerCommands(node.Events)
	if len(commands) == 0 {
		return
	}
	if len(commands) == 1 {
		md.WriteString("**Webhook trigger:**\n\n")
		md.WriteString("`" + commands[0] + "`\n\n")
		return
	}
	md.WriteString("**Webhook triggers:**\n\n")
	for _, command := range commands {
		md.WriteString("- `" + command + "`\n")
	}
	md.WriteString("\n")
}

func (m Model) writeAsyncHandlerExampleDetail(md *strings.Builder, node *coop.SessionNode) {
	if node.Type != coop.NodeAsyncHandler || len(node.Events) == 0 {
		return
	}
	example := strings.TrimSpace(coop.GenerateWebhookExample(node.Events, m.detailLanguage()))
	if example == "" {
		return
	}
	md.WriteString("**Webhook handler example**\n")
	md.WriteString("```" + webhookExampleFenceLanguage(m.detailLanguage()) + "\n")
	md.WriteString(example + "\n")
	md.WriteString("```\n\n")
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

func (m Model) renderDetailSuffix(node *coop.SessionNode, width int) string {
	var suffix string
	if target, ok := m.selectedReviewTarget(); ok && target.kind == "node" && node.State == coop.NodeReview {
		suffix = "\n" + m.attentionWrapped("Waiting for you: c confirm · r request changes", width)
	}
	return suffix
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

func webhookExampleFenceLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "node", "javascript", "js":
		return "javascript"
	case "typescript", "ts":
		return "typescript"
	case "python", "py":
		return "python"
	case "go", "golang":
		return "go"
	default:
		return "text"
	}
}
