package tui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/glamour/v2"
	glamouransi "charm.land/glamour/v2/ansi"
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

func (m Model) renderMarkdown(content string, width int) string {
	if content == "" {
		return ""
	}
	renderer, err := markdownRenderer(width, m.isDark)
	if err != nil {
		return content
	}
	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimSpace(rendered)
}

type markdownRendererKey struct {
	width int
	dark  bool
	style string
}

var markdownRenderers = struct {
	sync.Mutex
	byKey map[markdownRendererKey]*glamour.TermRenderer
}{byKey: map[markdownRendererKey]*glamour.TermRenderer{}}

func markdownRenderer(width int, isDark bool) (*glamour.TermRenderer, error) {
	if width < 1 {
		width = 1
	}
	style := os.Getenv("GLAMOUR_STYLE")
	key := markdownRendererKey{width: width, dark: isDark, style: style}

	markdownRenderers.Lock()
	defer markdownRenderers.Unlock()
	if renderer := markdownRenderers.byKey[key]; renderer != nil {
		return renderer, nil
	}

	var styleOpt glamour.TermRendererOption
	if style != "" {
		styleOpt = glamour.WithEnvironmentConfig()
	} else {
		styleOpt = glamour.WithStyles(compactMarkdownStyle(isDark))
	}
	renderer, err := glamour.NewTermRenderer(
		styleOpt,
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
		glamour.WithPreservedNewLines(),
		glamour.WithTableWrap(false),
	)
	if err != nil {
		return nil, err
	}
	markdownRenderers.byKey[key] = renderer
	return renderer, nil
}

func compactMarkdownStyle(isDark bool) glamouransi.StyleConfig {
	text := "252"
	muted := "245"
	accent := "141"
	rule := "240"
	codeBG := "236"
	codeTheme := "monokai"
	if !isDark {
		text = "236"
		muted = "244"
		accent = "63"
		rule = "250"
		codeBG = "255"
		codeTheme = "github"
	}

	return glamouransi.StyleConfig{
		Document: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: stringPtr(text),
			},
			Margin: uintPtr(0),
		},
		Heading: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: stringPtr(accent),
				Bold:  boolPtr(true),
			},
		},
		List: glamouransi.StyleList{
			StyleBlock: glamouransi.StyleBlock{
				Margin: uintPtr(0),
			},
			LevelIndent: 2,
		},
		BlockQuote: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: stringPtr(muted),
			},
			Indent:      uintPtr(1),
			IndentToken: stringPtr("│ "),
			Margin:      uintPtr(0),
		},
		Strong: glamouransi.StylePrimitive{
			Bold: boolPtr(true),
		},
		Emph: glamouransi.StylePrimitive{
			Italic: boolPtr(true),
			Color:  stringPtr(muted),
		},
		HorizontalRule: glamouransi.StylePrimitive{
			Color:  stringPtr(rule),
			Format: "\n--------\n",
		},
		Item: glamouransi.StylePrimitive{
			BlockPrefix: "• ",
		},
		Enumeration: glamouransi.StylePrimitive{
			BlockPrefix: ". ",
		},
		Task: glamouransi.StyleTask{
			Ticked:   "[✓] ",
			Unticked: "[ ] ",
		},
		Link: glamouransi.StylePrimitive{
			Color:     stringPtr(accent),
			Underline: boolPtr(true),
		},
		LinkText: glamouransi.StylePrimitive{
			Color: stringPtr(accent),
			Bold:  boolPtr(true),
		},
		Code: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color:           stringPtr(text),
				BackgroundColor: stringPtr(codeBG),
			},
		},
		CodeBlock: glamouransi.StyleCodeBlock{
			StyleBlock: glamouransi.StyleBlock{
				StylePrimitive: glamouransi.StylePrimitive{
					Color: stringPtr(text),
				},
				Margin: uintPtr(0),
			},
			Theme: codeTheme,
		},
		Table: glamouransi.StyleTable{
			StyleBlock: glamouransi.StyleBlock{
				Margin: uintPtr(0),
			},
			CenterSeparator: stringPtr("|"),
			ColumnSeparator: stringPtr("|"),
			RowSeparator:    stringPtr("-"),
		},
	}
}

