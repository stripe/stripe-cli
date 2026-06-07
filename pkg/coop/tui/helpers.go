package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var openBrowserFn = openBrowserDefault
var copyTextFn = copyTextDefault

func openBrowser(url string) {
	openBrowserFn(url)
}

func copyText(text string) error {
	return copyTextFn(text)
}

func openBrowserDefault(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start() //nolint:gosec
	case "linux":
		exec.Command("xdg-open", url).Start() //nolint:gosec
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start() //nolint:gosec
	}
}

func copyTextDefault(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy") //nolint:gosec
	case "linux":
		cmd = exec.Command("wl-copy") //nolint:gosec
	case "windows":
		cmd = exec.Command("clip") //nolint:gosec
	default:
		return fmt.Errorf("clipboard unsupported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
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
	if width <= 0 || len(s) <= width {
		return s
	}
	var result strings.Builder
	line := ""
	for _, word := range strings.Fields(s) {
		switch {
		case line == "":
			line = word
		case len(line)+1+len(word) > width:
			result.WriteString(line + "\n")
			line = word
		default:
			line += " " + word
		}
	}
	if line != "" {
		result.WriteString(line)
	}
	return result.String()
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
