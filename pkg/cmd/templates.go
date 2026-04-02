package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yuin/goldmark"
	goldmarkast "github.com/yuin/goldmark/ast"
	goldmarktext "github.com/yuin/goldmark/text"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/useragent"
)

//
// Public functions
//

// WrappedInheritedFlagUsages returns a string containing the usage information
// for all flags which were inherited from parent commands, wrapped to the
// terminal's width.
func WrappedInheritedFlagUsages(cmd *cobra.Command) string {
	return cmd.InheritedFlags().FlagUsagesWrapped(getTerminalWidth())
}

// WrappedLocalFlagUsages returns a string containing the usage information
// for all flags specifically set in the current command, wrapped to the
// terminal's width.
func WrappedLocalFlagUsages(cmd *cobra.Command) string {
	return cmd.LocalFlags().FlagUsagesWrapped(getTerminalWidth())
}

// WrappedRequestParamsFlagUsages returns a string containing the usage
// information for all request parameters flags, i.e. flags used in operation
// commands to set values for request parameters.
//
// For enum parameters, the possible values are shown inline (e.g. --status
// complete|expired|open), truncated with "..." if they would exceed the
// terminal width. For other parameters, the API type is shown in angle
// brackets (e.g. --amount <integer>).
func WrappedRequestParamsFlagUsages(cmd *cobra.Command) string {
	var sb strings.Builder

	descIndent := strings.Repeat(" ", 10)
	termWidth := getTerminalWidth()

	cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		if _, ok := flag.Annotations["request"]; !ok {
			return
		}

		if enumVals, hasEnum := flag.Annotations["enum"]; hasEnum {
			const maxDisplayedEnumValues = 5
			prefix := fmt.Sprintf("      --%s ", flag.Name)

			if len(enumVals) <= maxDisplayedEnumValues {
				fmt.Fprintf(&sb, "%s%s\n", prefix, strings.Join(enumVals, "|"))
			} else {
				fmt.Fprintf(&sb, "%s%s|...\n", prefix, strings.Join(enumVals[:maxDisplayedEnumValues], "|"))
			}
		} else if apiType, ok := flag.Annotations["apitype"]; ok {
			typeName := apiType[0]
			switch typeName {
			case "array":
				fmt.Fprintf(&sb, "      --%s <%s>  [can be specified multiple times]\n", flag.Name, "string")
			case "boolean":
				fmt.Fprintf(&sb, "      --%s true|false\n", flag.Name)
			case "clearable_object":
				fmt.Fprintf(&sb, "      --%s=\"\"  (pass empty string to remove this field)\n", flag.Name)
			default:
				label := typeName
				if formatVals, hasFormat := flag.Annotations["format"]; hasFormat && len(formatVals) > 0 {
					label = formatVals[0]
				}
				fmt.Fprintf(&sb, "      --%s <%s>\n", flag.Name, label)
			}
		} else {
			fmt.Fprintf(&sb, "      --%s\n", flag.Name)
		}

		if flag.Usage != "" {
			rendered := renderMarkdown(flag.Usage, os.Stdout)
			fmt.Fprintf(&sb, "%s%s\n", descIndent, wrapText(rendered, termWidth, len(descIndent)))
		}
	})

	return sb.String()
}

// renderMarkdown parses s as inline CommonMark and returns a string with ANSI
// formatting applied: bold → ansi.Bold, italic → ansi.Italic, code spans →
// faint-colored `backticks`, links → underlined cyan + OSC 8 in TTY or
// "text (url)" in non-TTY.
func renderMarkdown(s string, w io.Writer) string {
	doc := goldmark.New().Parser().Parse(goldmarktext.NewReader([]byte(s)))
	var sb strings.Builder
	renderNode(doc, []byte(s), w, &sb)
	return strings.TrimSpace(sb.String())
}

func renderNode(n goldmarkast.Node, src []byte, w io.Writer, sb *strings.Builder) {
	switch node := n.(type) {
	case *goldmarkast.Text:
		sb.Write(node.Segment.Value(src))
		return
	case *goldmarkast.CodeSpan:
		var code strings.Builder
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*goldmarkast.Text); ok {
				code.Write(t.Segment.Value(src))
			}
		}
		color := ansi.Color(w)
		sb.WriteString(color.Sprintf(color.Bold(color.Blue(code.String()))))
		return
	case *goldmarkast.Emphasis:
		var inner strings.Builder
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			renderNode(c, src, w, &inner)
		}
		if node.Level == 2 {
			sb.WriteString(ansi.Bold(inner.String()))
		} else {
			sb.WriteString(ansi.Italic(inner.String()))
		}
		return
	case *goldmarkast.Link:
		var linkText strings.Builder
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			renderNode(c, src, w, &linkText)
		}
		url := string(node.Destination)
		color := ansi.Color(w)
		sb.WriteString(color.Sprintf(color.Bold(linkText.String())))
		sb.WriteByte(' ')
		sb.WriteString(ansi.Linkify(color.Sprintf(color.Faint(color.Underline(color.Cyan(url)))), url, w))
		return
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		renderNode(c, src, w, sb)
	}
}

