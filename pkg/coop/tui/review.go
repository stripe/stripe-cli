package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func (m Model) renderFooter() string {
	// Completion view has its own footer; don't render step footer.
	if m.session != nil && m.session.IsComplete() {
		return ""
	}

	var lines []string

	switch {
	case m.agentHeartbeatMissing:
		lines = append(lines, m.theme.ErrorStyle.Render("  Agent is not running. Restart it, or run: stripe coop status"))
	case m.agentIdle():
		lines = append(lines, m.theme.AttentionStyle.Render("  Waiting for agent: no recent updates. Reconnect: stripe coop status"))
	}

	if m.statusMessage != "" {
		lines = append(lines, m.theme.AttentionStyle.Render("  "+m.statusMessage))
	}

	if m.session != nil {
		if count := m.actionableReviewCount(); count > 0 {
			lines = append(lines, "")
			lines = append(lines, m.theme.AttentionStyle.Render("  Waiting for you: review step"))
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
	// Too short for the bordered card, which costs two rows of frame. Fall back
	// to a borderless line so the reviewer still learns there is something
	// waiting, rather than the card vanishing outright. Never do this while the
	// user is typing feedback — the input has to stay on screen.
	if maxHeight > 0 && maxHeight < 3 && !m.rejecting {
		if maxHeight < 1 {
			return ""
		}
		return m.theme.MutedStyle.Render("  To confirm: enter for details")
	}
	w, _ := m.reviewCardWidths()

	var lines []cardLine
	lines = append(lines, cardLine{text: m.theme.ReviewStyle.Render("Review step")})

	// The blueprint's own description says what the step is for. It is phrased
	// as an imperative, exactly like the review prompt below it, so without a
	// label the reviewer reads two commands and cannot tell which one to act
	// on. "Goal:" marks it as context.
	if goal := m.reviewGoalLabel(target.stepIndex); goal != "" {
		lines = append(lines, cardLine{})
		lines = append(lines, cardLine{
			text: m.theme.MutedStyle.Render("Goal ") + m.theme.MutedStyle.Render(goal),
		})
	}

	// Feedback the user already gave outranks a fresh instruction: the agent is
	// reworking, so there is nothing to confirm yet.
	if note := m.requestedChangeLabel(target.nodeNumbers); note != "" {
		lines = append(lines, cardLine{})
		lines = append(lines, cardLine{text: feedbackLabel(m.theme, "Requested change")})
		lines = append(lines, cardLine{text: note})
	}

	// What the reviewer should do comes next: the blueprint's own review prompt
	// is written for a human, unlike the agent's verification notes.
	prompts := m.reviewPromptLabels(target.nodeNumbers)
	if len(prompts) > 0 {
		lines = append(lines, cardLine{})
		lines = append(lines, cardLine{text: actionLabel(m.theme, "To confirm")})
		for _, prompt := range prompts {
			if len(prompts) > 1 {
				lines = append(lines, cardLine{text: m.theme.BrandStyle.Render("• ") + prompt})
				continue
			}
			lines = append(lines, cardLine{text: prompt})
		}
	}
	// Say where to go. A review command already answers that, so only fall back
	// to the node type when there isn't one.
	if command := m.reviewCommandLabel(target.nodeNumbers); command != "" {
		lines = append(lines, cardLine{
			text: m.theme.MutedStyle.Render("Run ") + m.theme.BrandStyle.Render(command),
		})
	} else if venue := m.reviewVenueLabel(target.nodeNumbers); venue != "" {
		lines = append(lines, cardLine{
			text: m.theme.MutedStyle.Render("Where ") + venue,
		})
	}

	// Failures are named; passes collapse to a count. A reviewer needs to know
	// which check failed far more than which ones succeeded.
	failed := m.reviewFailedCheckLabels(target.nodeNumbers)
	passed := m.reviewPassedCheckCount(target.nodeNumbers)
	if len(failed) > 0 || passed > 0 {
		lines = append(lines, cardLine{})
		lines = append(lines, cardLine{text: checksLabel(m.theme, len(failed) > 0)})
	}
	for _, label := range failed {
		lines = append(lines, cardLine{
			text: m.theme.ErrorStyle.Render("✗ ") + label,
		})
	}
	if passed > 0 {
		lines = append(lines, cardLine{
			text: m.theme.SuccessStyle.Render("✓ ") + m.theme.MutedStyle.Render(fmt.Sprintf("%s passed", pluralChecks(passed))),
		})
	}

	metadataStart := len(lines)
	// A single-task step already names itself in the outline, so only spell out
	// the covered tasks when the step actually groups several.
	if len(target.nodeNumbers) > 1 {
		if included := m.reviewNodeTitleLabel(target.nodeNumbers); included != "" {
			lines = append(lines, cardLine{
				text: m.theme.MutedStyle.Render("Includes ") + m.theme.DimmedStyle.Render(included),
			})
		}
	}
	if changed := m.reviewChangedLabel(target.nodeNumbers); changed != "" {
		lines = append(lines, cardLine{
			text: m.theme.MutedStyle.Render("Changed ") + m.theme.FileAnnotationStyle.Render(changed),
		})
	}
	if len(lines) > metadataStart {
		lines = append(lines[:metadataStart], append([]cardLine{{}}, lines[metadataStart:]...)...)
	}
	// The feedback editor is pinned: it is appended last, so plain tail
	// truncation would cut the very thing the user is typing into. Wrap it
	// separately and always keep it, trimming the body above it instead.
	var pinned []cardLine
	if m.rejecting {
		m.rejectionInput.SetWidth(m.requestChangesInputWidth())
		inputView := m.rejectionInput.View()
		if m.rejectionInput.Value() == "" {
			inputView = m.theme.DimmedStyle.Render(m.rejectionInput.Placeholder)
		}
		pinned = append(pinned, cardLine{text: m.theme.ErrorStyle.Render("Request changes: ") + inputView})
		if m.rejectionError != "" {
			pinned = append(pinned, cardLine{text: m.theme.ErrorStyle.Render(m.rejectionError)})
		}
	}

	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, line.wrap(w-4)...)
	}
	var wrappedPinned []string
	for _, line := range pinned {
		wrappedPinned = append(wrappedPinned, line.wrap(w-4)...)
	}

	if maxHeight > 0 {
		maxContentLines := maxHeight - 2 - len(wrappedPinned)
		if len(wrapped) > maxContentLines {
			switch {
			case maxContentLines < 1:
				wrapped = nil
			case maxContentLines == 1:
				wrapped = []string{m.theme.DimmedStyle.Render("Review step")}
			default:
				// Keep naming the action even when the card collapses, so a
				// narrow terminal still says there is something to do.
				more := m.theme.MutedStyle.Render("To confirm: enter for details")
				wrapped = append(trimDanglingSection(wrapped[:maxContentLines-1]), more)
			}
		}
	}
	return m.renderReviewCardLines(w, maxHeight, append(wrapped, wrappedPinned...))
}

