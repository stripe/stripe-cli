package tui

import (
	"time"

	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stripe/stripe-cli/pkg/coop"
)

const (
	// The outline is how the user knows where they are in the blueprint. The
	// footer can collapse to a one-line hint and the review card already
	// degrades gracefully, so when space runs out those give way first and the
	// outline keeps at least a few rows.
	minViewportHeight   = 3
	terminalScrollGuard = 1

	// readErrorTolerance is how many consecutive failed session polls are
	// tolerated before the error is shown. At a 500ms tick this is ~2 seconds.
	readErrorTolerance = 4

	// confirmSettleDuration is how long a confirmed step stays acknowledged in
	// place before it collapses into the finished list.
	confirmSettleDuration = 700 * time.Millisecond
	rowCursorWidth        = 2
	rowRightGap           = 2
	maxRuleWidth          = 80
	detailIndent          = rowCursorWidth
	cursorMarker          = "> "
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
	subtitleLines := strings.Split(wordWrap("The agent will start the next session here. You can leave this open.", w), "\n")

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

	skipped := m.session.NodeSummary()[coop.NodeSkipped]

	// Count steps, not tasks: the step is the unit the user confirms, so a
	// task-based count reported progress through something they never act on
	// directly. A step counts as finished when all of its tasks are.
	stepsDone := 0
	for _, step := range m.session.Steps {
		finished := true
		for _, node := range step.Nodes {
			if node.State != coop.NodeDone && node.State != coop.NodeSkipped {
				finished = false
				break
			}
		}
		if finished {
			stepsDone++
		}
	}
	progress := fmt.Sprintf("step %d/%d", min(stepsDone+1, len(m.session.Steps)), len(m.session.Steps))
	if skipped > 0 {
		progress += fmt.Sprintf(" · %d skipped", skipped)
	}
	rightPart := m.theme.MutedStyle.Render(right + " · " + progress)

	available := m.contentWidth()
	var header string
	if lipgloss.Width(left)+lipgloss.Width(rightPart)+4 > available {
		// Stacked fallback: truncate too, or a long blueprint name runs past
		// the terminal edge (real sessions carry names the fixtures do not).
		header = left + "\n  " + ansi.Truncate(rightPart, max(available-2, 1), "…")
	} else {
		header = lipgloss.JoinHorizontal(lipgloss.Top, left, lipgloss.PlaceHorizontal(available-lipgloss.Width(left), lipgloss.Right, rightPart))
	}

	if claimURL := m.sandboxClaimLink(); claimURL != "" {
		url := claimURL
		maxW := available - 10
		if maxW > 0 {
			// Width-aware truncation (byte-slicing could split a multibyte rune).
			url = ansi.Truncate(url, maxW, "…")
		}
		header += "\n" + m.theme.DimmedStyle.Render("  ⚡ ") + m.theme.BrandStyle.Hyperlink(claimURL).Render(url)
	}

	return header
}
