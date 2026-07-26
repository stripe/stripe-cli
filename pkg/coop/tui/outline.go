package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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

// outlineWidth returns the column width the step outline should render into.
// In the split workspace this is the narrow left column; otherwise the full
// content width.
func (m Model) outlineWidth() int {
	if m.outlineWidthOverride > 0 {
		return m.outlineWidthOverride
	}
	return m.contentWidth()
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

	// Constrain the outline's rules and wrapping to the left column so dividers
	// and long titles don't overflow and wrap into a broken multi-line mess.
	// Scoped to the outline copy only — the right detail panel keeps full width.
	navModel := m
	navModel.outlineWidthOverride = leftW
	nav := navModel.renderStepOutline().content
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
	// Trim blank lines, not whitespace: renderDetail indents every line, and
	// TrimSpace would strip that indent from the first line only — leaving the
	// box's top border two columns left of its own sides.
	detail := strings.Trim(m.renderDetail(), "\n")
	if detail == "" {
		return m.theme.MutedStyle.Render("No details available yet.")
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(detail)
}

type renderedOutline struct {
	content        string
	navigationLine map[int]navigationItem
}

// stepShowsInlineCard reports whether this step draws its full card in place.
//
// The step in play always does: its card is the thing the user is here to read,
// and it belongs under the step, not at the bottom of the screen. Any other
// step draws one only when the user opens it with enter.
func (m Model) stepShowsInlineCard(stepIndex int) bool {
	if m.useSplitWorkspace() {
		return false
	}
	if selected, ok := m.selectedStepIndex(); !ok || selected != stepIndex {
		return false
	}
	// Below this the card cannot show enough of itself to be worth the space:
	// it would sit entirely under the fold, and the user would see a step line
	// and nothing to act on. The footer carries a one-line fallback instead.
	//
	// Measured against the terminal, not the viewport: the viewport's height is
	// derived from the footer, and the footer asks this question, so reading the
	// viewport here would make the two definitions circular.
	// While the user is typing feedback the card is the editor's only home, so
	// it renders in place at any height; the body above the input is what gives
	// way instead.
	if m.height > 0 && m.height < minInlineCardRows && !m.rejecting {
		return false
	}
	if _, hasReview := m.selectedReviewTarget(); hasReview {
		return true
	}
	return m.expanded
}

// minInlineCardRows is the terminal height below which the card stops rendering
// in place: header, step line and rule, the card's own frame, a few rows of
// content, and the footer.
const minInlineCardRows = 18

func (m Model) renderStepOutline() renderedOutline {
	if m.session == nil {
		return renderedOutline{navigationLine: map[int]navigationItem{}}
	}

	var lines []string
	navigationLines := map[int]navigationItem{}
	nodeIdx := 0

	ruleWidth := m.outlineRuleWidth()

	cramped := m.outlineIsCramped()
	first, last, above, below := m.outlineWindow()
	if above > 0 {
		lines = append(lines, m.theme.MutedStyle.Render(
			fmt.Sprintf("  ▲ %d more above", above)))
	}

	for stepIdx, ch := range m.session.Steps {
		if stepIdx < first || stepIdx > last {
			nodeIdx += len(ch.Nodes)
			continue
		}
		stepItem := navigationItem{kind: navigationStep, stepIndex: stepIdx}
		stepSelected := m.navigationItemSelected(stepItem)
		// In a cramped list the neighbors are there for orientation only, so
		// they give up their separator and rule; the selected step keeps both.
		compact := cramped && !stepSelected
		if !compact {
			lines = append(lines, "")
		}
		navigationLines[len(lines)] = stepItem
		lines = append(lines, m.renderStepLine(ch, stepIdx, stepSelected))
		if !compact {
			lines = append(lines, strings.Repeat(" ", rowCursorWidth)+m.theme.StepRuleStyle.Render(strings.Repeat("─", ruleWidth)))
		}
		// The step's card renders here, directly under the step it belongs to.
		// It used to live pinned at the bottom of the screen, with `enter`
		// toggling between that and an inline copy — two presentations of one
		// step, in two places, and shrinking the terminal moved the content
		// away from the step it described.
		if m.stepShowsInlineCard(stepIdx) {
			if detail := m.renderDetail(); detail != "" {
				lines = append(lines, detail)
			}
			// The card lists the tasks itself, so listing them again out here
			// would show each one twice.
			nodeIdx += len(ch.Nodes)
			continue
		}

		if m.stepCollapsed(stepIdx) {
			nodeIdx += len(ch.Nodes)
			continue
		}
		for _, node := range ch.Nodes {
			nodeItem := navigationItem{kind: navigationNode, nodeIndex: nodeIdx, stepIndex: stepIdx}
			nodeSelected := m.navigationItemSelected(nodeItem)
			navigationLines[len(lines)] = nodeItem
			lines = append(lines, m.renderNodeLine(node, nodeIdx, nodeSelected))
			if m.expanded && nodeSelected && !m.useSplitWorkspace() {
				if detail := m.renderDetail(); detail != "" {
					lines = append(lines, detail)
				}
			}
			nodeIdx++
		}
	}

	if below > 0 {
		lines = append(lines, m.theme.MutedStyle.Render(
			fmt.Sprintf("  ▼ %d more below", below)))
	}

	return renderedOutline{
		content:        strings.Join(lines, "\n"),
		navigationLine: navigationLines,
	}
}

// outlineWindow decides which steps to draw.
//
// With room, every step is listed. When the viewport is short the detail view
// is what the user is actually reading, so the list gives up its rows and keeps
// only the selected step with one either side — enough to know where you are —
// plus counts of what was hidden, so the list never silently looks complete.
func (m Model) outlineWindow() (first, last, above, below int) {
	total := len(m.session.Steps)
	if total == 0 {
		return 0, -1, 0, 0
	}
	last = total - 1
	if !m.outlineIsCramped() {
		return 0, last, 0, 0
	}

	selected := 0
	if idx, ok := m.selectedStepIndex(); ok {
		selected = idx
	}
	first = max(selected-1, 0)
	last = min(selected+1, total-1)
	return first, last, first, total - 1 - last
}

// outlineIsCramped reports whether the full list would not fit. Each step costs
// a blank line, its title and a rule, and the step in play also lists its tasks.
func (m Model) outlineIsCramped() bool {
	height := m.viewport.Height()
	if height <= 0 {
		return false
	}
	rows := 0
	for stepIdx, step := range m.session.Steps {
		rows += 3
		if !m.stepCollapsed(stepIdx) {
			rows += len(step.Nodes)
		}
	}
	// Tolerance, so a list that only just overflows still renders in full and
	// scrolls. Windowing is a real loss of context; it should take a clear
	// shortfall to trigger, not a single row.
	const tolerance = 4
	return rows > height+tolerance
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
	switch {
	case m.stepJustConfirmed(stepIndex):
		line += "  " + m.theme.SuccessStyle.Render("✓ confirmed")
	case m.stepReviewCount(stepIndex) > 0:
		line += "  " + m.theme.SoftAttentionStyle.Render("needs you")
	}
	// A failed check is carried on the step line, not only inside the card.
	// The card scrolls; this row is what the user sees while moving through the
	// blueprint, and a failure they have to scroll to find is one they miss.
	if failed, _ := stepCheckResults(&ch); len(failed) > 0 {
		line += "  " + m.theme.SoftErrorStyle.Render(fmt.Sprintf("✗%d", len(failed)))
	}
	if m.stepCollapsed(stepIndex) {
		if summary := m.collapsedStepSummary(stepIndex); summary != "" {
			candidate := line + "  " + m.theme.MutedStyle.Render(summary)
			if lipgloss.Width(candidate) <= m.outlineWidth() {
				line = candidate
			}
		}
	}
	// Truncate rather than let a long title run off: every other truncation in
	// the UI marks itself, and this one did not.
	if width := m.outlineWidth(); width > 0 && lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "…")
	}
	return line
}

