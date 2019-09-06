package resource

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
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

//
// Public functions
//

// NewOperationCmd returns a new OperationCmd.
func NewOperationCmd(parentCmd *cobra.Command, name, path, httpVerb string, cfg *config.Config) *OperationCmd {
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
	}
	cmd := &cobra.Command{
		Use:         name,
		Annotations: make(map[string]string),
		RunE:        operationCmd.runOperationCmd,
		Args:        cobra.MinimumNArgs(len(urlParams)),
	}
	cmd.SetUsageTemplate(operationUsageTemplate(urlParams))
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
