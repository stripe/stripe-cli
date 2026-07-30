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

// clipRenderedBox trims an already-drawn card to maxHeight, keeping its borders
// and leaving a hint where it dropped content.
func (m Model) clipRenderedBox(box string, maxHeight int) string {
	lines := strings.Split(box, "\n")
	if maxHeight <= 0 || len(lines) <= maxHeight {
		return box
	}
	// Top border, at least one row of content, the hint, and the bottom border.
	const minRows = 4
	if maxHeight < minRows {
		return box
	}
	bottom := lines[len(lines)-1]
	hint := lines[1]
	if inner := lipgloss.Width(ansi.Strip(hint)) - 4; inner > 0 {
		hint = replaceBoxRowText(lines[1], m.theme.MutedStyle.Render(cardOverflowHint), inner)
	}
	kept := append([]string{}, lines[:maxHeight-2]...)
	return strings.Join(append(kept, hint, bottom), "\n")
}

// replaceBoxRowText swaps the text inside a rendered card row, keeping the row's
// own borders and padding so the replacement lines up with the rows around it.
func replaceBoxRowText(row, text string, inner int) string {
	plain := ansi.Strip(row)
	left := strings.Index(plain, "│")
	right := strings.LastIndex(plain, "│")
	if left < 0 || right <= left {
		return row
	}
	body := ansi.Truncate(text, inner, "…")
	pad := inner - lipgloss.Width(ansi.Strip(body))
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", left) + boxSide + " " + body + strings.Repeat(" ", pad) + " " + boxSide
}

// boxSide is the vertical border glyph the cards are drawn with.
const boxSide = "│"

func (m Model) renderSplitDetail(width int) string {
	// Trim blank lines, not whitespace: renderDetail indents every line, and
	// TrimSpace would strip that indent from the first line only — leaving the
	// box's top border two columns left of its own sides.
	detail := strings.Trim(m.renderDetail(), "\n")
	if detail == "" {
		return m.theme.MutedStyle.Render("No details available yet.")
	}
	// Bound the pane to the rows it has. The joined columns scroll as one
	// block, so a pane taller than the viewport was cut by the scroll — and a
	// cut that lands inside a card leaves it with no bottom border, which
	// cannot be repaired afterwards without overwriting the other column too.
	detail = m.clipRenderedBox(detail, m.viewport.Height())
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
	// A step the reader just confirmed shows the acknowledgement in its small
	// card instead of the review card it has finished with.
	if m.stepJustConfirmed(stepIndex) {
		return false
	}
	// Arriving on a step opens it. It used to open only when a review was
	// waiting on it, so reading a finished or running step meant moving to it
	// and then pressing enter — a second keystroke to see the thing the cursor
	// was already pointing at. enter now closes it again.
	return !m.cardCollapsed
}

// minInlineCardRows is the terminal height below which the card stops rendering
// in place: header, step line and rule, the card's own frame, a few rows of
// content, and the footer.
const minInlineCardRows = 18

// stepBlockLines renders one step as it appears in the outline: its separator,
// title, rule, card, and tasks. It returns the lines and the offset of the
// title within them.
//
// The severe-cramping check measures with this same function rather than
// estimating row counts of its own. The estimate and the renderer had drifted —
// tasks were counted as one row each while rendering two — so the layout
// dropped to its most degraded tier at heights where the windowed one fits.
// The timeline rail down the left of the outline. Solid through finished work,
// dotted through work still ahead; the step the cursor is on is a filled node.
const (
	timelineDone        = "│"
	timelinePending     = "╎"
	timelineNode        = "○"
	timelineNodeCurrent = "●"
)

