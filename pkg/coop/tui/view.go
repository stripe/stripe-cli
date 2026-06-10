package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/coop"
)

var detailSections = []string{"Summary", "Files", "Checks", "Reference"}

const (
	minViewportHeight   = 1
	terminalScrollGuard = 1
	rowCursorWidth      = 2
	rowRightGap         = 2
	maxRuleWidth        = 80
	detailIndent        = rowCursorWidth
	cursorMarker        = "> "
)

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
	content = m.theme.HeaderStyle.Render("● Stripe Co-op") + "\n\n"
	for i, line := range waitingLines {
		if i == 0 {
			content += "  " + m.spinner.View() + " " + m.theme.BrandStyle.Render(line) + "\n"
		} else {
			content += "    " + m.theme.BrandStyle.Render(line) + "\n"
		}
	}
	content += "\n"
	for _, line := range subtitleLines {
		content += "  " + m.theme.MutedStyle.Render(line) + "\n"
	}

	footer := m.theme.FooterStyle.Render("  q quit")
	return m.pinFooter(content, footer)
}

func (m Model) renderHeader() string {
	if m.session == nil {
		return m.theme.HeaderStyle.Render("● Stripe Co-op")
	}

	left := m.theme.HeaderStyle.Render("● Stripe Co-op")
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
	rightPart := m.theme.MutedStyle.Render(right + " · " + progress)

	available := m.contentWidth()
	var header string
	if lipgloss.Width(left)+lipgloss.Width(rightPart)+4 > available {
		header = left + "\n  " + rightPart
	} else {
		header = lipgloss.JoinHorizontal(lipgloss.Top, left, lipgloss.PlaceHorizontal(available-lipgloss.Width(left), lipgloss.Right, rightPart))
	}

	if m.session.ClaimURL != "" {
		url := m.session.ClaimURL
		maxW := available - 10
		if maxW > 0 && len(url) > maxW {
			url = url[:maxW-1] + "…"
		}
		header += "\n" + m.theme.DimmedStyle.Render("  ⚡ ") + m.theme.BrandStyle.Hyperlink(m.session.ClaimURL).Render(url)
	}

	return header
}

func (m Model) renderStepList() string {
	if m.useSplitWorkspace() {
		return m.renderSplitWorkspace()
	}
	return m.renderStepOutline().content
}

func (m Model) useSplitWorkspace() bool {
	return m.width >= 100 && m.session != nil && !m.session.IsComplete()
}

func (m Model) renderSplitWorkspace() string {
	leftW := m.width / 3
	if leftW < 34 {
		leftW = 34
	}
	if leftW > 48 {
		leftW = 48
	}
	gapW := 2
	rightW := m.width - leftW - gapW
	if rightW < 40 {
		return m.renderStepOutline().content
	}

	nav := m.renderStepOutline().content
	detail := m.renderSplitDetail(rightW)
	left := lipgloss.NewStyle().
		Width(leftW).
		MaxWidth(leftW).
		Render(nav)
	right := lipgloss.NewStyle().
		Width(rightW).
		MaxWidth(rightW).
		Render(detail)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gapW), right)
}

