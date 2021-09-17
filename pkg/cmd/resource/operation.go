package resource

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
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

	if oc.HTTPVerb == http.MethodDelete {
		// display account information and confirm whether user wants to proceed
		var mode = "Test"
		displayName := oc.Profile.GetDisplayName()

		if oc.Livemode {
			mode = "Live"
		}

		// display account information and confirmation to proceed
		fmt.Printf("This command will be executed on the account with the following details:\n")
		fmt.Printf("> Mode: %s\n", mode)
		if displayName != "" {
			fmt.Printf("> Account Name: %s\n", displayName)
		}

		// call the confirm command from base request
		confirmation, err := oc.Confirm()
		if err != nil {
			return err
		} else if !confirmation {
			fmt.Println("Exiting without execution. User did not confirm the command.")
			return nil
		}

		// if confirmation is provided, make the request
		_, err = oc.MakeRequest(cmd.Context(), apiKey, path, &oc.Parameters, false)

		return err
	}
	// else
	_, err = oc.MakeRequest(cmd.Context(), apiKey, path, &oc.Parameters, false)
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
