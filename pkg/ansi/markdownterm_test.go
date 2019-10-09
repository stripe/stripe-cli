package ansi

import (
	"os"
	"testing"

	"github.com/russross/blackfriday"

	"github.com/stretchr/testify/require"
)

func render(s string, useAnsi bool) string {
	flags := 0
	if useAnsi {
		flags |= MDTERM_USE_ANSI
	}

	r := MarkdownTermRenderer(flags)

	return string(blackfriday.Markdown([]byte(s), r, blackfriday.EXTENSION_STRIKETHROUGH))
}

func TestMain(m *testing.M) {
	ForceColors = true
	code := m.Run()
	ForceColors = false

	os.Exit(code)
}

func TestMarkdownTermRenderer(t *testing.T) {
	r := MarkdownTermRenderer(MDTERM_USE_ANSI)
	require.Equal(t, MDTERM_USE_ANSI, r.GetFlags())
}

func TestMarkdownTerm_Header(t *testing.T) {
	require.Equal(t, "Foo\n", render("# Foo\n", true))
	require.Equal(t, "Foo\n", render("# Foo\n", false))
	require.Equal(t, "", render("", true))
	require.Equal(t, "", render("", false))
}

func TestMarkdownTerm_Paragraph(t *testing.T) {
	require.Equal(t, "\nFoo.\n\nBar.\n", render("Foo.\n\nBar.", true))
	require.Equal(t, "\nFoo.\n\nBar.\n", render("Foo.\n\nBar.", false))
	require.Equal(t, "", render("", true))
	require.Equal(t, "", render("", false))
}

func TestMarkdownTerm_DoubleEmphasis(t *testing.T) {
	require.Equal(t, "\n\x1b[1mfoo\x1b[0m\n", render("**foo**", true))
	require.Equal(t, "\nfoo\n", render("**foo**", false))
}

func TestMarkdownTerm_Emphasis(t *testing.T) {
	require.Equal(t, "\n\x1b[3mfoo\x1b[0m\n", render("*foo*", true))
	require.Equal(t, "\nfoo\n", render("*foo*", false))
}

func TestMarkdownTerm_Link(t *testing.T) {
	require.Equal(t, "\nfoo [https://example.com/foo]\n", render("[foo](https://example.com/foo)", true))
	require.Equal(t, "\nfoo [https://example.com/foo]\n", render("[foo](https://example.com/foo)", false))
}

func TestMarkdownTerm_TripleEmphasis(t *testing.T) {
	require.Equal(t, "\n\x1b[1m\x1b[3mfoo\x1b[0m\x1b[0m\n", render("***foo***", true))
	require.Equal(t, "\nfoo\n", render("***foo***", false))
}

func TestMarkdownTerm_StrikeThrough(t *testing.T) {
	require.Equal(t, "\n\x1b[9mfoo\x1b[0m\n", render("~~foo~~", true))
	require.Equal(t, "\nfoo\n", render("~~foo~~", false))
}