func (m Model) renderSplitDetail(width int) string {
	if !m.expanded {
		if m.selected.kind == navigationChapter {
			return m.theme.MutedStyle.Render("Press enter to inspect this section.")
		}
		return m.theme.MutedStyle.Render("Press enter to inspect this step.")
	}
	detail := strings.TrimSpace(m.renderDetail())
	if detail == "" {
		return m.theme.MutedStyle.Render("No details available yet.")
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(detail)
}

type renderedOutline struct {
	content        string
	navigationLine map[int]navigationItem
}

func (m Model) renderStepOutline() renderedOutline {
	if m.session == nil {
		return renderedOutline{navigationLine: map[int]navigationItem{}}
	}

	var lines []string
	navigationLines := map[int]navigationItem{}
	stepIdx := 0

	ruleWidth := m.outlineRuleWidth()

	for chIdx, ch := range m.session.Chapters {
		chapterItem := navigationItem{kind: navigationChapter, chapterIndex: chIdx}
		chapterSelected := m.navigationItemSelected(chapterItem)
		chapterReviewReady := m.chapterReviewReady(chIdx)
		lines = append(lines, "")
		navigationLines[len(lines)] = chapterItem
		lines = append(lines, m.renderChapterLine(ch, chIdx, chapterSelected))
		lines = append(lines, strings.Repeat(" ", rowCursorWidth)+m.theme.ChapterRuleStyle.Render(strings.Repeat("─", ruleWidth)))
		if m.expanded && chapterSelected && !m.useSplitWorkspace() {
			if detail := m.renderDetail(); detail != "" {
				lines = append(lines, detail)
			}
		}

		if m.chapterCollapsed(chIdx) {
			stepIdx += len(ch.Nodes)
			continue
		}
		for _, node := range ch.Nodes {
			stepItem := navigationItem{kind: navigationStep, stepIndex: stepIdx, chapterIndex: chIdx}
			stepSelected := m.navigationItemSelected(stepItem)
			navigationLines[len(lines)] = stepItem
			lines = append(lines, m.renderStepLine(node, stepIdx, chapterReviewReady, stepSelected))
			if m.expanded && stepSelected && !m.useSplitWorkspace() {
				if detail := m.renderDetail(); detail != "" {
					lines = append(lines, detail)
				}
			}
			stepIdx++
		}
	}

	return renderedOutline{
		content:        strings.Join(lines, "\n"),
		navigationLine: navigationLines,
	}
}

func (m Model) renderChapterLine(ch coop.SessionChapter, chapterIndex int, selected bool) string {
	prefix := "  "
	if selected {
		prefix = m.theme.BrandStyle.Render(cursorMarker)
	}
	disclosure := "- "
	if m.chapterCollapsed(chapterIndex) {
		disclosure = "+ "
	}
	title := ch.Title
	if selected {
		title = lipgloss.NewStyle().Bold(true).Render(title)
	}
	line := prefix + m.theme.MutedStyle.Render(disclosure) + m.theme.ChapterTitleStyle.Render(title)
	if m.chapterReviewCount(chapterIndex) > 0 {
		line += "  " + m.theme.ReviewStyle.Render("Awaiting review")
	}
	if m.chapterCollapsed(chapterIndex) {
		if summary := m.collapsedChapterSummary(chapterIndex); summary != "" {
			candidate := line + "  " + m.theme.MutedStyle.Render(summary)
			if lipgloss.Width(candidate) <= m.contentWidth() {
				line = candidate
			}
		}
	}
	return line
}

func (m Model) outlineRuleWidth() int {
	w := m.contentWidth() - rowCursorWidth - rowRightGap
	if w < 20 {
		return 20
	}
	if w > maxRuleWidth {
		return maxRuleWidth
	}
	return w
}

func (m Model) collapsedChapterSummary(chapterIndex int) string {
	if m.session == nil || chapterIndex < 0 || chapterIndex >= len(m.session.Chapters) {
		return ""
	}
	var done, review, active, pending, skipped int
	for _, node := range m.session.Chapters[chapterIndex].Nodes {
		switch node.State {
		case coop.StepDone:
			done++
		case coop.StepReview:
			review++
		case coop.StepActive:
			active++
		case coop.StepPending:
			pending++
		case coop.StepSkipped:
			skipped++
		}
	}
	var parts []string
	if done > 0 {
		parts = append(parts, fmt.Sprintf("✓%d", done))
	}
	if review > 0 {
		parts = append(parts, fmt.Sprintf("◆%d", review))
	}
	if active > 0 {
		parts = append(parts, fmt.Sprintf("●%d", active))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("○%d", pending))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("–%d", skipped))
	}
	return strings.Join(parts, " ")
}

func formatStepCount(count int) string {
	if count == 1 {
		return "1 step"
	}
	return fmt.Sprintf("%d steps", count)
}

func (m Model) chapterReviewReady(chapterIndex int) bool {
	return m.chapterHasPendingReviewWithNoActiveWork(chapterIndex)
}

func (m Model) chapterReviewCount(chapterIndex int) int {
	if !m.chapterReviewReady(chapterIndex) {
		return 0
	}
	return m.chapterReviewCountRaw(chapterIndex)
}

