package resource

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/russross/blackfriday"

	"github.com/kr/text"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/spec"
)

//
// Public types
//

// OperationCmd represents operation commands. Operation commands are nested
// under resource commands and represent a specific API operation for that
// resource.
//
// Examples of operations: `create`, `retrieve` (standard CRUD methods),
// `capture` (custom method for the `charges` resource).
type OperationCmd struct {
	*requests.Base

	Name      string
	HTTPVerb  string
	Path      string
	URLParams []string
}

func (oc *OperationCmd) runOperationCmd(cmd *cobra.Command, args []string) error {
	apiKey, err := oc.Profile.GetAPIKey(oc.Livemode)
	if err != nil {
		return err
	}

	path := formatURL(oc.Path, args[:len(oc.URLParams)])

	oc.Parameters.AppendData(args[len(oc.URLParams):])

	_, err = oc.MakeRequest(apiKey, path, &oc.Parameters)

	return err
}

func (oc *OperationCmd) helpFunc(cmd *cobra.Command, args []string) {
	err := pageString(oc.helpString(cmd), cmd.OutOrStdout())
	if err != nil {
		panic(err)
	}
}

func (oc *OperationCmd) helpString(cmd *cobra.Command) string {
	var sb strings.Builder

	stripeSpec, err := spec.LoadSpec("")
	if err != nil {
		panic(err)
	}
	opSpec := stripeSpec.Paths[spec.Path(oc.Path)][spec.HTTPVerb(strings.ToLower(oc.HTTPVerb))]

	sb.WriteString(fmt.Sprintf(`%s

%s

%s

%s

%s
`,
		ansi.Bold("USAGE"),
		text.Indent(cmd.CommandPath(), "    "),
		ansi.Bold("DESCRIPTION"),
		text.Indent(text.Wrap(opSpec.Description, 76), "    "),
		ansi.Bold("PARAMETERS"),
	))

	paramsSchema := opSpec.RequestBody.Content["application/x-www-form-urlencoded"].Schema
	for _, name := range sortedParamNames(paramsSchema) {
		schema := paramsSchema.Properties[name]
		sb.WriteString(paramHelpString(name, schema, "    "))
		sb.WriteString("\n")
	}

	return sb.String()
}

//
// Public functions
//

// NewOperationCmd returns a new OperationCmd.
func NewOperationCmd(parentCmd *cobra.Command, name, path, httpVerb string, cfg *config.Config) *OperationCmd {
	urlParams := extractURLParams(path)
	operationCmd := &OperationCmd{
		Base: &requests.Base{
			Method:  httpVerb,
			Profile: &cfg.Profile,
		},
		Name:      name,
		HTTPVerb:  httpVerb,
		Path:      path,
		URLParams: urlParams,
	}
	cmd := &cobra.Command{
		Use:         name,
		Annotations: make(map[string]string),
		RunE:        operationCmd.runOperationCmd,
		Args:        cobra.MinimumNArgs(len(urlParams)),
	}
	cmd.SetUsageTemplate(operationUsageTemplate(urlParams))
	cmd.SetHelpFunc(operationCmd.helpFunc)
	cmd.DisableFlagsInUseLine = true
	operationCmd.Cmd = cmd
	operationCmd.InitFlags(false)

	parentCmd.AddCommand(cmd)
	parentCmd.Annotations[name] = "operation"

	return operationCmd
}

//
// Private functions
//

func extractURLParams(path string) []string {
	re := regexp.MustCompile(`{\w+}`)
	return re.FindAllString(path, -1)
}

func formatURL(path string, urlParams []string) string {
	s := make([]interface{}, len(urlParams))
	for i, v := range urlParams {
		s[i] = v
	}
	re := regexp.MustCompile(`{\w+}`)
	format := re.ReplaceAllString(path, "%s")
	return fmt.Sprintf(format, s...)
}

func operationUsageTemplate(urlParams []string) string {
	args := strings.Map(func(r rune) rune {
		switch r {
		case '{':
			return '<'
		case '}':
			return '>'
		}
		return r
	}, strings.Join(urlParams, " "))
	args += " [param1=value1] [param2=value2] ..."

	return fmt.Sprintf(`%s{{if .Runnable}}
  {{.UseLine}} %s{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s
{{WrappedLocalFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{WrappedInheritedFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

%s{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`,
		ansi.Bold("Usage:"),
		args,
		ansi.Bold("Aliases:"),
		ansi.Bold("Examples:"),
		ansi.Bold("Available Operations:"),
		ansi.Bold("Flags:"),
		ansi.Bold("Global Flags:"),
		ansi.Bold("Additional help topics:"),
	)
}

func paramHelpString(name string, schema *spec.Schema, indent string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n%so %s (%s)\n", indent, ansi.Bold(name), ansi.Italic(schema.Type)))

	indent += "  "

	if len(schema.Description) > 0 {
		converted := string(blackfriday.Markdown([]byte(schema.Description), ansi.MarkdownTermRenderer(0), 0))
		wrapped := text.Wrap(converted, 80-len(indent))
		sb.WriteString(text.Indent(wrapped, indent))
	}

	for _, subName := range sortedParamNames(schema) {
		subSchema := schema.Properties[subName]
		sb.WriteString(paramHelpString(subName, subSchema, indent))
	}

	return sb.String()
}

func sortedParamNames(schema *spec.Schema) []string {
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func pageString(s string, out io.Writer) error {
	pagerExe := ""
	switch {
	case len(os.Getenv("PAGER")) > 0:
		pagerExe = os.Getenv("PAGER")
	case runtime.GOOS == "windows":
		pagerExe = "more"
	default:
		pagerExe = "less"
	}

	pager := exec.Command(pagerExe)

	pager.Stdin = strings.NewReader(s)
	pager.Stdout = out

	err := pager.Run()
	if err != nil {
		return err
	}

	return nil
}