func (m Model) outlineRuleWidth() int {
	w := m.outlineWidth() - rowCursorWidth - rowRightGap
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

func (m Model) renderNodeLine(node coop.SessionNode, idx int, selected bool) string {
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
	if label, style := m.nodeStatusLabel(node); label != "" {
		line += "  " + style(label)
	}

	if annText != "" {
		wrapW := m.outlineWidth() - 8
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

func (m Model) nodeStatusLabel(node coop.SessionNode) (string, func(string) string) {
	switch node.State {
	case coop.NodeDone:
		return "done", func(s string) string { return m.theme.SuccessStyle.Render(s) }
	case coop.NodeActive:
		return "working", func(s string) string { return m.theme.MutedStyle.Render(s) }
	case coop.NodeReview:
		// A task is never reviewed on its own, so a finished task is simply
		// ready; the step above it is what the user acts on.
		return "ready", func(s string) string { return m.theme.MutedStyle.Render(s) }
	case coop.NodeSkipped:
		return "skipped", func(s string) string { return m.theme.DimmedStyle.Render(s) }
	case coop.NodePending:
		return "pending", func(s string) string { return m.theme.MutedStyle.Render(s) }
	default:
		return "", func(s string) string { return s }
	}
}

func (m Model) nodeIcon(node coop.SessionNode) string {
	switch node.State {
	case coop.NodeDone:
		return m.theme.SuccessStyle.Render("✓")
	case coop.NodeActive:
		// Width sets a minimum, not a maximum, so a spinner that renders wider
		// than one cell — an uninitialised one yields "(error)" — would push
		// everything after it out of alignment. Bound it to a single cell.
		return lipgloss.NewStyle().Width(1).MaxWidth(1).Render(m.spinner.View())
	case coop.NodeReview:
		// Muted, not attention-colored: the task is finished and waiting on the
		// step, and an orange marker on every row made routine progress look
		// like a problem.
		return m.theme.MutedStyle.Render("◆")
	case coop.NodeSkipped:
		return m.theme.DimmedStyle.Render("–")
	default:
		return m.theme.MutedStyle.Render("○")
	}
}
