package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/coop"
)

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