func (m Model) chapterReviewCountRaw(chapterIndex int) int {
	if m.session == nil || chapterIndex < 0 || chapterIndex >= len(m.session.Chapters) {
		return 0
	}
	count := 0
	for _, node := range m.session.Chapters[chapterIndex].Nodes {
		if node.State == coop.StepReview {
			count++
		}
	}
	return count
}

func (m Model) chapterHasPendingReviewWithNoActiveWork(chapterIndex int) bool {
	if m.session == nil || chapterIndex < 0 || chapterIndex >= len(m.session.Chapters) {
		return false
	}
	hasReview := false
	for _, node := range m.session.Chapters[chapterIndex].Nodes {
		if node.AutoConfirm {
			continue
		}
		switch node.State {
		case coop.StepReview:
			hasReview = true
		case coop.StepDone, coop.StepSkipped:
		default:
			return false
		}
	}
	return hasReview
}

func (m Model) renderStepLine(node coop.SessionNode, idx int, includedInChapterReview bool, selected bool) string {
	icon := m.stepIcon(node)

	cursor := "  "
	if selected {
		cursor = m.theme.BrandStyle.Render(cursorMarker)
	}

	title := node.Title
	if node.State == coop.StepSkipped {
		title = m.theme.DimmedStyle.Render(title)
	} else if selected {
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
		annStyle = func(s string) string { return m.theme.FileAnnotationStyle.Render(s) }
	case node.State == coop.StepReview && !includedInChapterReview:
		annText = "Waiting for you to review"
		annStyle = func(s string) string { return m.theme.AttentionStyle.Render(s) }
	case node.State == coop.StepActive && node.Activity != "":
		elapsed := ""
		if node.StartedAt != nil {
			dur := time.Since(*node.StartedAt).Truncate(time.Second)
			if dur >= time.Second {
				elapsed = " [" + formatDuration(dur) + "]"
			}
		}
		annText = "Agent working: " + node.Activity + elapsed
		annStyle = func(s string) string { return m.theme.DimmedStyle.Render(s) }
	case node.State == coop.StepSkipped && node.Activity != "":
		annText = "— " + node.Activity
		annStyle = func(s string) string { return m.theme.DimmedStyle.Render(s) }
	}

	line := fmt.Sprintf("%s%s %s", cursor, icon, title)
	if label, style := m.stepStatusLabel(node, includedInChapterReview); label != "" {
		line += "  " + style(label)
	}

	if annText != "" {
		wrapW := m.contentWidth() - 8
		if wrapW < 20 {
			wrapW = 20
		}
		wrapped := wordWrap(annText, wrapW)
		for _, wl := range strings.Split(wrapped, "\n") {
			line += "\n" + strings.Repeat(" ", rowCursorWidth+2) + annStyle(wl)
		}
	}

	return line
}

func (m Model) stepStatusLabel(node coop.SessionNode, includedInChapterReview bool) (string, func(string) string) {
	switch node.State {
	case coop.StepDone:
		return "Done", func(s string) string { return m.theme.SuccessStyle.Render(s) }
	case coop.StepActive:
		return "Agent working", func(s string) string { return m.theme.MutedStyle.Render(s) }
	case coop.StepReview:
		if includedInChapterReview {
			return "Included", func(s string) string { return m.theme.MutedStyle.Render(s) }
		}
		return "Needs review", func(s string) string { return m.theme.AttentionStyle.Render(s) }
	case coop.StepSkipped:
		return "Skipped", func(s string) string { return m.theme.DimmedStyle.Render(s) }
	case coop.StepPending:
		return "Pending", func(s string) string { return m.theme.MutedStyle.Render(s) }
	default:
		return "", func(s string) string { return s }
	}
}

func (m Model) stepIcon(node coop.SessionNode) string {
	// All icons rendered at fixed 1-cell width for alignment
	switch node.State {
	case coop.StepDone:
		return m.theme.SuccessStyle.Render("✓")
	case coop.StepActive:
		return lipgloss.NewStyle().Width(1).Render(m.spinner.View())
	case coop.StepReview:
		return m.theme.AttentionStyle.Render("◆")
	case coop.StepSkipped:
		return m.theme.DimmedStyle.Render("–")
	default:
		return m.theme.MutedStyle.Render("○")
	}
}

