package ansi

import (
	"fmt"
	"io"
	"strings"

	"github.com/logrusorgru/aurora"
)

// RenderDiff prints a colorized, line-numbered unified diff to w.
// path is shown as a header. oldContent/newContent are full file contents.
// Colors respect --color flag, CLICOLOR, and TTY detection via Color(w).
func RenderDiff(w io.Writer, path, oldContent, newContent string) {
	if oldContent == newContent {
		return
	}

	color := Color(w)

	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	// Print header
	if oldContent == "" && newContent != "" {
		fmt.Fprintf(w, "\n%s %s:\n\n", path, color.Sprintf(color.Faint("(new file)")))
	} else {
		fmt.Fprintf(w, "\n%s:\n\n", path)
	}

	// Find common prefix (identical lines from start)
	prefixLen := 0
	for prefixLen < len(oldLines) && prefixLen < len(newLines) && oldLines[prefixLen] == newLines[prefixLen] {
		prefixLen++
	}

	// Find common suffix (identical lines from end, not overlapping prefix)
	suffixLen := 0
	for suffixLen < len(oldLines)-prefixLen && suffixLen < len(newLines)-prefixLen &&
		oldLines[len(oldLines)-1-suffixLen] == newLines[len(newLines)-1-suffixLen] {
		suffixLen++
	}

	oldMiddleStart := prefixLen
	oldMiddleEnd := len(oldLines) - suffixLen
	newMiddleStart := prefixLen
	newMiddleEnd := len(newLines) - suffixLen

	hunks := buildHunks(oldLines, newLines, oldMiddleStart, oldMiddleEnd, newMiddleStart, newMiddleEnd)

	useColors := shouldUseColors(w)

	for i, h := range hunks {
		if i > 0 {
			fmt.Fprintf(w, "%s\n", color.Sprintf(color.Faint("        ···")))
		}
		renderHunk(w, color, useColors, oldLines, newLines, h, 3)
	}

	fmt.Fprintln(w)
}

// diffHunk represents a contiguous changed region.
type diffHunk struct {
	oldStart int // index into oldLines where removed lines start
	oldEnd   int // exclusive end
	newStart int // index into newLines where added lines start
	newEnd   int // exclusive end
}

// buildHunks identifies individual hunks within the middle changed region.
// When the middle contains runs of 7+ identical lines, the region is split
// into separate hunks so the output shows ··· between distant changes.
func buildHunks(oldLines, newLines []string, oldStart, oldEnd, newStart, newEnd int) []diffHunk {
	oldMiddle := oldLines[oldStart:oldEnd]
	newMiddle := newLines[newStart:newEnd]

	if len(oldMiddle) == 0 && len(newMiddle) == 0 {
		return nil
	}

	// For purely added or purely removed, there are no common inner lines to split on.
	if len(oldMiddle) == 0 || len(newMiddle) == 0 {
		return []diffHunk{{oldStart: oldStart, oldEnd: oldEnd, newStart: newStart, newEnd: newEnd}}
	}

	// Look for runs of common lines within the middle region that are long
	// enough (7+) to justify splitting into separate hunks.
	// Use a simple LCS-like scan: match identical lines at the same offset.
	type matchRun struct {
		oldIdx, newIdx, length int
	}

	var runs []matchRun
	oi, ni := 0, 0
	for oi < len(oldMiddle) && ni < len(newMiddle) {
		if oldMiddle[oi] == newMiddle[ni] {
			start := oi
			nStart := ni
			for oi < len(oldMiddle) && ni < len(newMiddle) && oldMiddle[oi] == newMiddle[ni] {
				oi++
				ni++
			}
			if oi-start >= 7 {
				runs = append(runs, matchRun{oldIdx: start, newIdx: nStart, length: oi - start})
			}
		} else {
			// Advance whichever side is "behind"
			if oi < ni {
				oi++
			} else {
				ni++
			}
		}
	}

	if len(runs) == 0 {
		return []diffHunk{{oldStart: oldStart, oldEnd: oldEnd, newStart: newStart, newEnd: newEnd}}
	}

	// Split the middle region around the runs
	var hunks []diffHunk
	curOldStart, curNewStart := 0, 0

	for _, run := range runs {
		// Everything before this run is a hunk
		if curOldStart < run.oldIdx || curNewStart < run.newIdx {
			hunks = append(hunks, diffHunk{
				oldStart: oldStart + curOldStart,
				oldEnd:   oldStart + run.oldIdx,
				newStart: newStart + curNewStart,
				newEnd:   newStart + run.newIdx,
			})
		}
		curOldStart = run.oldIdx + run.length
		curNewStart = run.newIdx + run.length
	}

	// Remaining after last run
	if curOldStart < len(oldMiddle) || curNewStart < len(newMiddle) {
		hunks = append(hunks, diffHunk{
			oldStart: oldStart + curOldStart,
			oldEnd:   oldEnd,
			newStart: newStart + curNewStart,
			newEnd:   newEnd,
		})
	}

	return hunks
}