func (m Model) stepBlockLines(stepIndex int, selected, compact bool) (lines []string, titleOffset int, taskOffsets []int) {
	ch := m.session.Steps[stepIndex]
	// Physical lines, not logical ones. Cards and task rows are multi-line
	// strings, and holding one in a single slice entry meant the timeline rail
	// reached only its first row and the offsets below counted a whole card as
	// one line.
	add := func(block string) {
		lines = append(lines, strings.Split(block, "\n")...)
	}

	if !compact {
		lines = append(lines, "")
	}
	titleOffset = len(lines)
	add(m.renderStepLine(ch, stepIndex, selected))
	// No rule under the title. It existed to separate one step from the next
	// when a step was a bare line; every step draws a card now, and the rule sat
	// one row above that card's own top border drawing the same boundary twice.

	// The step's card renders here, directly under the step it belongs to.
	if m.stepShowsInlineCard(stepIndex) {
		if detail := m.renderDetail(); detail != "" {
			add(detail)
		}
	} else if card := m.renderStepStatusCard(stepIndex); card != "" {
		add(card)
	}

	// Tasks render outside the card, underneath it.
	if !m.stepShowsTasks(stepIndex) {
		m.drawTimelineGutter(lines, stepIndex, titleOffset, selected)
		return lines, titleOffset, nil
	}
	for i := range ch.Nodes {
		taskOffsets = append(taskOffsets, len(lines))
		add(m.renderNodeLine(ch.Nodes[i], m.nodeGlobalIndex(stepIndex, i), false))
	}
	m.drawTimelineGutter(lines, stepIndex, titleOffset, selected)
	return lines, titleOffset, taskOffsets
}

// drawTimelineGutter replaces each line's first two columns with a rail, the
// way a commit graph runs one down the left of the log.
//
// It reuses the columns the cursor marker had to itself, so the rail costs no
// width. The step the cursor is on is a filled node on that rail; the rest are
// open ones. The rail itself is solid through work that is finished and dotted
// through work that is not, which is the one thing the outline never said
// without the reader counting glyphs across four separate step headers.
func (m Model) drawTimelineGutter(lines []string, stepIndex, titleOffset int, selected bool) {
	rail := m.theme.StepRuleStyle.Render(timelinePending)
	if m.stepComplete(stepIndex) {
		rail = m.theme.MutedStyle.Render(timelineDone)
	}
	node := m.theme.MutedStyle.Render(timelineNode)
	if selected {
		node = m.theme.BrandStyle.Render(timelineNodeCurrent)
	}

	for i, line := range lines {
		glyph := rail
		if i == titleOffset {
			glyph = node
		}
		// A blank separator keeps the rail running so the line is continuous
		// between steps rather than restarting at each one.
		lines[i] = glyph + " " + ansi.TruncateLeft(line, rowCursorWidth, "")
	}
}

// stepComplete reports whether every task in a step is finished, which is what
// the solid rail means.
func (m Model) stepComplete(stepIndex int) bool {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return false
	}
	nodes := m.session.Steps[stepIndex].Nodes
	if len(nodes) == 0 {
		return false
	}
	for _, node := range nodes {
		if node.State != coop.NodeDone && node.State != coop.NodeSkipped {
			return false
		}
	}
	return true
}

// nodeGlobalIndex is a task's position across the whole session, which is what
// the node-level selection and mouse mapping are keyed on.
func (m Model) nodeGlobalIndex(stepIndex, nodeIndex int) int {
	idx := 0
	for i := 0; i < stepIndex; i++ {
		idx += len(m.session.Steps[i].Nodes)
	}
	return idx + nodeIndex
}

