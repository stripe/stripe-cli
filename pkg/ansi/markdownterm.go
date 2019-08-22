// nolint:golint
package ansi

import (
	"bytes"

	"github.com/russross/blackfriday"
)

// Markdown terminal renderer configuration options.
const (
	MDTERM_USE_ANSI = 1 << iota // use ANSI sequences
)

// MarkdownTerm is a type that implements the blackfriday.Renderer interface
// for terminal output.
//
// Note that it only supports a small subset of Markdown features. It should
// only be used for rendering documentation strings from Stripe's OpenAPI
// specification file.
//
// Do not create this directly, instead use the MarkdownTermRenderer function.
type MarkdownTerm struct {
	flags int // MDTERM_* options
}

// MarkdownTermRenderer creates and configures a MarkdownTerm object, which
// satisfies the Renderer interface.
//
// flags is a set of MDTERM_* options ORed together.
func MarkdownTermRenderer(flags int) blackfriday.Renderer {
	return &MarkdownTerm{flags: flags}
}

func (options *MarkdownTerm) GetFlags() int {
	return options.flags
}

// Block-level callbacks

func (options *MarkdownTerm) BlockCode(out *bytes.Buffer, text []byte, info string) {
	out.Write(text)
}

func (options *MarkdownTerm) BlockQuote(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (options *MarkdownTerm) BlockHtml(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (options *MarkdownTerm) Header(out *bytes.Buffer, text func() bool, level int, id string) {
	marker := out.Len()

	if !text() {
		out.Truncate(marker)
		return
	}
	out.WriteString("\n")
}

func (options *MarkdownTerm) HRule(out *bytes.Buffer) {
}

func (options *MarkdownTerm) List(out *bytes.Buffer, text func() bool, flags int) {
}

func (options *MarkdownTerm) ListItem(out *bytes.Buffer, text []byte, flags int) {
	out.WriteString("o ")
	out.Write(text)
}

func (options *MarkdownTerm) Paragraph(out *bytes.Buffer, text func() bool) {
	marker := out.Len()
	out.WriteString("\n")
	if !text() {
		out.Truncate(marker)
		return
	}
	out.WriteString("\n")
}

func (options *MarkdownTerm) Table(out *bytes.Buffer, header []byte, body []byte, columnData []int) {
}

func (options *MarkdownTerm) TableRow(out *bytes.Buffer, text []byte) {
}

func (options *MarkdownTerm) TableHeaderCell(out *bytes.Buffer, text []byte, align int) {
}

func (options *MarkdownTerm) TableCell(out *bytes.Buffer, text []byte, align int) {
}

func (options *MarkdownTerm) Footnotes(out *bytes.Buffer, text func() bool) {
}

func (options *MarkdownTerm) FootnoteItem(out *bytes.Buffer, name, text []byte, flags int) {
}

func (options *MarkdownTerm) TitleBlock(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

// Span-level callbacks

func (options *MarkdownTerm) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	out.Write(link)
}

func (options *MarkdownTerm) CodeSpan(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (options *MarkdownTerm) DoubleEmphasis(out *bytes.Buffer, text []byte) {
	if options.flags&MDTERM_USE_ANSI != 0 {
		out.WriteString(Bold(string(text)))
	} else {
		out.Write(text)
	}
}

func (options *MarkdownTerm) Emphasis(out *bytes.Buffer, text []byte) {
	if options.flags&MDTERM_USE_ANSI != 0 {
		out.WriteString(Italic(string(text)))
	} else {
		out.Write(text)
	}
}

func (options *MarkdownTerm) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
}

func (options *MarkdownTerm) LineBreak(out *bytes.Buffer) {
	out.WriteString("\n")
}

func (options *MarkdownTerm) Link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	// We're not using Linkify here because pager programs like less don't
	// support the hyperlink ANSI sequence.
	out.Write(content)
	out.WriteString(" [")
	out.Write(link)
	out.WriteString("]")
}

func (options *MarkdownTerm) RawHtmlTag(out *bytes.Buffer, tag []byte) {
}

func (options *MarkdownTerm) TripleEmphasis(out *bytes.Buffer, text []byte) {
	if options.flags&MDTERM_USE_ANSI != 0 {
		out.WriteString(Bold(Italic(string(text))))
	} else {
		out.Write(text)
	}
}

func (options *MarkdownTerm) StrikeThrough(out *bytes.Buffer, text []byte) {
	if options.flags&MDTERM_USE_ANSI != 0 {
		out.WriteString(StrikeThrough(string(text)))
	} else {
		out.Write(text)
	}
}

func (options *MarkdownTerm) FootnoteRef(out *bytes.Buffer, ref []byte, id int) {
}

// Low-level callbacks

func (options *MarkdownTerm) Entity(out *bytes.Buffer, entity []byte) {
	out.Write(entity)
}

func (options *MarkdownTerm) NormalText(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

// Header and footer

func (options *MarkdownTerm) DocumentHeader(out *bytes.Buffer) {
}

func (options *MarkdownTerm) DocumentFooter(out *bytes.Buffer) {
}