// ansiEscRe matches CSI escape sequences (\x1b[...m) and OSC sequences
// (\x1b]...\x1b\) used by ANSI color codes and OSC 8 hyperlinks.
var ansiEscRe = regexp.MustCompile("\x1b(?:\\[[0-9;]*[a-zA-Z]|\\][^\x1b]*\x1b\\\\)")

// visibleLen returns the display width of s, ignoring ANSI escape bytes.
func visibleLen(s string) int {
	return utf8.RuneCountInString(ansiEscRe.ReplaceAllString(s, ""))
}

// ansiFields splits s on ASCII whitespace, treating ANSI escape sequences
// (including OSC sequences) as opaque — spaces inside an OSC sequence are
// not used as word-break points.
func ansiFields(s string) []string {
	var words []string
	var cur strings.Builder
	inOSC := false
	i := 0
	for i < len(s) {
		b := s[i]
		switch {
		case b == 0x1b && i+1 < len(s) && s[i+1] == ']':
			inOSC = true
			cur.WriteByte(b)
			i++
		case b == 0x1b && i+1 < len(s) && s[i+1] == '\\' && inOSC:
			inOSC = false
			cur.WriteByte(b)
			i++
		case b == 0x07 && inOSC:
			inOSC = false
			cur.WriteByte(b)
			i++
		case !inOSC && (b == ' ' || b == '\t' || b == '\n' || b == '\r'):
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
			i++
		default:
			cur.WriteByte(b)
			i++
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}

// wrapText word-wraps s to fit within width columns. Continuation lines are
// indented by indent spaces. ANSI escape sequences (including OSC 8 hyperlinks)
// are treated as zero-width for measurement purposes.
func wrapText(s string, width, indent int) string {
	words := ansiFields(s)
	if len(words) == 0 {
		return ""
	}

	prefix := strings.Repeat(" ", indent)
	lineWidth := width - indent
	if lineWidth < 20 {
		lineWidth = 20
	}

	var sb strings.Builder
	col := 0
	for i, word := range words {
		wlen := visibleLen(word)
		if i == 0 {
			sb.WriteString(word)
			col = wlen
		} else if col+1+wlen > lineWidth {
			sb.WriteString("\n")
			sb.WriteString(prefix)
			sb.WriteString(word)
			col = wlen
		} else {
			sb.WriteString(" ")
			sb.WriteString(word)
			col += 1 + wlen
		}
	}
	return sb.String()
}

// WrappedNonRequestParamsFlagUsages returns a string containing the usage
// information for all non-request parameters flags. The string is wrapped to
// the terminal's width.
func WrappedNonRequestParamsFlagUsages(cmd *cobra.Command) string {
	nonRequestParamsFlags := pflag.NewFlagSet("request", pflag.ExitOnError)

	cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		if _, ok := flag.Annotations["request"]; !ok {
			nonRequestParamsFlags.AddFlag(flag)
		}
	})

	return nonRequestParamsFlags.FlagUsagesWrapped(getTerminalWidth())
}

//
// Private functions
//

func isAIAgent() bool {
	return useragent.DetectAIAgent(os.Getenv) != ""
}

// AIAgentHelpAnnotationKey is the Cobra annotation key used to store
// per-command help text shown only when an AI agent is detected.
// Set it on any command via cmd.Annotations["ai_agent_help"] = "your text".
const AIAgentHelpAnnotationKey = "ai_agent_help"

func formatAgentGuidance(cmd *cobra.Command) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "\n\n%s\n", ansi.Bold("[Agent guidance]"))

	if extra, ok := cmd.Annotations[AIAgentHelpAnnotationKey]; ok && extra != "" {
		sb.WriteString(extra + "\n")
	}

	fmt.Fprintf(&sb, "  Use %s to pass your key non-interactively (or set %s).\n", ansi.Bold("--api-key"), ansi.Bold("STRIPE_API_KEY"))

	if cmd.Flags().Lookup("data") != nil {
		fmt.Fprintf(&sb, "  Use %s to set nested params, e.g. %s.\n", ansi.Bold("-d"), ansi.Italic(`-d "metadata[key]=value"`))
	}

	fmt.Fprintf(&sb, "  Run %s to quickly see all available commands.\n", ansi.Bold("stripe --map"))
	fmt.Fprintf(&sb, "  Run %s to discover all available API resources.\n", ansi.Bold("stripe resources"))
	fmt.Fprintf(&sb, "  Run %s to see operations and parameters for a resource.\n", ansi.Bold("stripe [resource] --help"))

	if cmd.Flags().Lookup("stripe-account") != nil {
		fmt.Fprintf(&sb, "  Use %s to make requests on behalf of connected accounts.", ansi.Bold("--stripe-account"))
	}

	return sb.String()
}