func stringPtr(v string) *string {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func uintPtr(v uint) *uint {
	return &v
}

func (m Model) renderFooter() string {
	// Completion view has its own footer — don't render step footer
	if m.session != nil && m.session.IsComplete() {
		return ""
	}

	var lines []string

	// Agent disconnected warning
	if m.agentIdle() {
		lines = append(lines, m.theme.AttentionStyle.Render("  Waiting for agent: no recent updates. Reconnect: stripe coop status"))
	}

	if m.statusMessage != "" {
		lines = append(lines, m.theme.AttentionStyle.Render("  "+m.statusMessage))
	}

	if m.session != nil {
		if count := m.actionableReviewCount(); count > 0 {
			lines = append(lines, "")
			lines = append(lines, m.theme.AttentionStyle.Render("  Waiting for you: review section"))
		}
	}

	h := m.help
	h.SetWidth(m.width - 2)
	h.ShortSeparator = " · "
	actionLine := m.theme.FooterStyle.MaxWidth(m.width).Render("  " + h.View(m))

	if _, ok := m.selectedReviewTarget(); ok && !m.expanded {
		budget := m.footerHeightBudget()
		cardGapH := 1
		actionH := lipgloss.Height(actionLine)
		prefixH := lipgloss.Height(strings.Join(lines, "\n"))
		cardMaxHeight := budget - prefixH - cardGapH - actionH
		card := m.renderReviewCardWithMaxHeight(cardMaxHeight)
		if card != "" {
			result := append(append([]string{}, lines...), card, "", actionLine)
			if footerLinesFit(result, budget) {
				return strings.Join(result, "\n")
			}
		}

		cardMaxHeight = budget - cardGapH - actionH
		card = m.renderReviewCardWithMaxHeight(cardMaxHeight)
		if card != "" {
			return strings.Join([]string{card, "", actionLine}, "\n")
		}
	}

	lines = append(lines, actionLine)
	if budget := m.footerHeightBudget(); budget > 0 && lipgloss.Height(strings.Join(lines, "\n")) > budget {
		lines = append(lines[:max(len(lines)-2, 0)], actionLine)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderReviewCard() string {
	return m.renderReviewCardWithMaxHeight(0)
}

func (m Model) renderReviewCardWithMaxHeight(maxHeight int) string {
	target, ok := m.selectedReviewTarget()
	if !ok {
		return ""
	}
	if maxHeight > 0 && maxHeight < 3 {
		return ""
	}
	w, _ := m.reviewCardWidths()

	var lines []string
	prefix := "Review"
	if target.kind == "chapter" {
		prefix = "Review section"
	}
	lines = append(lines, m.theme.ReviewStyle.Render(prefix))
	check := m.reviewPromptLabel(target.steps)
	if check != "" {
		lines = append(lines, m.theme.ConfirmationHeaderStyle.Render("Confirmation steps"))
		lines = append(lines, check)
	}
	metadataStart := len(lines)
	if target.kind == "chapter" {
		if included := m.reviewStepTitleLabel(target.steps); included != "" {
			lines = append(lines, m.theme.MutedStyle.Render("Includes: ")+included)
		}
	}
	if changed := m.reviewChangedLabel(target.steps); changed != "" {
		lines = append(lines, m.theme.MutedStyle.Render("Agent changed: ")+changed)
	}
	if verified := m.reviewVerificationLabel(target.steps); verified != "" {
		lines = append(lines, m.theme.MutedStyle.Render("Agent verified: ")+verified)
	}
	if command := m.reviewCommandLabel(target.steps); command != "" {
		lines = append(lines, m.theme.MutedStyle.Render("Run: ")+command)
	}
	if len(lines) > metadataStart && check != "" {
		lines = append(lines[:metadataStart], append([]string{""}, lines[metadataStart:]...)...)
	}
	if m.rejecting {
		m.rejectionInput.SetWidth(m.requestChangesInputWidth())
		inputView := m.rejectionInput.View()
		if m.rejectionInput.Value() == "" {
			inputView = m.theme.DimmedStyle.Render(m.rejectionInput.Placeholder)
		}
		lines = append(lines, m.theme.ErrorStyle.Render("Request changes: ")+inputView)
		if m.rejectionError != "" {
			lines = append(lines, m.theme.ErrorStyle.Render(m.rejectionError))
		}
	}

	var wrapped []string
	for _, line := range lines {
		for _, segment := range strings.Split(line, "\n") {
			wrapped = append(wrapped, strings.Split(wordWrap(segment, w-4), "\n")...)
		}
	}
	if maxHeight > 0 {
		maxContentLines := maxHeight - 2
		if len(wrapped) > maxContentLines {
			if maxContentLines <= 1 {
				wrapped = []string{m.theme.DimmedStyle.Render("Review: more checks available")}
			} else {
				more := m.theme.DimmedStyle.Render("Confirmation steps: enter/e for more")
				wrapped = append(wrapped[:maxContentLines-1], more)
			}
		}
	}
	return m.renderReviewCardLines(w, maxHeight, wrapped)
}

func footerLinesFit(lines []string, budget int) bool {
	return budget <= 0 || lipgloss.Height(strings.Join(lines, "\n")) <= budget
}

func (m Model) reviewCardWidths() (int, int) {
	w := min(m.contentWidth()-2, 84)
	if w < 20 {
		w = m.contentWidth() - 2
	}
	frameW, _ := m.theme.ReviewCardStyle.GetFrameSize()
	innerW := w - frameW
	if innerW < 8 {
		innerW = 8
	}
	return w, innerW
}

func (m Model) requestChangesInputWidth() int {
	_, innerW := m.reviewCardWidths()
	width := innerW - lipgloss.Width("Request changes: ")
	if width < 8 {
		return 8
	}
	return width
}

func (m Model) renderReviewCardLines(width, maxHeight int, lines []string) string {
	more := m.theme.DimmedStyle.Render("Review: more checks available")
	style := m.theme.ReviewCardStyle.Width(width).MaxWidth(width + 4)
	for {
		rendered := style.Render(strings.Join(lines, "\n"))
		if maxHeight <= 0 || lipgloss.Height(rendered) <= maxHeight {
			return rendered
		}
		if len(lines) <= 2 {
			return style.MaxHeight(maxHeight).Render(strings.Join(lines, "\n"))
		}
		lines = append(lines[:len(lines)-2], more)
	}
}

func (m Model) renderViewportRegion() string {
	return m.renderViewportRegionWithHeight(m.viewport.Height())
}

func (m Model) renderViewportRegionWithHeight(height int) string {
	if m.width <= 0 || height <= 0 {
		return m.viewport.View()
	}
	hasMoreBelow := m.viewport.YOffset()+height < m.viewport.TotalLineCount()
	if hasMoreBelow && height >= 3 {
		vp := m.viewport
		vp.SetHeight(height - 2)
		body := lipgloss.NewStyle().
			Width(m.width).
			Height(height - 2).
			MaxHeight(height - 2).
			Render(vp.View())
		body = closeOpenBoxAtViewportBoundary(body)
		indicator := m.renderMoreBelowIndicator()
		return strings.Join([]string{body, "", indicator}, "\n")
	}
	view := m.viewport.View()
	rendered := lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		MaxHeight(height).
		Render(view)
	return rendered
}

func (m Model) renderMoreBelowIndicator() string {
	label := m.theme.MutedStyle.Render("more below")
	width := m.outlineRuleWidth()
	if width < lipgloss.Width(label) {
		width = lipgloss.Width(label)
	}
	centered := lipgloss.PlaceHorizontal(width, lipgloss.Center, label)
	return lipgloss.NewStyle().
		Width(m.width).
		MaxWidth(m.width).
		Render(strings.Repeat(" ", rowCursorWidth) + centered)
}

func closeOpenBoxAtViewportBoundary(s string) string {
	if !strings.Contains(s, "╭") || strings.Contains(s, "╰") {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}
	topLine := -1
	bottomLine := -1
	for i, line := range lines {
		if strings.Contains(line, "╭") && strings.Contains(line, "╮") {
			topLine = i
		}
		if strings.Contains(line, "╰") && strings.Contains(line, "╯") {
			bottomLine = i
		}
	}
	if topLine == -1 || bottomLine > topLine || topLine >= len(lines)-1 {
		return s
	}
	top := lines[topLine]
	bottom := strings.NewReplacer("╭", "╰", "╮", "╯").Replace(top)
	lines[len(lines)-1] = bottom
	return strings.Join(lines, "\n")
}

func (m Model) renderPinnedViewport(header, footer string) string {
	footerGap := 2
	viewHeight := m.viewport.Height()
	if m.height > 0 {
		headerH := lipgloss.Height(header) + 1
		footerH := lipgloss.Height(footer)
		available := m.height - headerH - footerH - footerGap
		if available < minViewportHeight {
			available = minViewportHeight
		}
		if viewHeight <= 0 || viewHeight > available {
			viewHeight = available
		}
	}
	view := m.renderViewportRegionWithHeight(viewHeight)
	rendered := header + "\n" + view + strings.Repeat("\n", footerGap) + footer
	if m.height <= 0 {
		return rendered
	}
	if pad := m.height - lipgloss.Height(rendered); pad > 0 {
		rendered = header + "\n" + view + strings.Repeat("\n", footerGap+pad) + footer
	}
	return rendered
}

func (m Model) footerHeightBudget() int {
	if m.height <= 0 {
		return 0
	}
	headerHeight := lipgloss.Height(m.renderHeader())
	budget := m.height - headerHeight - minViewportHeight - 2 - terminalScrollGuard
	if budget < 1 {
		return 1
	}
	return budget
}

func (m Model) requestChangesPlaceholder(target reviewTarget) string {
	if target.kind == "chapter" {
		return "Describe what should change in this section"
	}
	for _, step := range target.steps {
		node, err := m.session.NodeByNumber(step)
		if err != nil {
			continue
		}
		switch node.Type {
		case coop.NodeAsyncHandler, coop.NodeSetUpWebhooks:
			return "Describe what should change in signature verification or event handling"
		case coop.NodeAPIRequest:
			return "Describe what should change in the API call, IDs, or stored values"
		case coop.NodeUIComponent:
			return "Describe what should change in the user-facing flow"
		case coop.NodeTestHelper:
			return "Describe the failing path or expected result"
		}
	}
	return "Describe what should change"
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

func (m Model) reviewStepTitleLabel(steps []int) string {
	if m.session == nil {
		return ""
	}
	var titles []string
	for _, step := range steps {
		node, err := m.session.NodeByNumber(step)
		if err != nil || node.Title == "" {
			continue
		}
		titles = append(titles, node.Title)
	}
	if len(titles) == 0 {
		return ""
	}
	if len(titles) > 3 {
		return strings.Join(titles[:3], ", ") + fmt.Sprintf(" +%d more", len(titles)-3)
	}
	return strings.Join(titles, ", ")
}

func (m Model) reviewPromptLabel(steps []int) string {
	if agentChecks := m.reviewAgentConfirmationLabel(steps); agentChecks != "" {
		return agentChecks
	}
	if blueprintChecks := m.reviewBlueprintConfirmationLabel(steps); blueprintChecks != "" {
		return blueprintChecks
	}
	return "Confirm the completed work matches this step and its verification evidence."
}

func (m Model) reviewAgentConfirmationLabel(steps []int) string {
	var checks []string
	seen := map[string]bool{}
	showStepTitle := len(steps) > 1
	for _, step := range steps {
		node, err := m.session.NodeByNumber(step)
		if err != nil {
			continue
		}
		for _, verification := range node.Verifications {
			check := strings.TrimSpace(verification.Check)
			if !verification.Passed || check == "" || seen[check] {
				continue
			}
			seen[check] = true
			if showStepTitle && node.Title != "" {
				check = node.Title + ": " + check
			}
			checks = append(checks, check)
		}
	}
	return reviewConfirmationSummary(checks, 3)
}

func (m Model) reviewBlueprintConfirmationLabel(steps []int) string {
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
	return reviewConfirmationSummary(prompts, 2)
}

func reviewConfirmationSummary(checks []string, limit int) string {
	if len(checks) == 0 {
		return ""
	}
	if len(checks) > limit {
		return strings.Join(checks[:limit], "\n") + fmt.Sprintf("\nOpen details for %d more check(s).", len(checks)-limit)
	}
	return strings.Join(checks, "\n")
}

func (m Model) reviewCommandLabel(steps []int) string {
	var commands []string
	seen := map[string]bool{}
	for _, step := range steps {
		node, err := m.session.NodeByNumber(step)
		if err != nil {
			continue
		}
		command := reviewCommandForNode(node)
		if command == "" || seen[command] {
			continue
		}
		seen[command] = true
		commands = append(commands, command)
	}
	if len(commands) == 0 {
		return ""
	}
	if len(commands) > 2 {
		return strings.Join(commands[:2], " && ") + fmt.Sprintf(" && # +%d more", len(commands)-2)
	}
	return strings.Join(commands, " && ")
}

func reviewCommandForNode(node *coop.SessionNode) string {
	if node.ReviewCommand != "" {
		return node.ReviewCommand
	}
	if node.Type == coop.NodeAsyncHandler && len(node.Events) > 0 {
		return "stripe trigger " + node.Events[0]
	}
	return ""
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
