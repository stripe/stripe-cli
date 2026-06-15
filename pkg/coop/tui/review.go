package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

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

	var lines []string
	prefix := "Review"
	if target.kind == "step" {
		prefix = "Review step"
	}
	lines = append(lines, m.theme.ReviewStyle.Render(prefix))
	check := m.reviewPromptLabel(target.nodeNumbers)
	if check != "" {
		lines = append(lines, m.theme.ConfirmationHeaderStyle.Render("Confirmation steps"))
		lines = append(lines, check)
	}
	metadataStart := len(lines)
	if target.kind == "step" {
		if included := m.reviewNodeTitleLabel(target.nodeNumbers); included != "" {
			lines = append(lines, m.theme.MutedStyle.Render("Includes: ")+included)
		}
	}
	if changed := m.reviewChangedLabel(target.nodeNumbers); changed != "" {
		lines = append(lines, m.theme.MutedStyle.Render("Agent changed: ")+changed)
	}
	if verified := m.reviewVerificationLabel(target.nodeNumbers); verified != "" {
		lines = append(lines, m.theme.MutedStyle.Render("Agent verified: ")+verified)
	}
	if command := m.reviewCommandLabel(target.nodeNumbers); command != "" {
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

func (m Model) requestChangesPlaceholder(target reviewTarget) string {
	if target.kind == "step" {
		return "Describe what should change in this step"
	}
	for _, nodeNumber := range target.nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
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

func (m Model) reviewVerificationLabel(nodeNumbers []int) string {
	passed := 0
	total := 0
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
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

func (m Model) reviewPromptLabel(nodeNumbers []int) string {
	if agentChecks := m.reviewAgentConfirmationLabel(nodeNumbers); agentChecks != "" {
		return agentChecks
	}
	if blueprintChecks := m.reviewBlueprintConfirmationLabel(nodeNumbers); blueprintChecks != "" {
		return blueprintChecks
	}
	return "Confirm the completed work matches this step and its verification evidence."
}

func (m Model) reviewAgentConfirmationLabel(nodeNumbers []int) string {
	var checks []string
	seen := map[string]bool{}
	showStepTitle := len(nodeNumbers) > 1
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
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

func (m Model) reviewBlueprintConfirmationLabel(nodeNumbers []int) string {
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
	countedSteps := map[int]bool{}
	step := 0
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			step++
			if m.session.Steps[i].Nodes[j].State != coop.NodeReview || !m.reviewIsActionable(step) {
				continue
			}
			if m.session.ReviewGranularityForNode(step) == coop.ReviewGranularityStep {
				if !countedSteps[i] {
					count++
					countedSteps[i] = true
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
	age, err := m.store.HeartbeatAge(m.sessionID)
	if err != nil {
		return false
	}
	if age >= 0 && age < 5*time.Second {
		return false
	}
	if m.lastUpdateTime.IsZero() {
		return false
	}
	return time.Since(m.lastUpdateTime) > 2*time.Minute
}