// aiAgentHelpTop renders agent guidance only for the root command (no parent).
// Used at the top of the root usage template.
func aiAgentHelpTop(cmd *cobra.Command) string {
	if !isAIAgent() || cmd.HasParent() {
		return ""
	}
	return formatAgentGuidance(cmd)
}

// aiAgentHelp renders agent guidance for non-root commands.
// Used in the pre-flags position of usage templates.
func aiAgentHelp(cmd *cobra.Command) string {
	if !isAIAgent() || !cmd.HasParent() {
		return ""
	}
	return formatAgentGuidance(cmd)
}

func getLogin(fs *afero.Fs, cfg *config.Config) string {
	// We're checking against the path because we don't initialize the config
	// at this point of execution.
	path := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	file := filepath.Join(path, "config.toml")

	exists, _ := afero.Exists(*fs, file)

	if !exists {
		return `
Before using the CLI, you'll need to login:

  $ stripe login

If you're working on multiple projects, you can run the login command with the
--project-name flag:

  $ stripe login --project-name rocket-rides`
	}

	return ""
}

func getUsageTemplate() string {
	return fmt.Sprintf(`%s{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{AIAgentHelpTop .}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{if (index .Annotations "get")}}

%s{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "webhooks")}}
  {{rpad $cmd.Name $cmd.NamePadding}} {{$cmd.Short}}{{end}}{{end}}

%s{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "stripe")}}
  {{rpad $cmd.Name $cmd.NamePadding}} {{$cmd.Short}}{{end}}{{end}}

%s
  {{rpad "charges" 29}} Make requests (capture, create, list, etc) on charges
  {{rpad "customers" 29}} Make requests (create, delete, list, etc) on customers
  {{rpad "payment_intents" 29}} Make requests (cancel, capture, confirm, etc) on payment intents
  {{rpad "..." 29}} %s
  {{rpad "v2" 29}} %s

%s
  {{rpad "get" 29}} Make GET requests to the Stripe API
  {{rpad "post" 29}} Make POST requests to the Stripe API
  {{rpad "delete" 29}} Make DELETE requests to the Stripe API

%s{{range $index, $cmd := .Commands}}{{if (not (or (index $.Annotations $cmd.Name) $cmd.Hidden))}}
  {{rpad $cmd.Name $cmd.NamePadding}} {{$cmd.Short}}{{end}}{{end}}{{else}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{AIAgentHelp .}}{{if .HasAvailableLocalFlags}}

%s
{{WrappedLocalFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{WrappedInheritedFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`,
		ansi.Bold("Usage:"),
		ansi.Bold("Aliases:"),
		ansi.Bold("Examples:"),
		ansi.Bold("Webhook commands:"),
		ansi.Bold("Stripe commands:"),
		ansi.Bold("Resource commands:"),
		ansi.Italic("To see more resource commands, run `stripe resources help`"),
		ansi.Italic("To see only v2 resource commands, run `stripe v2 help`"),
		ansi.Bold("API commands:"),
		ansi.Bold("Other commands:"),
		ansi.Bold("Available commands:"),
		ansi.Bold("Flags:"),
		ansi.Bold("Global flags:"),
	)
}

func getTerminalWidth() int {
	var width int

	width, _, err := term.GetSize(0)
	if err != nil {
		width = 80
	}

	return width
}

func init() {
	cobra.AddTemplateFunc("WrappedInheritedFlagUsages", WrappedInheritedFlagUsages)
	cobra.AddTemplateFunc("WrappedLocalFlagUsages", WrappedLocalFlagUsages)
	cobra.AddTemplateFunc("WrappedRequestParamsFlagUsages", WrappedRequestParamsFlagUsages)
	cobra.AddTemplateFunc("WrappedNonRequestParamsFlagUsages", WrappedNonRequestParamsFlagUsages)
	cobra.AddTemplateFunc("IsAIAgent", isAIAgent)
	cobra.AddTemplateFunc("AIAgentHelp", aiAgentHelp)
	cobra.AddTemplateFunc("AIAgentHelpTop", aiAgentHelpTop)
}
