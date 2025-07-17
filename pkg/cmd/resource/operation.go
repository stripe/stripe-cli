package resource

import (
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/spec"
	"github.com/stripe/stripe-cli/pkg/stripe"
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

	IsPreviewCommand bool

	stringFlags  map[string]*string
	arrayFlags   map[string]*[]string
	integerFlags map[string]*int
	boolFlags    map[string]*bool
}

func (oc *OperationCmd) runOperationCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateAPIBaseURL(oc.APIBaseURL); err != nil {
		return err
	}

	apiKey, err := oc.Profile.GetAPIKey(oc.Livemode)
	if err != nil {
		return err
	}

	path := formatURL(oc.Path, args)
	requestParams := make(map[string]interface{})
	oc.addStringRequestParams(requestParams)
	oc.addIntRequestParams(requestParams)
	oc.addBoolRequestParams(requestParams)

	err = oc.addArrayRequestParams(requestParams)
	if err != nil {
		return err
	}

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
		if strings.HasPrefix(path, "/v1/accounts/") {
			connectedAccountID := strings.Split(path, "/")[3]
			fmt.Printf("> Connected Account: %s\n", connectedAccountID)
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
		_, err = oc.MakeRequest(cmd.Context(), apiKey, path, &oc.Parameters, requestParams, false, nil)

		return err
	}
	// else
	_, err = oc.MakeRequest(cmd.Context(), apiKey, path, &oc.Parameters, requestParams, false, nil)
	return err
}

//
// Public functions
//

// NewUnsupportedV2BillingOperationCmd returns a new cobra command for an unsupported v2 billing command.
// This is temporary until resource commands support the /v2/billing namespace.
func NewUnsupportedV2BillingOperationCmd(parentCmd *cobra.Command, name string, path string) *cobra.Command {
	cmd := &cobra.Command{
		Use:         name,
		Annotations: make(map[string]string),
		Run: func(cmd *cobra.Command, args []string) {
			output := `
%s is not supported by Stripe CLI yet. Please use the %s or cURL to create a %s, instead.

* Hint: If you're trying to test webhook events, you can always use %s or %s.
			`

			fmt.Println(fmt.Sprintf(output, ansi.Bold(path), ansi.Linkify("Dashboard", "https://dashboard.stripe.com", os.Stdout), parentCmd.Name(), ansi.Bold("stripe trigger v1.billing.meter.no_meter_found"), ansi.Bold("stripe trigger v1.billing.meter.error_report_triggered")))
		},
	}
	parentCmd.AddCommand(cmd)
	return cmd
}

// NewOperationCmd returns a new OperationCmd.
func NewOperationCmd(parentCmd *cobra.Command, name, path, httpVerb string,
	propFlags map[string]string, enumFlags map[string][]spec.StripeEnumValue, cfg *config.Config, isPreview bool) *OperationCmd {
	urlParams := extractURLParams(path)
	httpVerb = strings.ToUpper(httpVerb)
	operationCmd := &OperationCmd{
		Base: &requests.Base{
			Method:           httpVerb,
			Profile:          &cfg.Profile,
			IsPreviewCommand: isPreview,
		},
		Name:             name,
		HTTPVerb:         httpVerb,
		Path:             path,
		URLParams:        urlParams,
		IsPreviewCommand: isPreview,

		arrayFlags:   make(map[string]*[]string),
		stringFlags:  make(map[string]*string),
		integerFlags: make(map[string]*int),
		boolFlags:    make(map[string]*bool),
	}
	cmd := &cobra.Command{
		Use:         name,
		Annotations: make(map[string]string),
		RunE:        operationCmd.runOperationCmd,
		Args:        validators.ExactArgs(len(urlParams)),
	}

	for prop, propType := range propFlags {
		// it's ok to treat all flags as string flags because we don't send any default flag values to the API
		// i.e. "account_balance" default is "" not 0 but this is ok
		flagName := strings.ReplaceAll(prop, "_", "-")

		// Create flag description
		var description string
		if enums, hasEnum := enumFlags[prop]; hasEnum {
			// Create a description that includes enum values
			enumValues := []string{}
			for _, enum := range enums {
				enumValues = append(enumValues, fmt.Sprintf("%s (%s)", enum.Value, enum.Description))
			}
			description = fmt.Sprintf("Possible values: %s", strings.Join(enumValues, ", "))
		} else {
			description = "" // Default empty description
		}

		switch propType {
		case "array":
			operationCmd.arrayFlags[flagName] = cmd.Flags().StringArray(flagName, []string{}, description)
		case "string":
			operationCmd.stringFlags[flagName] = cmd.Flags().String(flagName, "", description)
		case "number":
			operationCmd.stringFlags[flagName] = cmd.Flags().String(flagName, "", description)
		case "integer":
			operationCmd.integerFlags[flagName] = cmd.Flags().Int(flagName, -1, description)
		case "boolean":
			operationCmd.boolFlags[flagName] = cmd.Flags().Bool(flagName, false, description)
		default:
		}
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
		ansi.Italic("Note: all types are specifically for the Stripe CLI itself, not the Stripe API. The CLI handles\ntransforming types to what the API expects."),
		ansi.Bold("Flags:"),
		ansi.Bold("Global Flags:"),
		ansi.Bold("Additional help topics:"),
	)
}

