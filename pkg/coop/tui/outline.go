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
		if m.selected.kind == navigationStep {
			return m.theme.MutedStyle.Render("Press enter to inspect this step.")
		}
		return m.theme.MutedStyle.Render("Press enter to inspect this node.")
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
	nodeIdx := 0

	ruleWidth := m.outlineRuleWidth()

	for stepIdx, ch := range m.session.Steps {
		stepItem := navigationItem{kind: navigationStep, stepIndex: stepIdx}
		stepSelected := m.navigationItemSelected(stepItem)
		stepReviewReady := m.stepReviewReady(stepIdx)
		lines = append(lines, "")
		navigationLines[len(lines)] = stepItem
		lines = append(lines, m.renderStepLine(ch, stepIdx, stepSelected))
		lines = append(lines, strings.Repeat(" ", rowCursorWidth)+m.theme.StepRuleStyle.Render(strings.Repeat("─", ruleWidth)))
		if m.expanded && stepSelected && !m.useSplitWorkspace() {
			if detail := m.renderDetail(); detail != "" {
				lines = append(lines, detail)
			}
		}

		if m.stepCollapsed(stepIdx) {
			nodeIdx += len(ch.Nodes)
			continue
		}
		for _, node := range ch.Nodes {
			nodeItem := navigationItem{kind: navigationNode, nodeIndex: nodeIdx, stepIndex: stepIdx}
			nodeSelected := m.navigationItemSelected(nodeItem)
			navigationLines[len(lines)] = nodeItem
			lines = append(lines, m.renderNodeLine(node, nodeIdx, stepReviewReady, nodeSelected))
			if m.expanded && nodeSelected && !m.useSplitWorkspace() {
				if detail := m.renderDetail(); detail != "" {
					lines = append(lines, detail)
				}
			}
			nodeIdx++
		}
	}

	return renderedOutline{
		content:        strings.Join(lines, "\n"),
		navigationLine: navigationLines,
	}
}

func (m Model) renderStepLine(ch coop.SessionStep, stepIndex int, selected bool) string {
	prefix := "  "
	if selected {
		prefix = m.theme.BrandStyle.Render(cursorMarker)
	}
	disclosure := "- "
	if m.stepCollapsed(stepIndex) {
		disclosure = "+ "
	}
	title := ch.Title
	if selected {
		title = lipgloss.NewStyle().Bold(true).Render(title)
	}
	line := prefix + m.theme.MutedStyle.Render(disclosure) + m.theme.StepTitleStyle.Render(title)
	if m.stepReviewCount(stepIndex) > 0 {
		line += "  " + m.theme.ReviewStyle.Render("Awaiting review")
	}
	if m.stepCollapsed(stepIndex) {
		if summary := m.collapsedStepSummary(stepIndex); summary != "" {
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

func (m Model) collapsedStepSummary(stepIndex int) string {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return ""
	}
	var done, review, active, pending, skipped int
	for _, node := range m.session.Steps[stepIndex].Nodes {
		switch node.State {
		case coop.NodeDone:
			done++
		case coop.NodeReview:
			review++
		case coop.NodeActive:
			active++
		case coop.NodePending:
			pending++
		case coop.NodeSkipped:
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

func (m Model) stepReviewReady(stepIndex int) bool {
	return m.stepHasPendingReviewWithNoActiveWork(stepIndex)
}

func (m Model) stepReviewCount(stepIndex int) int {
	if !m.stepReviewReady(stepIndex) {
		return 0
	}
	return m.stepReviewCountRaw(stepIndex)
}

func (m Model) stepReviewCountRaw(stepIndex int) int {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return 0
	}
	count := 0
	for _, node := range m.session.Steps[stepIndex].Nodes {
		if node.State == coop.NodeReview {
			count++
		}
	}
	return count
}

func (m Model) stepHasPendingReviewWithNoActiveWork(stepIndex int) bool {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return false
	}
	hasReview := false
	for _, node := range m.session.Steps[stepIndex].Nodes {
		if node.AutoConfirm {
			continue
		}
		switch node.State {
		case coop.NodeReview:
			hasReview = true
		case coop.NodeDone, coop.NodeSkipped:
		default:
			return false
		}
	}
	return hasReview
}

func (m Model) renderNodeLine(node coop.SessionNode, idx int, includedInStepReview bool, selected bool) string {
	icon := m.nodeIcon(node)

	cursor := "  "
	if selected {
		cursor = m.theme.BrandStyle.Render(cursorMarker)
	}

	title := node.Title
	if node.State == coop.NodeSkipped {
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
	case node.State == coop.NodeReview && !includedInStepReview:
		annText = "Waiting for you to review"
		annStyle = func(s string) string { return m.theme.AttentionStyle.Render(s) }
	case node.State == coop.NodeActive && node.Activity != "":
		elapsed := ""
		if node.StartedAt != nil {
			dur := time.Since(*node.StartedAt).Truncate(time.Second)
			if dur >= time.Second {
				elapsed = " [" + formatDuration(dur) + "]"
			}
		}
		annText = "Agent working: " + node.Activity + elapsed
		annStyle = func(s string) string { return m.theme.DimmedStyle.Render(s) }
	case node.State == coop.NodeSkipped && node.Activity != "":
		annText = "— " + node.Activity
		annStyle = func(s string) string { return m.theme.DimmedStyle.Render(s) }
	}

	line := fmt.Sprintf("%s%s %s", cursor, icon, title)
	if label, style := m.nodeStatusLabel(node, includedInStepReview); label != "" {
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

func (m Model) nodeStatusLabel(node coop.SessionNode, includedInStepReview bool) (string, func(string) string) {
	switch node.State {
	case coop.NodeDone:
		return "Done", func(s string) string { return m.theme.SuccessStyle.Render(s) }
	case coop.NodeActive:
		return "Agent working", func(s string) string { return m.theme.MutedStyle.Render(s) }
	case coop.NodeReview:
		if includedInStepReview {
			return "Included", func(s string) string { return m.theme.MutedStyle.Render(s) }
		}
		return "Needs review", func(s string) string { return m.theme.AttentionStyle.Render(s) }
	case coop.NodeSkipped:
		return "Skipped", func(s string) string { return m.theme.DimmedStyle.Render(s) }
	case coop.NodePending:
		return "Pending", func(s string) string { return m.theme.MutedStyle.Render(s) }
	default:
		return "", func(s string) string { return s }
	}
}

func (m Model) nodeIcon(node coop.SessionNode) string {
	switch node.State {
	case coop.NodeDone:
		return m.theme.SuccessStyle.Render("✓")
	case coop.NodeActive:
		return lipgloss.NewStyle().Width(1).Render(m.spinner.View())
	case coop.NodeReview:
		return m.theme.AttentionStyle.Render("◆")
	case coop.NodeSkipped:
		return m.theme.DimmedStyle.Render("–")
	default:
		return m.theme.MutedStyle.Render("○")
	}
}