// cardLine is one logical line of the review card. indent is applied to
// continuation lines so wrapped text stays visually attached to its label
// instead of reading as a new item.
type cardLine struct {
	text   string
	indent string
}

func (l cardLine) wrap(width int) []string {
	var out []string
	available := width - lipgloss.Width(l.indent)
	for _, segment := range strings.Split(l.text, "\n") {
		for i, piece := range strings.Split(wordWrap(segment, available), "\n") {
			if i > 0 {
				piece = l.indent + piece
			}
			out = append(out, piece)
		}
	}
	return out
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
	more := m.theme.MutedStyle.Render("To confirm: enter for details")
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

func (m Model) requestChangesPlaceholder(_ reviewTarget) string {
	return "Describe what should change in this step"
}

func (m Model) reviewChangedLabel(nodeNumbers []int) string {
	_, innerW := m.reviewCardWidths()
	var labels []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil || node.Implementation == nil || node.Implementation.File == "" {
			continue
		}
		label := truncatePath(implementationFileLabel(node.Implementation), innerW)
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

func (m Model) reviewNodeTitleLabel(nodeNumbers []int) string {
	if m.session == nil {
		return ""
	}
	var titles []string
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
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

// reviewPromptLabel returns the instruction for the reviewer. Blueprint review
// prompts are authored for a human and say what to go and check; the agent's
// verification notes are free-form prose describing what it already did, so
// they belong in the detail view rather than in the card.
// sectionNames are the box's headings, used to avoid truncating a card so that
// a heading is left with nothing under it.
var sectionNames = map[string]bool{
	"To confirm":   true,
	"Agent checks": true,

	"Requested change": true,
}

// trimDanglingSection drops trailing blank lines and any heading left with no
// content beneath it, so a truncated card never ends on an empty section.
func trimDanglingSection(lines []string) []string {
	for len(lines) > 0 {
		last := strings.TrimSpace(ansi.Strip(lines[len(lines)-1]))
		if last != "" && !sectionNames[last] {
			break
		}
		lines = lines[:len(lines)-1]
	}
	return lines
}

// Section headings carry hue as well as weight. The layout is flush left, so
// color and weight are the only things left to separate a heading from the
// sentence under it — and the headings do not all mean the same thing, so they
// do not all look the same:
//
//	Do this           the action, in the same orange as "needs you"
//	Checks            evidence, in the review purple
//	Requested change  feedback already given, in the error hue
//
// All three stay bold, so the structure survives a terminal without color.
func actionLabel(t Theme, text string) string {
	return t.ActionHeadingStyle.Render(text)
}

// evidenceLabel colors the Checks heading by its own outcome, so the heading
// carries information rather than just naming the section: red when something
// failed, green when everything passed.
func evidenceLabel(t Theme, text string) string {
	return t.ReviewStyle.Bold(true).Render(text)
}

// checksLabel names whose checks these are — the agent's, not the reader's
// task — and colors the heading by its own outcome.
func checksLabel(t Theme, failed bool) string {
	if failed {
		return t.ErrorStyle.Bold(true).Render("Agent checks")
	}
	return t.SuccessStyle.Bold(true).Render("Agent checks")
}

func feedbackLabel(t Theme, text string) string {
	return t.ErrorStyle.Bold(true).Render(text)
}

// reviewGoalLabel is the blueprint's own statement of what the step is for.
func (m Model) reviewGoalLabel(stepIndex int) string {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return ""
	}
	return strings.TrimSpace(m.session.Steps[stepIndex].DescriptionText())
}

// requestedChangeLabel surfaces feedback the user already sent. It was stored
// on the node and never rendered anywhere, so by the time the agent came back
// the user had no reminder of what they had asked for.
func (m Model) requestedChangeLabel(nodeNumbers []int) string {
	var notes []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		note := strings.TrimSpace(node.RejectionNote)
		if note == "" || seen[note] {
			continue
		}
		seen[note] = true
		notes = append(notes, note)
	}
	return strings.Join(notes, " ")
}

func (m Model) reviewPromptLabels(nodeNumbers []int) []string {
	if prompts := m.reviewBlueprintConfirmationPrompts(nodeNumbers); len(prompts) > 0 {
		return prompts
	}
	return []string{"Confirm the completed work matches this step and its verification evidence."}
}

// reviewFailedCheckLabels names the checks the agent could not get passing.
// failedCheckBudget keeps a named failure to roughly two wrapped lines in the
// card. The full text remains available in the detail view.
const failedCheckBudget = 160

func (m Model) reviewFailedCheckLabels(nodeNumbers []int) []string {
	var checks []string
	seen := map[string]bool{}
	showStepTitle := len(nodeNumbers) > 1
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		for _, verification := range node.Verifications {
			check := summarizeCheck(verification.Check, failedCheckBudget)
			if verification.Passed || check == "" || seen[check] {
				continue
			}
			seen[check] = true
			if showStepTitle && node.Title != "" {
				check = node.Title + ": " + check
			}
			checks = append(checks, check)
		}
	}
	if len(checks) > 3 {
		return append(checks[:3], fmt.Sprintf("+%d more failed", len(checks)-3))
	}
	return checks
}

