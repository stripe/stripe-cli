package resource

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/spec"
)

//
// Public types
//

// OperationCmd represents operation commands. Operation commands are nested
// under resource commands and represent a specific API operation for that
// resource.
type OperationCmd struct {
	*requests.Base

	Name      string
	HTTPVerb  string
	Path      string
	URLParams []string
}

func (oc *OperationCmd) runOperationCmd(cmd *cobra.Command, args []string) error {
	secretKey, err := oc.Profile.GetSecretKey()
	if err != nil {
		return err
	}

	path := formatURL(oc.Path, args[:len(oc.URLParams)])

	oc.Parameters.AppendData(args[len(oc.URLParams):])

	_, err = oc.MakeRequest(secretKey, path, &oc.Parameters)

	return err
}

//
// Public functions
//

// NewOperationCmd returns a new OperationCmd.
func NewOperationCmd(parentCmd *cobra.Command, op spec.StripeOperation, op2 spec.Operation) *OperationCmd {
	urlParams := extractURLParams(op.Path)
	httpVerb := strings.ToUpper(string(op.Operation))
	operationCmd := &OperationCmd{
		Base: &requests.Base{
			Method: httpVerb,
		},
		Name:      op.MethodName,
		HTTPVerb:  httpVerb,
		Path:      op.Path,
		URLParams: urlParams,
	}
	cmd := &cobra.Command{
		Use:         op.MethodName,
		Long:        op2.Description,
		Annotations: make(map[string]string),
		RunE:        operationCmd.runOperationCmd,
		Args:        cobra.MinimumNArgs(len(urlParams)),
	}
	cmd.SetUsageTemplate(operationUsageTemplate(op2, urlParams))
	cmd.DisableFlagsInUseLine = true
	operationCmd.Cmd = cmd
	operationCmd.InitFlags()

	parentCmd.AddCommand(cmd)
	parentCmd.Annotations[op.MethodName] = "operation"

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

func operationUsageTemplate(op spec.Operation, urlParams []string) string {
	params := strings.Map(func(r rune) rune {
		switch r {
		case '{':
			return '<'
		case '}':
			return '>'
		}
		return r
	}, strings.Join(urlParams, " "))
	params += " [param1=value1] [param2=value2] ..."

	return fmt.Sprintf(`%s{{if .Runnable}}
  {{.UseLine}} %s{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
{{.Example}}{{end}}%s{{if .HasAvailableSubCommands}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

%s{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`,
		ansi.Bold("Usage:"),
		params,
		ansi.Bold("Aliases:"),
		ansi.Bold("Examples:"),
		parametersHelp(op),
		ansi.Bold("Available Operations:"),
		ansi.Bold("Flags:"),
		ansi.Bold("Global Flags:"),
		ansi.Bold("Additional help topics:"),
	)
}

func parametersHelp(op spec.Operation) string {
	t := template.Must(template.New("parameters").Parse(fmt.Sprintf(`

%s{{with (index .RequestBody.Content "application/x-www-form-urlencoded").Schema }}{{ range $key, $value := .Properties }}
  {{ $key }} ({{ $value.Type }}){{end}}{{end}}`,
		ansi.Bold("Parameters:"),
	)))

	var out bytes.Buffer
	if err := t.Execute(&out, op); err != nil {
		// TODO better error handling
		log.Fatal(err)
		return ""
	}

	return out.String()
}