// renderHunk renders a single hunk with surrounding context lines.
func renderHunk(w io.Writer, color aurora.Aurora, useColors bool, oldLines, newLines []string, h diffHunk, contextSize int) {
	// Leading context: use lines from whichever side has them at the common prefix position
	contextStart := max(0, min(h.oldStart, h.newStart)-contextSize)

	// Use newLines for context when available (they represent the final state),
	// falling back to oldLines for the leading context region.
	contextSource := newLines
	if len(newLines) == 0 {
		contextSource = oldLines
	}

	// Format: left-aligned number in a fixed-width gutter, then prefix+text.
	// The +/- prefix replaces the second space in context lines, keeping
	// the text aligned while visually marking the change.
	// Context:  "  123  line text"
	// Removed:  "  123 -line text"   (dark red background)
	// Added:    "  123 +line text"   (dark green background)

	changeStart := min(h.oldStart, h.newStart)
	for i := contextStart; i < changeStart && i < len(contextSource); i++ {
		fmt.Fprintf(w, "%5d  %s\n", i+1, contextSource[i])
	}

	// Removed lines — very dark red background covering full line including gutter,
	// desaturated red line number (index 131: #af5f5f)
	for i := h.oldStart; i < h.oldEnd; i++ {
		renderColoredLine(w, useColors, i+1, "-", oldLines[i], diffColorRemoved, 131)
	}

	// Added lines — very dark green background covering full line including gutter,
	// bright green line number (index 34: #00af00)
	for i := h.newStart; i < h.newEnd; i++ {
		renderColoredLine(w, useColors, i+1, "+", newLines[i], diffColorAdded, 34)
	}

	// Trailing context (use new file lines and line numbers)
	trailStart := h.newEnd
	trailEnd := min(len(newLines), trailStart+contextSize)
	for i := trailStart; i < trailEnd; i++ {
		fmt.Fprintf(w, "%5d  %s\n", i+1, newLines[i])
	}
}

// diffColor holds the true-color RGB for a diff line background.
type diffColor struct {
	r, g, b uint8
}

var (
	// Very dark green (#002200) and very dark red (#220000) for diff backgrounds.
	// True-color (24-bit) ANSI escapes, darker than the 256-color palette allows.
	diffColorAdded   = diffColor{0x00, 0x22, 0x00}
	diffColorRemoved = diffColor{0x22, 0x00, 0x00}
)

// renderColoredLine writes a full-width background-colored diff line using
// raw ANSI escapes. The background covers the entire line including the gutter.
// The line number foreground is set without resetting, so the background persists.
func renderColoredLine(w io.Writer, useColors bool, lineNum int, prefix, text string, bg diffColor, fgIndex uint8) {
	if useColors {
		// Set background (24-bit), then foreground (256-color) for line number,
		// print number, reset foreground to default, print prefix+text, then full reset.
		fmt.Fprintf(w, "\x1b[48;2;%d;%d;%dm\x1b[38;5;%dm%5d %s\x1b[39m%s\x1b[0m\n",
			bg.r, bg.g, bg.b, fgIndex, lineNum, prefix, text)
	} else {
		fmt.Fprintf(w, "%5d %s%s\n", lineNum, prefix, text)
	}
}

// splitLines splits a string into lines, removing a trailing empty element
// caused by a final newline.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
