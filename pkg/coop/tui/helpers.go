package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/open"
)

var openBrowserFn = open.Browser

func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := openBrowserFn(url); err != nil {
			return statusMsg{message: fmt.Sprintf("Could not open browser: %s", err), ttl: 5 * time.Second}
		}
		return nil
	}
}

func (m Model) sandboxClaimLink() string {
	if m.session == nil || !m.session.UsedSandbox {
		return ""
	}
	return m.sandboxClaimURL
}

func (m Model) contentWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}

func (m Model) pinFooter(content, footer string) string {
	if m.width > 0 {
		content = clampLines(content, m.width)
		footer = clampLines(footer, m.width)
	}
	contentH := strings.Count(content, "\n") + 1
	footerH := strings.Count(footer, "\n") + 1
	if m.height > 0 {
		pad := m.height - contentH - footerH - 1
		if pad < 0 {
			pad = 0
		}
		if pad > 0 {
			content += strings.Repeat("\n", pad)
		}
	}
	return content + "\n" + footer
}

func clampLines(s string, width int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = lipgloss.NewStyle().MaxWidth(width).Render(line)
		}
	}
	return strings.Join(lines, "\n")
}

func wordWrap(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	return lipgloss.Wrap(s, width, " ")
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
