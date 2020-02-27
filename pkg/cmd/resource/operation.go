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

	"github.com/kr/text"
	"github.com/russross/blackfriday"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/spec"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const (
	pathStripeSpec = "./api/openapi-spec/spec3.sdk.json"
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

	stringFlags map[string]*string

	data []string
}

func (oc *OperationCmd) runOperationCmd(cmd *cobra.Command, args []string) error {
	apiKey, err := oc.Profile.GetAPIKey(oc.Livemode)
	if err != nil {
		return err
	}

	path := formatURL(oc.Path, args)

	flagParams := make([]string, 0)

	for stringProp, stringVal := range oc.stringFlags {
		// only include fields explicitly set by the user to avoid conflicts between e.g. account_balance, balance
		if oc.Cmd.Flags().Changed(stringProp) {
			paramName := strings.ReplaceAll(stringProp, "-", "_")
			flagParams = append(flagParams, fmt.Sprintf("%s=%s", paramName, *stringVal))
		}
	}

	for _, datum := range oc.data {
		split := strings.SplitN(datum, "=", 2)
		if len(split) < 2 {
			return fmt.Errorf("Invalid data argument: %s", datum)
		}

		if _, ok := oc.stringFlags[split[0]]; ok {
			return fmt.Errorf("Flag \"%s\" already set", split[0])
		}

		flagParams = append(flagParams, datum)
	}

	oc.Parameters.AppendData(flagParams)

	_, err = oc.MakeRequest(apiKey, path, &oc.Parameters, false)

	return err
}

//
// Public functions
//

// NewOperationCmd returns a new OperationCmd.
func NewOperationCmd(parentCmd *cobra.Command, name, path, httpVerb string, propFlags map[string]string, cfg *config.Config) *OperationCmd {
	urlParams := extractURLParams(path)
	httpVerb = strings.ToUpper(httpVerb)
	operationCmd := &OperationCmd{
		Base: &requests.Base{
			Method:  httpVerb,
			Profile: &cfg.Profile,
		},
		Name:      name,
		HTTPVerb:  httpVerb,
		Path:      path,
		URLParams: urlParams,

		stringFlags: make(map[string]*string),
	}
	cmd := &cobra.Command{
		Use:         name,
		Annotations: make(map[string]string),
		RunE:        operationCmd.runOperationCmd,
		Args:        validators.ExactArgs(len(urlParams)),
	}

	for prop := range propFlags {
		// it's ok to treat all flags as string flags because we don't send any default flag values to the API
		// i.e. "account_balance" default is "" not 0 but this is ok
		flagName := strings.ReplaceAll(prop, "_", "-")
		operationCmd.stringFlags[flagName] = cmd.Flags().String(flagName, "", "")
		cmd.Flags().SetAnnotation(flagName, "request", []string{"true"})
	}

	cmd.SetHelpFunc(operationCmd.helpFunc)
	cmd.SetUsageTemplate(operationUsageTemplate(urlParams))
	cmd.DisableFlagsInUseLine = true
	operationCmd.Cmd = cmd
	operationCmd.InitFlags()

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
	if args != "" {
		args += " "
	}

	args += "[--param=value] [-d \"nested[param]=value\"]"

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
{{WrappedRequestParamsFlagUsages . | trimTrailingWhitespaces}}

%s
{{WrappedNonRequestParamsFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

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
		ansi.Bold("Request Parameters:"),
		ansi.Bold("Flags:"),
		ansi.Bold("Global Flags:"),
		ansi.Bold("Additional help topics:"),
	)
}

func (oc *OperationCmd) helpFunc(cmd *cobra.Command, args []string) {
	err := pageString(oc.helpString(cmd), cmd.OutOrStdout())
	if err != nil {
		panic(err)
	}
}

func (oc *OperationCmd) helpString(cmd *cobra.Command) string {
	var sb strings.Builder

	stripeSpec, err := spec.LoadSpec(pathStripeSpec)
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
