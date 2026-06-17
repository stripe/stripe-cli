package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/coop"
)

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