func (m Model) reviewPassedCheckCount(nodeNumbers []int) int {
	passed := 0
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		for _, verification := range node.Verifications {
			if verification.Passed {
				passed++
			}
		}
	}
	return passed
}

// checkNoisePrefixes are lines agents paste in from CLI output they ran. They
// describe the run, not the result, so they are dropped before summarizing.
var checkNoisePrefixes = []string{
	"Setting up fixture for:",
	"Running fixture for:",
	"Checking for new versions",
	"A newer version of the Stripe CLI is available",
	"We recommend updating regularly",
	"Trigger succeeded!",
	"Ran fixture",
}

// summarizeCheck condenses an agent-authored verification note to one scannable
// line.
//
// These notes are frequently pasted CLI transcripts rather than prose. Agents
// splice command output into the middle of their own sentence, so the note
// opens with a fragment, carries dozens of progress lines, and states the
// actual finding last. Keeping the head — whether by sentence split or plain
// truncation — therefore returns the chatter and discards the result. Drop the
// pure progress lines, then keep the tail. The untouched text stays in the
// detail view.
func summarizeCheck(s string, budget int) string {
	var kept []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || isProgressLine(line) {
			continue
		}
		kept = append(kept, line)
	}
	summary := strings.Join(strings.Fields(strings.Join(kept, " ")), " ")
	if budget <= 0 || lipgloss.Width(summary) <= budget {
		return summary
	}
	return "…" + ansi.TruncateLeft(summary, lipgloss.Width(summary)-budget, "")
}