func constructParamFromDot(dotParam string) string {
	paramPath := strings.Split(dotParam, ".")
	var param string
	for i, p := range paramPath {
		if i == 0 {
			param = p
		} else {
			param = fmt.Sprintf("%s[%s]", param, p)
		}
	}

	return param
}

func (oc *OperationCmd) addStringRequestParams(requestParams map[string]interface{}) {
	for stringProp, stringVal := range oc.stringFlags {
		// only include fields explicitly set by the user to avoid conflicts between e.g. account_balance, balance
		if oc.Cmd.Flags().Changed(stringProp) {
			paramName := getParamName(stringProp)
			if strings.Contains(paramName, ".") {
				constructedNestedStringParams(requestParams, strings.Split(paramName, "."), stringVal)
			} else {
				requestParams[paramName] = *stringVal
			}
		}
	}
}

func (oc *OperationCmd) addIntRequestParams(requestParams map[string]interface{}) {
	for intProp, intVal := range oc.integerFlags {
		if oc.Cmd.Flags().Changed(intProp) {
			paramName := getParamName(intProp)
			if strings.Contains(paramName, ".") {
				constructedNestedIntParams(requestParams, strings.Split(paramName, "."), intVal)
			} else {
				requestParams[paramName] = *intVal
			}
		}
	}
}

func (oc *OperationCmd) addBoolRequestParams(requestParams map[string]interface{}) {
	for boolProp, boolVal := range oc.boolFlags {
		if oc.Cmd.Flags().Changed(boolProp) {
			paramName := getParamName(boolProp)
			if strings.Contains(paramName, ".") {
				constructedNestedBoolParams(requestParams, strings.Split(paramName, "."), boolVal)
			} else {
				requestParams[paramName] = *boolVal
			}
		}
	}
}

func (oc *OperationCmd) addArrayRequestParams(requestParams map[string]interface{}) error {
	for arrayProp, arrayVal := range oc.arrayFlags {
		// only include fields explicitly set by the user to avoid conflicts between e.g. account_balance, balance
		if oc.Cmd.Flags().Changed(arrayProp) {
			paramName := getParamName(arrayProp)
			for _, arrayItem := range *arrayVal {
				if strings.Contains(paramName, ".") {
					constructedNestedArrayParams(requestParams, strings.Split(paramName, "."), arrayItem)
				} else {
					if _, ok := requestParams[paramName]; !ok {
						requestParams[paramName] = make([]interface{}, 0)
					}
					switch v := reflect.ValueOf(requestParams[paramName]); v.Kind() {
					case reflect.Array, reflect.Slice:
						requestParams[paramName] = append(requestParams[paramName].([]interface{}), arrayItem)
					default:
						return fmt.Errorf("array parameter flag %s has conflict with another non-array parameter flag", paramName)
					}
				}
			}
		}
	}
	return nil
}

func constructedNestedStringParams(params map[string]interface{}, paramKeys []string, stringVal *string) {
	if len(paramKeys) == 0 {
		return
	}

	field := paramKeys[0]

	if len(paramKeys) == 1 {
		params[field] = *stringVal
		return
	}

	if _, ok := params[field]; !ok {
		params[field] = make(map[string]interface{}, 0)
	}

	constructedNestedStringParams(params[field].(map[string]interface{}), paramKeys[1:], stringVal)
}

func constructedNestedIntParams(params map[string]interface{}, paramKeys []string, intVal *int) {
	if len(paramKeys) == 0 {
		return
	}

	field := paramKeys[0]

	if len(paramKeys) == 1 {
		params[field] = *intVal
		return
	}

	if _, ok := params[field]; !ok {
		params[field] = make(map[string]interface{}, 0)
	}

	constructedNestedIntParams(params[field].(map[string]interface{}), paramKeys[1:], intVal)
}

func constructedNestedBoolParams(params map[string]interface{}, paramKeys []string, boolVal *bool) {
	if len(paramKeys) == 0 {
		return
	}

	field := paramKeys[0]

	if len(paramKeys) == 1 {
		params[field] = *boolVal
		return
	}

	if _, ok := params[field]; !ok {
		params[field] = make(map[string]interface{}, 0)
	}

	constructedNestedBoolParams(params[field].(map[string]interface{}), paramKeys[1:], boolVal)
}

func constructedNestedArrayParams(params map[string]interface{}, paramKeys []string, arrayVal string) {
	if len(paramKeys) == 0 {
		return
	}

	field := paramKeys[0]

	if len(paramKeys) == 1 {
		if _, ok := params[field]; !ok {
			params[field] = make([]interface{}, 0)
		}
		params[field] = append(params[field].([]interface{}), arrayVal)
		return
	}

	if _, ok := params[field]; !ok {
		params[field] = make(map[string]interface{}, 0)
	}

	constructedNestedArrayParams(params[field].(map[string]interface{}), paramKeys[1:], arrayVal)
}

func getParamName(prop string) string {
	return strings.ReplaceAll(prop, "-", "_")
}