func (m Model) renderStepOutline() renderedOutline {
	if m.session == nil {
		return renderedOutline{navigationLine: map[int]navigationItem{}}
	}

	var lines []string
	navigationLines := map[int]navigationItem{}
	nodeIdx := 0

	cramped := m.outlineIsCramped()
	first, last, above, below := m.outlineWindow()
	if above > 0 {
		lines = append(lines, m.theme.MutedStyle.Render("  "+scrollMarker("▲", above, m.outlineIsSeverelyCramped())))
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
		block, titleOffset, taskOffsets := m.stepBlockLines(stepIdx, stepSelected, compact)
		navigationLines[len(lines)+titleOffset] = stepItem
		// Task rows stay mouse targets even though the cursor only stops on
		// steps: clicking one selects its step.
		for i, offset := range taskOffsets {
			navigationLines[len(lines)+offset] = navigationItem{
				kind: navigationNode, nodeIndex: nodeIdx + i, stepIndex: stepIdx,
			}
		}
		lines = append(lines, block...)
		nodeIdx += len(ch.Nodes)
	}

	if below > 0 {
		lines = append(lines, m.theme.MutedStyle.Render("  "+scrollMarker("▼", below, m.outlineIsSeverelyCramped())))
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
// scrollMarker labels what is hidden in one direction. Under severe pressure it
// drops to the arrow and a count, since the row it saves is a row of content.
func scrollMarker(arrow string, count int, compact bool) string {
	if compact {
		return fmt.Sprintf("%s %d", arrow, count)
	}
	direction := "above"
	if arrow == "▼" {
		direction = "below"
	}
	return fmt.Sprintf("%s %d more %s", arrow, count, direction)
}

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
	// Two levels of pressure. With the card not fitting, keep one step either
	// side for orientation. Tighter than that, the neighbors are the first
	// thing to go — the selected step and the two markers are what is left.
	span := 1
	if m.outlineIsSeverelyCramped() {
		span = 0
	}
	first = max(selected-span, 0)
	last = min(selected+span, total-1)
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

// outlineIsSeverelyCramped reports whether even the selected step plus one
// neighbor either side will not fit. At that point the neighbors are dropped
// and only the markers say there is more in each direction.
func (m Model) outlineIsSeverelyCramped() bool {
	height := m.viewport.Height()
	if height <= 0 || m.session == nil {
		return false
	}
	selected := 0
	if idx, ok := m.selectedStepIndex(); ok {
		selected = idx
	}
	blockRows := func(stepIdx int) int {
		block, _, _ := m.stepBlockLines(stepIdx, stepIdx == selected, stepIdx != selected)
		return lipgloss.Height(strings.Join(block, "\n"))
	}

	const markerRows = 2
	own := markerRows + blockRows(selected)
	withNeighbors := own
	for stepIdx := max(selected-1, 0); stepIdx <= min(selected+1, len(m.session.Steps)-1); stepIdx++ {
		if stepIdx != selected {
			withNeighbors += blockRows(stepIdx)
		}
	}

	// Drop the neighbors only when that is what buys the fit. If the selected
	// step overflows on its own, the view is scrolling either way and hiding
	// the neighbors costs orientation without gaining a row of content.
	//
	// The same tolerance the cramping check uses: a window that overshoots by a
	// row or two still shows its neighbors and scrolls, rather than blanking
	// them and leaving the space empty.
	const tolerance = 4
	return own <= height && withNeighbors > height+tolerance
}

// renderStepStatusCard is the small card under every step header.
//
// It carries what used to trail the header: the step's state and the count of
// its tasks by state. A collapsed step shows only this; the step in play shows
// the full review card instead.
// stepShowsTasks reports whether a step lists its tasks under its card.
//
// The step in play always does. A collapsed step does too when there is height
// for it — collapsing is about reclaiming rows under pressure, and with rows to
// spare there is no reason to hide work the user could be reading.
func (m Model) stepShowsTasks(stepIndex int) bool {
	// Only the selected step. Listing every step's tasks turned the outline
	// into a wall of rows where nothing stood out, and the small card now
	// carries what a step the reader is not standing on needs to say.
	selected, ok := m.selectedStepIndex()
	return ok && selected == stepIndex
}

// startsWithMarker reports whether a line opens with a status glyph and a
// space, which is what a hanging indent aligns under.
func startsWithMarker(text string) bool {
	plain := strings.TrimLeft(ansi.Strip(text), " ")
	for _, marker := range []string{"✓ ", "✗ ", "◆ ", "● ", "○ ", "– "} {
		if strings.HasPrefix(plain, marker) {
			return true
		}
	}
	return false
}

// wrapHanging wraps text and indents every line after the first, so a wrapped
// summary lines up under its own opening word instead of under its marker.
func wrapHanging(text string, width, indent int) string {
	// Only hang under a leading marker. A sentence that starts flush left —
	// "Needs you — 2 checks could not finish: X" — was being indented from its
	// second line on, which reads as a mid-sentence step in rather than as text
	// aligned under its own glyph.
	if width-indent < 8 || !startsWithMarker(text) {
		return wordWrap(text, width)
	}
	// Wrap to the narrower width up front. Wrapping to the full width and then
	// adding the indent afterwards pushes every continuation line past the
	// border by exactly the indent.
	lines := strings.Split(wordWrap(text, width-indent), "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.Repeat(" ", indent) + lines[i]
	}
	return strings.Join(lines, "\n")
}

// limitWrappedLines keeps at most max lines of already-wrapped text, marking
// the last one when it drops any.
func limitWrappedLines(s string, max, width int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= max {
		return s
	}
	lines = lines[:max]
	last := len(lines) - 1
	lines[last] = ansi.Truncate(lines[last], width-1, "") + "…"
	return strings.Join(lines, "\n")
}

// stepCardSummary is what the small card says about a step.
//
// It used to be a row of glyphs — "✓1", "●1", "needs you · ✗2 · ◆2" — which
// said how many of something there were without saying what happened or what
// the reader should do about it. A step the reader is not standing on shows no
// tasks now, so this line carries the whole story for it.
func (m Model) stepCardSummary(stepIndex int) string {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return ""
	}
	step := &m.session.Steps[stepIndex]

	if m.stepJustConfirmed(stepIndex) {
		return m.theme.SuccessStyle.Render("✓ confirmed")
	}

	// A failure outranks everything else the card could say. A finished step
	// with a check that did not pass was reporting a green tick and the note
	// describing what it built, which reads as success.
	failed, _ := stepCheckResults(step)
	awaiting := m.stepReviewCount(stepIndex) > 0
	if len(failed) > 0 {
		lead := m.theme.SoftErrorStyle.Render("✗ ")
		if awaiting {
			lead = m.theme.SoftAttentionStyle.Render("Needs you") + m.theme.MutedStyle.Render(" — ")
		}
		if cause := stepBlockingCause(step); cause != "" {
			return lead + m.theme.MutedStyle.Render(pluralChecks(len(failed))+" could not finish: ") +
				highlightIdentifiers(cause, m.theme.MutedStyle, m.theme.IdentifierStyle)
		}
		return lead + highlightIdentifiers(failed[0], m.theme.MutedStyle, m.theme.IdentifierStyle)
	}

	// Waiting on the reader: say so, and say what to do rather than counting
	// the tasks involved.
	if awaiting {
		return m.theme.SoftAttentionStyle.Render("Needs you") +
			m.theme.MutedStyle.Render(" — review the work and confirm the step.")
	}

	// In progress: name the task actually running, which nothing else did once
	// the tasks stopped being listed under every step.
	for _, node := range step.Nodes {
		if node.State != coop.NodeActive {
			continue
		}
		detail := node.TitleText()
		if node.Activity != "" {
			detail = node.Activity
		}
		// Same icon the task row shows, so a running step reads as running at a
		// glance rather than only on the word "Working".
		return m.nodeIcon(node) + m.theme.MutedStyle.Render(" Working on ") +
			m.theme.StepTitleStyle.Render(detail)
	}

	// Finished: what the agent actually built, in its own words.
	if note := stepWorkSummary(step); note != "" {
		return m.theme.SuccessStyle.Render("✓ ") + m.theme.MutedStyle.Render(note)
	}
	if pending := countNodes(step, coop.NodePending); pending == len(step.Nodes) && pending > 0 {
		return m.theme.DimmedStyle.Render(fmt.Sprintf("Not started — %s queued", pluralTasks(pending)))
	}
	return ""
}

// stepWorkSummary is the agent's own account of what it built, trimmed to the
// first sentence or two.
func stepWorkSummary(step *coop.SessionStep) string {
	for _, node := range step.Nodes {
		if node.Implementation == nil {
			continue
		}
		if note := firstSentences(node.Implementation.Note, 2); note != "" {
			return note
		}
	}
	return ""
}

// stepBlockingCause returns the shared reason a step's checks did not pass,
// when every failure names the same one.
func stepBlockingCause(step *coop.SessionStep) string {
	cause := ""
	for _, node := range step.Nodes {
		for _, verification := range node.Verifications {
			if verification.Passed {
				continue
			}
			reason := summarizeCheckFailure(verification.Check, 0)
			if reason == "" {
				continue
			}
			if cause == "" {
				cause = reason
				continue
			}
			if cause != reason {
				return ""
			}
		}
	}
	return cause
}

// firstSentences keeps the opening of a longer note.
func firstSentences(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	sentences := splitSentences(s)
	if len(sentences) == 0 {
		return ""
	}
	if len(sentences) > n {
		sentences = sentences[:n]
	}
	return strings.Join(sentences, " ")
}

func countNodes(step *coop.SessionStep, state coop.NodeState) int {
	count := 0
	for _, node := range step.Nodes {
		if node.State == state {
			count++
		}
	}
	return count
}

func pluralTasks(n int) string {
	if n == 1 {
		return "1 task"
	}
	return fmt.Sprintf("%d tasks", n)
}

func (m Model) renderStepStatusCard(stepIndex int) string {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return ""
	}
	body := m.stepCardSummary(stepIndex)
	if body == "" {
		return ""
	}

	// Same widths as the review card, so the two line up under the same rule
	// instead of sitting at different insets depending on the step's state.
	width, inner := m.detailWidths()
	if inner < 8 {
		inner = 8
	}
	// Two lines, wrapped with a hanging indent so the second line starts under
	// the first line's text rather than under its glyph.
	const hang = 2
	body = limitWrappedLines(wrapHanging(body, inner, hang), 2, inner)
	card := m.theme.StatusCardStyle.Width(width).Render(body)
	return indentBlock(card, rowCursorWidth)
}

