package ansi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

func disableColors(t *testing.T) {
	t.Helper()
	ansi.DisableColors = true
	t.Cleanup(func() { ansi.DisableColors = false })
}

func TestRenderDiff_NewFile(t *testing.T) {
	disableColors(t)

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "~/.zshrc", "", "line one\nline two\nline three\n")

	output := buf.String()
	assert.Contains(t, output, "~/.zshrc (new file):")
	assert.Contains(t, output, "    1 +line one")
	assert.Contains(t, output, "    2 +line two")
	assert.Contains(t, output, "    3 +line three")
	assert.True(t, strings.HasSuffix(output, "\n"), "output should end with newline")
}

func TestRenderDiff_LinesAddedAtEnd(t *testing.T) {
	disableColors(t)

	oldContent := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	newContent := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", oldContent, newContent)

	output := buf.String()
	assert.Contains(t, output, "file.txt:")
	assert.NotContains(t, output, "(new file)")
	// Should show 3 lines of context before the additions
	assert.Contains(t, output, "    3  line 3")
	assert.Contains(t, output, "    4  line 4")
	assert.Contains(t, output, "    5  line 5")
	// Then the added lines (number, space, +, text)
	assert.Contains(t, output, "    6 +line 6")
	assert.Contains(t, output, "    7 +line 7")
}

func TestRenderDiff_LinesRemoved(t *testing.T) {
	disableColors(t)

	oldContent := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\n"
	newContent := "line 1\nline 2\nline 3\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", oldContent, newContent)

	output := buf.String()
	assert.Contains(t, output, "file.txt:")
	// Should show context before removed lines
	assert.Contains(t, output, "    1  line 1")
	assert.Contains(t, output, "    2  line 2")
	assert.Contains(t, output, "    3  line 3")
	// Then the removed lines (number, space, -, text)
	assert.Contains(t, output, "    4 -line 4")
	assert.Contains(t, output, "    5 -line 5")
	assert.Contains(t, output, "    6 -line 6")
}

func TestRenderDiff_LinesRemovedFromMiddle(t *testing.T) {
	disableColors(t)

	oldContent := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\n"
	newContent := "line 1\nline 2\nline 3\nline 6\nline 7\nline 8\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", oldContent, newContent)

	output := buf.String()
	// Should show 3 lines of context before
	assert.Contains(t, output, "    1  line 1")
	assert.Contains(t, output, "    2  line 2")
	assert.Contains(t, output, "    3  line 3")
	// Then removed lines
	assert.Contains(t, output, "    4 -line 4")
	assert.Contains(t, output, "    5 -line 5")
	// Then 3 lines of context after (new-file line numbers)
	assert.Contains(t, output, "    4  line 6")
	assert.Contains(t, output, "    5  line 7")
	assert.Contains(t, output, "    6  line 8")
}

func TestRenderDiff_Replacement(t *testing.T) {
	disableColors(t)

	oldContent := "line 1\nline 2\nold line 3\nold line 4\nline 5\nline 6\n"
	newContent := "line 1\nline 2\nnew line 3\nnew line 4\nline 5\nline 6\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", oldContent, newContent)

	output := buf.String()
	// Should show removal of old lines
	assert.Contains(t, output, "    3 -old line 3")
	assert.Contains(t, output, "    4 -old line 4")
	// And addition of new lines
	assert.Contains(t, output, "    3 +new line 3")
	assert.Contains(t, output, "    4 +new line 4")
	// With proper context
	assert.Contains(t, output, "    2  line 2")
	assert.Contains(t, output, "    5  line 5")
}

func TestRenderDiff_MultipleHunks(t *testing.T) {
	disableColors(t)

	// Create content with changes at start and end, separated by 7+ unchanged lines
	oldContent := "old line 1\nold line 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10\nold line 11\nold line 12\n"
	newContent := "new line 1\nnew line 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10\nnew line 11\nnew line 12\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", oldContent, newContent)

	output := buf.String()
	// Should show first hunk
	assert.Contains(t, output, "    1 -old line 1")
	assert.Contains(t, output, "    2 -old line 2")
	assert.Contains(t, output, "    1 +new line 1")
	assert.Contains(t, output, "    2 +new line 2")
	// Should show separator
	assert.Contains(t, output, "        ···")
	// Should show second hunk
	assert.Contains(t, output, "   11 -old line 11")
	assert.Contains(t, output, "   12 -old line 12")
	assert.Contains(t, output, "   11 +new line 11")
	assert.Contains(t, output, "   12 +new line 12")
}

func TestRenderDiff_NoChange(t *testing.T) {
	disableColors(t)

	content := "line 1\nline 2\nline 3\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", content, content)

	output := buf.String()
	// When there's no change, should output nothing
	assert.Empty(t, output)
}

