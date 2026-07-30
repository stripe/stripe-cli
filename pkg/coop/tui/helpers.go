package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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

// truncatePath shortens a file path to fit, keeping the filename and line range
// — the parts that identify the change — and eliding directories in the middle.
// wordWrap cannot help here: paths contain no spaces, so they overflow rather
// than wrap.
func truncatePath(path string, width int) string {
	if width <= 0 || lipgloss.Width(path) <= width {
		return path
	}
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ansi.Truncate(path, width, "…")
	}
	tail := path[idx+1:]
	// Keep the head only if the elision leaves room for a recognizable prefix.
	if lipgloss.Width(tail)+2 >= width {
		return ansi.Truncate(tail, width, "…")
	}
	head := ansi.Truncate(path[:idx], width-lipgloss.Width(tail)-2, "")
	return head + "…/" + tail
}

func clampLines(s string, width int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			// Truncate with an indicator: MaxWidth cuts silently, so a clipped
			// title is indistinguishable from one that simply ends there.
			lines[i] = ansi.Truncate(line, width, "…")
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