func (m Model) renderStepLine(ch coop.SessionStep, stepIndex int, selected bool) string {
	// The gutter is drawn by the timeline rail afterwards; this reserves its
	// two columns so the rest of the line lands where it always did.
	prefix := "  "
	// No disclosure marker. The timeline node on the left says where the cursor
	// is and the card below says what the step is doing; a "+" or "-" beside
	// them was a third glyph competing to say the same thing.
	disclosure := ""
	title := ch.TitleText()
	if selected {
		// A filled title, not just a bold one behind a marker. The fill needs a
		// column of padding on each side to read as a block rather than as
		// inverted text.
		title = m.theme.SelectedStepTitleStyle.Render(title)
	} else {
		// Match the fill's left padding so titles hold the same column whether
		// or not the cursor is on them, instead of shifting a column as it
		// moves.
		title = " " + m.theme.StepTitleStyle.Render(title)
	}
	// Title only. Everything that used to trail the header — the state, the
	// per-state counts, the failure badge — now sits in the card directly
	// beneath it, where it is one readable line instead of a row of glyphs
	// competing with the title for the same space.
	line := prefix + m.theme.MutedStyle.Render(disclosure) + title
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
		if node.IsInformationalNode {
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

// taskIndent is the column a task row starts in: the card's inner text column,
// so tasks line up with the card body they hang from rather than with its
// border. rowCursorWidth of it is the cursor gutter, the rest is the card's
// border and padding.
const taskIndent = rowCursorWidth + 2

// nodeCheckResults returns this task's failure reasons and its passed count.
func nodeCheckResults(node coop.SessionNode) ([]string, int) {
	var failed []string
	passed := 0
	for _, verification := range node.Verifications {
		if verification.Passed {
			passed++
			continue
		}
		if reason := summarizeCheckFailure(verification.Check, failedCheckBudget); reason != "" {
			failed = append(failed, reason)
		}
	}
	return failed, passed
}

func (m Model) renderNodeLine(node coop.SessionNode, idx int, selected bool) string {
	icon := m.nodeIcon(node)

	// Indent to the card's text column, not to its border. The tasks belong to
	// the card above them and read as a continuation of it; starting them two
	// columns to the left of its first character made them look like a separate
	// list that happened to follow.
	cursor := strings.Repeat(" ", taskIndent)
	if selected {
		cursor = m.theme.BrandStyle.Render(cursorMarker) + strings.Repeat(" ", taskIndent-rowCursorWidth)
	}

	title := node.TitleText()
	if node.State == coop.NodeSkipped {
		title = m.theme.DimmedStyle.Render(title)
	} else if selected {
		title = lipgloss.NewStyle().Bold(true).Render(title)
	}

	// One definition of the room a continuation line has. Truncating the path
	// to one width and then wrapping to a narrower one put the tail of an
	// already-ellipsised path on a second line.
	wrapW := m.outlineWidth() - (taskIndent + 2)
	if wrapW < 20 {
		wrapW = 20
	}

	var annText string
	var annStyle func(string) string
	switch {
	case node.Implementation != nil && node.Implementation.File != "":
		ann := node.Implementation.File
		if node.Implementation.Lines != "" {
			ann += ":" + node.Implementation.Lines
		}
		// One line, middle-ellipsised. Wrapping a long path spread it over four
		// rows of fragments that read as separate files.
		annText = truncatePath(ann, wrapW)
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
	// The task carries its own check results. They used to live in a separate
	// "Agent checks" section that restated every task title above it, so the
	// same list appeared twice with the second copy prefixed by the first.
	// A ratio when some checks passed and some did not. "done ✗1 ✓1" put three
	// status tokens in a row and left the reader to work out that one check
	// passed while another could not run; "✗1 of 2" says it.
	failed, passed := nodeCheckResults(node)
	switch {
	case len(failed) > 0 && passed > 0:
		line += "  " + m.theme.SoftErrorStyle.Render(fmt.Sprintf("✗%d of %d", len(failed), len(failed)+passed))
	case len(failed) > 0:
		line += "  " + m.theme.SoftErrorStyle.Render(fmt.Sprintf("✗%d", len(failed)))
	case passed > 0:
		line += "  " + m.theme.SoftSuccessStyle.Render(fmt.Sprintf("✓%d", passed))
	}

	// Continuation lines align with the task title, not with its icon.
	// Truncate the row like the step line above it. Left to wrap, a long task
	// title spilled past the outline's width — and in the split workspace the
	// overflow restarted at column zero, straddling both panes.
	if width := m.outlineWidth(); width > 0 && lipgloss.Width(ansi.Strip(line)) > width {
		line = ansi.Truncate(line, width, "…")
	}

	indent := strings.Repeat(" ", taskIndent+2)
	for _, reason := range failed {
		// The ✗ marks the finding once. Repeating it on every wrapped line read
		// as several separate failures.
		for i, wl := range strings.Split(wordWrap(reason, wrapW-2), "\n") {
			marker := m.theme.SoftErrorStyle.Render("✗ ")
			if i > 0 {
				marker = "  "
			}
			line += "\n" + indent + marker +
				highlightIdentifiers(wl, m.theme.MutedStyle, m.theme.IdentifierStyle)
		}
	}

	if annText != "" {
		wrapped := wordWrap(annText, wrapW)
		for _, wl := range strings.Split(wrapped, "\n") {
			line += "\n" + indent + annStyle(wl)
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
		// Truncate, do not clamp. MaxWidth wraps rather than cuts, so an
		// uninitialised spinner — which renders the literal "(error)" — came
		// out as seven rows of one character each, tearing the row apart. Any
		// frame that is not a single cell falls back to a static marker.
		frame := strings.TrimSpace(strings.ReplaceAll(m.spinner.View(), "\n", ""))
		if frame == "" || lipgloss.Width(ansi.Strip(frame)) != 1 {
			return m.theme.MutedStyle.Render("●")
		}
		return frame
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