func TestRenderDiff_EmptyToEmpty(t *testing.T) {
	disableColors(t)

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", "", "")

	output := buf.String()
	// Both empty means no change
	assert.Empty(t, output)
}

func TestRenderDiff_ContextCappingAtStart(t *testing.T) {
	disableColors(t)

	// Change on line 1 - should show only available context (none before)
	oldContent := "old line 1\nline 2\nline 3\nline 4\nline 5\n"
	newContent := "new line 1\nline 2\nline 3\nline 4\nline 5\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", oldContent, newContent)

	output := buf.String()
	// Should show the change immediately without 3 lines of context before
	assert.Contains(t, output, "    1 -old line 1")
	assert.Contains(t, output, "    1 +new line 1")
	// Should show 3 lines of context after
	assert.Contains(t, output, "    2  line 2")
	assert.Contains(t, output, "    3  line 3")
	assert.Contains(t, output, "    4  line 4")
}

func TestRenderDiff_ContextCappingAtStartWithTwoLines(t *testing.T) {
	disableColors(t)

	// Change on line 3 - should show only 2 lines of context before
	oldContent := "line 1\nline 2\nold line 3\nline 4\nline 5\nline 6\n"
	newContent := "line 1\nline 2\nnew line 3\nline 4\nline 5\nline 6\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", oldContent, newContent)

	output := buf.String()
	// Should show 2 lines of context before (not 3, because we're at line 3)
	assert.Contains(t, output, "    1  line 1")
	assert.Contains(t, output, "    2  line 2")
	assert.Contains(t, output, "    3 -old line 3")
	assert.Contains(t, output, "    3 +new line 3")
	// Should show 3 lines of context after
	assert.Contains(t, output, "    4  line 4")
	assert.Contains(t, output, "    5  line 5")
	assert.Contains(t, output, "    6  line 6")
}

func TestRenderDiff_ContextCappingAtEnd(t *testing.T) {
	disableColors(t)

	// Change on last line - should show only available context (none after)
	oldContent := "line 1\nline 2\nline 3\nline 4\nold line 5\n"
	newContent := "line 1\nline 2\nline 3\nline 4\nnew line 5\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", oldContent, newContent)

	output := buf.String()
	// Should show 3 lines of context before
	assert.Contains(t, output, "    2  line 2")
	assert.Contains(t, output, "    3  line 3")
	assert.Contains(t, output, "    4  line 4")
	// Then the change
	assert.Contains(t, output, "    5 -old line 5")
	assert.Contains(t, output, "    5 +new line 5")
	// No context after (it's the last line)
}

func TestRenderDiff_LineNumberAlignment(t *testing.T) {
	disableColors(t)

	// Test that line numbers are right-aligned in a 5-char gutter
	oldContent := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10\n"
	newContent := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10\nline 11\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", oldContent, newContent)

	output := buf.String()
	// Single-digit line numbers should be right-aligned
	assert.Contains(t, output, "    8  line 8")
	assert.Contains(t, output, "    9  line 9")
	// Double-digit line numbers should be right-aligned
	assert.Contains(t, output, "   10  line 10")
	assert.Contains(t, output, "   11 +line 11")
}

func TestRenderDiff_TrailingNewline(t *testing.T) {
	disableColors(t)

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", "", "line 1\n")

	output := buf.String()
	// Should end with trailing newline
	assert.True(t, strings.HasSuffix(output, "\n\n"), "output should end with two newlines (one from last line, one trailing)")
}

func TestRenderDiff_ComplexReplacement(t *testing.T) {
	disableColors(t)

	// Replace multiple lines with different number of lines
	oldContent := "line 1\nline 2\nold line 3\nold line 4\nold line 5\nline 6\nline 7\n"
	newContent := "line 1\nline 2\nnew line 3\nnew line 4\nline 6\nline 7\n"

	var buf bytes.Buffer
	ansi.RenderDiff(&buf, "file.txt", oldContent, newContent)

	output := buf.String()
	// Should show context
	assert.Contains(t, output, "    2  line 2")
	// Should show all removed lines
	assert.Contains(t, output, "    3 -old line 3")
	assert.Contains(t, output, "    4 -old line 4")
	assert.Contains(t, output, "    5 -old line 5")
	// Should show all added lines
	assert.Contains(t, output, "    3 +new line 3")
	assert.Contains(t, output, "    4 +new line 4")
	// Should show context after (new-file line numbers: line 6 is now at position 5)
	assert.Contains(t, output, "    5  line 6")
	assert.Contains(t, output, "    6  line 7")
}