// isProgressLine reports whether a line is only CLI progress output. A line that
// carries meaningful text alongside a known marker is kept, because agents
// append their finding to the last line of pasted output.
func isProgressLine(line string) bool {
	for _, prefix := range checkNoisePrefixes {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		return lipgloss.Width(strings.TrimSpace(strings.TrimPrefix(line, prefix))) <= 60
	}
	return false
}

func pluralChecks(n int) string {
	if n == 1 {
		return "1 check"
	}
	return fmt.Sprintf("%d checks", n)
}

// reviewVenueLabel says where the reviewer goes to check the work. Node type is
// the reliable signal: blueprint review prompts are shared across every node of
// a given type and mostly describe what to confirm, not where to confirm it.
func (m Model) reviewVenueLabel(nodeNumbers []int) string {
	var venues []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		venue := reviewVenueForNode(node)
		if venue == "" || seen[venue] {
			continue
		}
		seen[venue] = true
		venues = append(venues, venue)
	}
	const limit = 2
	if len(venues) > limit {
		venues = venues[:limit]
	}
	return strings.Join(venues, "; ")
}

func reviewVenueForNode(node *coop.SessionNode) string {
	switch node.Type {
	case coop.NodeAPIRequest, coop.NodeTestHelper:
		return "the changed files below"
	case coop.NodeUIComponent:
		return "your running app"
	case coop.NodeDashboard:
		return "the Stripe Dashboard"
	case coop.NodeCLICommand:
		return "your terminal"
	case coop.NodeAsyncHandler, coop.NodeSetUpWebhooks:
		return "your webhook handler, after triggering the event"
	}
	return ""
}

func (m Model) reviewBlueprintConfirmationPrompts(nodeNumbers []int) []string {
	var prompts []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil || node.ReviewPrompt == "" || seen[node.ReviewPrompt] {
			continue
		}
		seen[node.ReviewPrompt] = true
		prompts = append(prompts, node.ReviewPrompt)
	}
	const limit = 2
	if len(prompts) > limit {
		return append(prompts[:limit], fmt.Sprintf("Open details for %d more check(s).", len(prompts)-limit))
	}
	return prompts
}

func (m Model) reviewCommandLabel(nodeNumbers []int) string {
	var commands []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
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
	return strings.Join(commands, "\n")
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
	countedSteps := map[int]bool{}
	step := 0
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			step++
			if m.session.Steps[i].Nodes[j].State != coop.NodeReview || !m.reviewIsActionable(step) {
				continue
			}
			if !countedSteps[i] {
				count++
				countedSteps[i] = true
			}
		}
	}
	return count
}

func (m Model) agentIdle() bool {
	return m.agentIsIdle
}