func (m Model) renderDetail() string {
	if m.session == nil {
		return ""
	}
	if m.selected.kind == navigationChapter {
		return m.renderChapterDetail(m.selected.chapterIndex)
	}
	stepIndex, ok := m.selectedStepIndex()
	if !ok {
		return ""
	}
	node, err := m.session.NodeByNumber(stepIndex + 1)
	if err != nil {
		return ""
	}

	w, innerW := m.detailWidths()

	var md strings.Builder
	section := detailSections[m.detailTab%len(detailSections)]
	currentSnippet := m.sdkSnippetStep == stepIndex && m.sdkSnippet != ""

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

	if node.State == coop.StepSkipped && node.Activity != "" {
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

func (m Model) renderChapterDetail(chapterIndex int) string {
	if m.session == nil || chapterIndex < 0 || chapterIndex >= len(m.session.Chapters) {
		return ""
	}
	w, innerW := m.detailWidths()

	var md strings.Builder
	section := detailSections[m.detailTab%len(detailSections)]
	ch := &m.session.Chapters[chapterIndex]
	switch section {
	case "Summary":
		m.writeChapterSummaryDetail(&md, ch, innerW)
	case "Files":
		m.writeChapterFilesDetail(&md, ch)
	case "Checks":
		m.writeChapterChecksDetail(&md, ch)
	case "Reference":
		m.writeChapterReferenceDetail(&md, ch)
	}

	content := strings.TrimSpace(md.String())
	suffix := ""
	if target, ok := m.selectedReviewTarget(); ok && target.kind == "chapter" {
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
		Foreground(m.theme.HuePurple400).
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
	if stepIndex, ok := m.selectedStepIndex(); ok && m.sdkLoading && m.sdkLoadingStep == stepIndex {
		md.WriteString("*Loading SDK example...*\n\n")
	}
}

func (m Model) writeChapterSummaryDetail(md *strings.Builder, ch *coop.SessionChapter, width int) {
	wrapWidth := width - 2
	if wrapWidth < 20 {
		wrapWidth = 20
	}
	md.WriteString("Steps\n")
	for _, node := range ch.Nodes {
		md.WriteString("  " + chapterStepStatusLabel(node) + " " + node.Title + "\n")
	}
	md.WriteString("\n")
	if changed := chapterChangedFiles(ch); changed != "" {
		md.WriteString("Changed\n")
		for _, line := range strings.Split(wordWrap(changed, wrapWidth), "\n") {
			md.WriteString("  " + line + "\n")
		}
		md.WriteString("\n")
	}
	if checks := chapterConfirmationSteps(ch); checks != "" {
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

func (m Model) writeChapterFilesDetail(md *strings.Builder, ch *coop.SessionChapter) {
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
	md.WriteString("*No files reported for this section yet.*\n\n")
}

func (m Model) writeChapterChecksDetail(md *strings.Builder, ch *coop.SessionChapter) {
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
	md.WriteString("*No confirmation steps reported for this section yet.*\n\n")
}

func (m Model) writeChapterReferenceDetail(md *strings.Builder, ch *coop.SessionChapter) {
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
	md.WriteString("*No reference metadata for this section yet.*\n\n")
}

func chapterStepStatusLabel(node coop.SessionNode) string {
	switch node.State {
	case coop.StepDone:
		return "✓"
	case coop.StepActive:
		return "●"
	case coop.StepReview:
		return "◆"
	case coop.StepSkipped:
		return "–"
	default:
		return "○"
	}
}

func chapterChangedFiles(ch *coop.SessionChapter) string {
	var files []string
	for _, node := range ch.Nodes {
		if node.Implementation != nil && node.Implementation.File != "" {
			files = append(files, implementationFileLabel(node.Implementation))
		}
	}
	return strings.Join(files, ", ")
}

func chapterConfirmationSteps(ch *coop.SessionChapter) string {
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
	if stepIndex, ok := m.selectedStepIndex(); ok && m.sdkLoading && m.sdkLoadingStep == stepIndex {
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
	if target, ok := m.selectedReviewTarget(); ok && target.kind == "step" && node.State == coop.StepReview {
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
