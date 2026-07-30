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

	if m.agentIdle() {
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
	if maxHeight > 0 && maxHeight < 3 {
		return ""
	}
	w, _ := m.reviewCardWidths()

	var lines []cardLine
	prefix := "Review"
	if target.kind == "step" {
		prefix = "Review step"
	}
	lines = append(lines, cardLine{text: m.theme.ReviewStyle.Render(prefix)})

	// What the reviewer should do comes first: the blueprint's own review
	// prompt is written for a human, unlike the agent's verification notes.
	prompts := m.reviewPromptLabels(target.nodeNumbers)
	if len(prompts) > 0 {
		lines = append(lines, cardLine{})
		lines = append(lines, cardLine{text: m.theme.ConfirmationHeaderStyle.Render("Do this")})
		for _, prompt := range prompts {
			if len(prompts) > 1 {
				lines = append(lines, cardLine{text: "• " + prompt, indent: "  "})
				continue
			}
			lines = append(lines, cardLine{text: prompt})
		}
	}
	// Say where to go. A review command already answers that, so only fall back
	// to the node type when there isn't one.
	if command := m.reviewCommandLabel(target.nodeNumbers); command != "" {
		lines = append(lines, cardLine{
			text:   m.theme.MutedStyle.Render("Run: ") + command,
			indent: "  ",
		})
	} else if venue := m.reviewVenueLabel(target.nodeNumbers); venue != "" {
		lines = append(lines, cardLine{
			text:   m.theme.MutedStyle.Render("Where: ") + venue,
			indent: "  ",
		})
	}

	// Failures are named; passes collapse to a count. A reviewer needs to know
	// which check failed far more than which ones succeeded.
	failed := m.reviewFailedCheckLabels(target.nodeNumbers)
	passed := m.reviewPassedCheckCount(target.nodeNumbers)
	if len(failed) > 0 || passed > 0 {
		lines = append(lines, cardLine{})
	}
	for _, label := range failed {
		lines = append(lines, cardLine{
			text:   m.theme.ErrorStyle.Render("✗ ") + label,
			indent: "  ",
		})
	}
	if passed > 0 {
		lines = append(lines, cardLine{
			text: m.theme.MutedStyle.Render(fmt.Sprintf("✓ %s passed", pluralChecks(passed))),
		})
	}

	metadataStart := len(lines)
	if target.kind == "step" {
		if included := m.reviewNodeTitleLabel(target.nodeNumbers); included != "" {
			lines = append(lines, cardLine{
				text:   m.theme.MutedStyle.Render("Includes: ") + included,
				indent: "  ",
			})
		}
	}
	if changed := m.reviewChangedLabel(target.nodeNumbers); changed != "" {
		lines = append(lines, cardLine{
			text:   m.theme.MutedStyle.Render("Changed: ") + changed,
			indent: "  ",
		})
	}
	if len(lines) > metadataStart {
		lines = append(lines[:metadataStart], append([]cardLine{{}}, lines[metadataStart:]...)...)
	}
	if m.rejecting {
		m.rejectionInput.SetWidth(m.requestChangesInputWidth())
		inputView := m.rejectionInput.View()
		if m.rejectionInput.Value() == "" {
			inputView = m.theme.DimmedStyle.Render(m.rejectionInput.Placeholder)
		}
		lines = append(lines, cardLine{text: m.theme.ErrorStyle.Render("Request changes: ") + inputView})
		if m.rejectionError != "" {
			lines = append(lines, cardLine{text: m.theme.ErrorStyle.Render(m.rejectionError)})
		}
	}

	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, line.wrap(w-4)...)
	}
	if maxHeight > 0 {
		maxContentLines := maxHeight - 2
		if len(wrapped) > maxContentLines {
			if maxContentLines <= 1 {
				wrapped = []string{m.theme.DimmedStyle.Render("Review: enter for details")}
			} else {
				// Keep naming the action even when the card collapses, so a
				// narrow terminal still says there is something to do.
				more := m.theme.DimmedStyle.Render("Do this: enter/e for details")
				wrapped = append(wrapped[:maxContentLines-1], more)
			}
		}
	}
	return m.renderReviewCardLines(w, maxHeight, wrapped)
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
	more := m.theme.DimmedStyle.Render("Review: enter for details")
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

func (m Model) requestChangesPlaceholder(target reviewTarget) string {
	if target.kind == "step" {
		return "Describe what should change in this step"
	}
	for _, nodeNumber := range target.nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		switch node.NodeType {
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

func (m Model) reviewChangedLabel(nodeNumbers []int) string {
	var labels []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
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

func (m Model) reviewNodeTitleLabel(nodeNumbers []int) string {
	if m.session == nil {
		return ""
	}
	var titles []string
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil || node.TitleText() == "" {
			continue
		}
		titles = append(titles, node.TitleText())
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
			if showStepTitle && node.TitleText() != "" {
				check = node.TitleText() + ": " + check
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
	switch node.NodeType {
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
	return node.ReviewCommand
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
