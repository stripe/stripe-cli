package markdown

import (
	"charm.land/glamour/v2/ansi"
)

const (
	defaultListIndent      = 2
	defaultListLevelIndent = 4
	defaultMargin          = 2
)

// DarkStyleConfig is our default style for dark terminal backgrounds.
var DarkStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "\n",
			BlockSuffix: "\n",
			Color:       strPtr("252"),
		},
		Margin: uintPtr(defaultMargin),
	},
	BlockQuote: ansi.StyleBlock{
		Indent:      uintPtr(1),
		IndentToken: strPtr("│ "),
	},
	List: ansi.StyleList{
		LevelIndent: defaultListIndent,
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       strPtr("39"),
			Bold:        boolPtr(true),
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          " ",
			Suffix:          " ",
			Color:           strPtr("255"),
			BackgroundColor: strPtr("63"),
			Bold:            boolPtr(true),
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "## ",
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "### ",
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "#### ",
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "##### ",
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "###### ",
			Color:  strPtr("35"),
			Bold:   boolPtr(false),
		},
	},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut: boolPtr(true),
	},
	Emph: ansi.StylePrimitive{
		Italic: boolPtr(true),
	},
	Strong: ansi.StylePrimitive{
		Bold: boolPtr(true),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:  strPtr("240"),
		Format: "\n--------\n",
	},
	Item: ansi.StylePrimitive{
		BlockPrefix: "• ",
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix: ". ",
	},
	Task: ansi.StyleTask{
		Ticked:   "[✓] ",
		Unticked: "[ ] ",
	},
	Link: ansi.StylePrimitive{
		Color:     strPtr("30"),
		Underline: boolPtr(true),
	},
	LinkText: ansi.StylePrimitive{
		Color: strPtr("35"),
		Bold:  boolPtr(true),
	},
	Image: ansi.StylePrimitive{
		Color:     strPtr("212"),
		Underline: boolPtr(true),
	},
	ImageText: ansi.StylePrimitive{
		Color:  strPtr("243"),
		Format: "Image: {{.text}} →",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          " ",
			Suffix:          " ",
			Color:           strPtr("203"),
			BackgroundColor: strPtr("236"),
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strPtr("244"),
			},
			Margin: uintPtr(defaultMargin),
		},
		Chroma: &ansi.Chroma{
			Text:                ansi.StylePrimitive{Color: strPtr("#C4C4C4")},
			Error:               ansi.StylePrimitive{Color: strPtr("#F1F1F1"), BackgroundColor: strPtr("#F05B5B")},
			Comment:             ansi.StylePrimitive{Color: strPtr("#676767")},
			CommentPreproc:      ansi.StylePrimitive{Color: strPtr("#FF875F")},
			Keyword:             ansi.StylePrimitive{Color: strPtr("#00AAFF")},
			KeywordReserved:     ansi.StylePrimitive{Color: strPtr("#FF5FD2")},
			KeywordNamespace:    ansi.StylePrimitive{Color: strPtr("#FF5F87")},
			KeywordType:         ansi.StylePrimitive{Color: strPtr("#6E6ED8")},
			Operator:            ansi.StylePrimitive{Color: strPtr("#EF8080")},
			Punctuation:         ansi.StylePrimitive{Color: strPtr("#E8E8A8")},
			Name:                ansi.StylePrimitive{Color: strPtr("#C4C4C4")},
			NameBuiltin:         ansi.StylePrimitive{Color: strPtr("#FF8EC7")},
			NameTag:             ansi.StylePrimitive{Color: strPtr("#B083EA")},
			NameAttribute:       ansi.StylePrimitive{Color: strPtr("#7A7AE6")},
			NameClass:           ansi.StylePrimitive{Color: strPtr("#F1F1F1"), Underline: boolPtr(true), Bold: boolPtr(true)},
			NameDecorator:       ansi.StylePrimitive{Color: strPtr("#FFFF87")},
			NameFunction:        ansi.StylePrimitive{Color: strPtr("#00D787")},
			LiteralNumber:       ansi.StylePrimitive{Color: strPtr("#6EEFC0")},
			LiteralString:       ansi.StylePrimitive{Color: strPtr("#C69669")},
			LiteralStringEscape: ansi.StylePrimitive{Color: strPtr("#AFFFD7")},
			GenericDeleted:      ansi.StylePrimitive{Color: strPtr("#FD5B5B")},
			GenericEmph:         ansi.StylePrimitive{Italic: boolPtr(true)},
			GenericInserted:     ansi.StylePrimitive{Color: strPtr("#00D787")},
			GenericStrong:       ansi.StylePrimitive{Bold: boolPtr(true)},
			GenericSubheading:   ansi.StylePrimitive{Color: strPtr("#777777")},
			Background:          ansi.StylePrimitive{BackgroundColor: strPtr("#373737")},
		},
	},
	Table: ansi.StyleTable{},
	DefinitionDescription: ansi.StylePrimitive{
		BlockPrefix: "\n🠶 ",
	},
}

// LightStyleConfig is our default style for light terminal backgrounds.
var LightStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "\n",
			BlockSuffix: "\n",
			Color:       strPtr("234"),
		},
		Margin: uintPtr(defaultMargin),
	},
	BlockQuote: ansi.StyleBlock{
		Indent:      uintPtr(1),
		IndentToken: strPtr("│ "),
	},
	List: ansi.StyleList{
		LevelIndent: defaultListIndent,
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       strPtr("27"),
			Bold:        boolPtr(true),
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          " ",
			Suffix:          " ",
			Color:           strPtr("255"),
			BackgroundColor: strPtr("63"),
			Bold:            boolPtr(true),
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "## ",
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "### ",
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "#### ",
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "##### ",
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "###### ",
			Bold:   boolPtr(false),
		},
	},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut: boolPtr(true),
	},
	Emph: ansi.StylePrimitive{
		Italic: boolPtr(true),
	},
	Strong: ansi.StylePrimitive{
		Bold: boolPtr(true),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:  strPtr("249"),
		Format: "\n--------\n",
	},
	Item: ansi.StylePrimitive{
		BlockPrefix: "• ",
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix: ". ",
	},
	Task: ansi.StyleTask{
		Ticked:   "[✓] ",
		Unticked: "[ ] ",
	},
	Link: ansi.StylePrimitive{
		Color:     strPtr("36"),
		Underline: boolPtr(true),
	},
	LinkText: ansi.StylePrimitive{
		Color: strPtr("29"),
		Bold:  boolPtr(true),
	},
	Image: ansi.StylePrimitive{
		Color:     strPtr("205"),
		Underline: boolPtr(true),
	},
	ImageText: ansi.StylePrimitive{
		Color:  strPtr("243"),
		Format: "Image: {{.text}} →",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          " ",
			Suffix:          " ",
			Color:           strPtr("203"),
			BackgroundColor: strPtr("254"),
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strPtr("242"),
			},
			Margin: uintPtr(defaultMargin),
		},
		Chroma: &ansi.Chroma{
			Text:                ansi.StylePrimitive{Color: strPtr("#2A2A2A")},
			Error:               ansi.StylePrimitive{Color: strPtr("#F1F1F1"), BackgroundColor: strPtr("#FF5555")},
			Comment:             ansi.StylePrimitive{Color: strPtr("#8D8D8D")},
			CommentPreproc:      ansi.StylePrimitive{Color: strPtr("#FF875F")},
			Keyword:             ansi.StylePrimitive{Color: strPtr("#279EFC")},
			KeywordReserved:     ansi.StylePrimitive{Color: strPtr("#FF5FD2")},
			KeywordNamespace:    ansi.StylePrimitive{Color: strPtr("#FB406F")},
			KeywordType:         ansi.StylePrimitive{Color: strPtr("#7049C2")},
			Operator:            ansi.StylePrimitive{Color: strPtr("#FF2626")},
			Punctuation:         ansi.StylePrimitive{Color: strPtr("#FA7878")},
			NameBuiltin:         ansi.StylePrimitive{Color: strPtr("#0A1BB1")},
			NameTag:             ansi.StylePrimitive{Color: strPtr("#581290")},
			NameAttribute:       ansi.StylePrimitive{Color: strPtr("#8362CB")},
			NameClass:           ansi.StylePrimitive{Color: strPtr("#212121"), Underline: boolPtr(true), Bold: boolPtr(true)},
			NameConstant:        ansi.StylePrimitive{Color: strPtr("#581290")},
			NameDecorator:       ansi.StylePrimitive{Color: strPtr("#A3A322")},
			NameFunction:        ansi.StylePrimitive{Color: strPtr("#019F57")},
			LiteralNumber:       ansi.StylePrimitive{Color: strPtr("#22CCAE")},
			LiteralString:       ansi.StylePrimitive{Color: strPtr("#7E5B38")},
			LiteralStringEscape: ansi.StylePrimitive{Color: strPtr("#00AEAE")},
			GenericDeleted:      ansi.StylePrimitive{Color: strPtr("#FD5B5B")},
			GenericEmph:         ansi.StylePrimitive{Italic: boolPtr(true)},
			GenericInserted:     ansi.StylePrimitive{Color: strPtr("#00D787")},
			GenericStrong:       ansi.StylePrimitive{Bold: boolPtr(true)},
			GenericSubheading:   ansi.StylePrimitive{Color: strPtr("#777777")},
			Background:          ansi.StylePrimitive{BackgroundColor: strPtr("#F5F5F5")},
		},
	},
	Table: ansi.StyleTable{},
	DefinitionDescription: ansi.StylePrimitive{
		BlockPrefix: "\n🠶 ",
	},
}

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }
func uintPtr(u uint) *uint    { return &u }
