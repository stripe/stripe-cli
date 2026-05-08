package resource

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
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

	apiKey, apiKeyErr := oc.Profile.GetAPIKey(oc.Livemode)

	path := formatURL(oc.Path, args)
	requestParams := make(map[string]interface{})
	oc.addStringRequestParams(requestParams)
	oc.addIntRequestParams(requestParams)
	oc.addBoolRequestParams(requestParams)

	if err := oc.addArrayRequestParams(requestParams); err != nil {
		return err
	}

	if oc.DryRun {
		dryRunKey := apiKey
		if apiKeyErr != nil {
			dryRunKey = ""
		}
		output, err := oc.BuildDryRunOutput(dryRunKey, oc.APIBaseURL, path, &oc.Parameters, requestParams)
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil
	}

	if apiKeyErr != nil {
		return apiKeyErr
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
	_, err := oc.MakeRequest(cmd.Context(), apiKey, path, &oc.Parameters, requestParams, false, nil)
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
func NewOperationCmd(parentCmd *cobra.Command, opSpec *OperationSpec, cfg *config.Config) *OperationCmd {
	method := strings.ToUpper(opSpec.Method)
	urlParams := extractURLParams(opSpec.Path)

	operationCmd := &OperationCmd{
		Base: &requests.Base{
			Method:           method,
			Profile:          &cfg.Profile,
			IsPreviewCommand: opSpec.IsPreview,
		},
		Name:             opSpec.Name,
		HTTPVerb:         method,
		Path:             opSpec.Path,
		URLParams:        urlParams,
		IsPreviewCommand: opSpec.IsPreview,

		arrayFlags:   make(map[string]*[]string),
		stringFlags:  make(map[string]*string),
		integerFlags: make(map[string]*int),
		boolFlags:    make(map[string]*bool),
	}
	cmd := &cobra.Command{
		Use:         opSpec.Name,
		Annotations: make(map[string]string),
		RunE:        operationCmd.runOperationCmd,
		Args:        validators.ExactArgs(len(urlParams)),
	}

	for prop, paramSpec := range opSpec.Params {
		// it's ok to treat all flags as string flags because we don't send any default flag values to the API
		// i.e. "account_balance" default is "" not 0 but this is ok
		flagName := strings.ReplaceAll(prop, "_", "-")
		desc := paramSpec.ShortDescription

		switch paramSpec.Type {
		case "array":
			operationCmd.arrayFlags[flagName] = cmd.Flags().StringArray(flagName, []string{}, desc)
		case "string":
			operationCmd.stringFlags[flagName] = cmd.Flags().String(flagName, "", desc)
		case "clearable_object":
			operationCmd.stringFlags[flagName] = cmd.Flags().String(flagName, "", desc)
		case "number":
			operationCmd.stringFlags[flagName] = cmd.Flags().String(flagName, "", desc)
		case "integer":
			operationCmd.integerFlags[flagName] = cmd.Flags().Int(flagName, -1, desc)
		case "boolean":
			operationCmd.boolFlags[flagName] = cmd.Flags().Bool(flagName, false, desc)
		default:
		}
		cmd.Flags().SetAnnotation(flagName, "request", []string{"true"})
		cmd.Flags().SetAnnotation(flagName, "apitype", []string{paramSpec.Type})
		if paramSpec.Required {
			cmd.Flags().SetAnnotation(flagName, "required", []string{"true"})
		}
		if paramSpec.MostCommon {
			cmd.Flags().SetAnnotation(flagName, "mostcommon", []string{"true"})
		}
		if paramSpec.Format != "" {
			cmd.Flags().SetAnnotation(flagName, "format", []string{paramSpec.Format})
		}
		if len(paramSpec.Enum) > 0 {
			enumVals := make([]string, 0, len(paramSpec.Enum))
			for _, ev := range paramSpec.Enum {
				enumVals = append(enumVals, ev.Value)
			}
			cmd.Flags().SetAnnotation(flagName, "enum", enumVals)
		}
	}

	cmd.SetUsageTemplate(operationUsageTemplate(urlParams))
	cmd.DisableFlagsInUseLine = true
	operationCmd.Cmd = cmd
	operationCmd.InitFlags()

	// Set the operation-specific server URL after InitFlags if provided
	// We need to set both the value and the default value of the flag
	if opSpec.ServerURL != "" {
		operationCmd.APIBaseURL = opSpec.ServerURL
		// Also update the flag's default value so it doesn't get reset during parsing
		if flag := cmd.Flags().Lookup("api-base"); flag != nil {
			flag.DefValue = opSpec.ServerURL
			flag.Value.Set(opSpec.ServerURL)
		}
	}

	parentCmd.AddCommand(cmd)
	parentCmd.Annotations[opSpec.Name] = "operation"

	defaultHelp := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if c.Example == "" {
			c.Example = buildExamples(c.CommandPath(), opSpec)
		}
		defaultHelp(c, args)
	})

	return operationCmd
}

// paramFlagName converts a param key (underscore-separated) to its flag name (hyphen-separated).
// e.g. "account_balance" → "account-balance", "usage_threshold.gte" → "usage-threshold.gte"
func paramFlagName(param string) string {
	return strings.ReplaceAll(param, "_", "-")
}

// exampleValue returns the placeholder value string to use in an example for the given param.
func exampleValue(ps *ParamSpec) string {
	switch ps.Type {
	case "integer":
		return "<integer>"
	case "boolean":
		return "<boolean>"
	default:
		if len(ps.Enum) > 0 {
			return "<enum>"
		}
		return "<string>"
	}
}

// buildExamples generates an example invocation for a command's --help output.
// The goal is quick orientation: show the minimum needed to call the API.
//
// If there are required params: show a single "# required fields" line with those params.
// If there are no required params but MostCommon params exist: show up to the first two
// (alphabetically), with a trailing " ..." if more exist.
// If there are no params at all, or none are required or MostCommon: return "".
func buildExamples(cmdPath string, opSpec *OperationSpec) string {
	var reqFields []string
	for name, p := range opSpec.Params {
		if p.Required {
			reqFields = append(reqFields, name)
		}
	}
	sort.Strings(reqFields)

	if len(reqFields) > 0 {
		return "  # required fields\n" + buildExampleLine(cmdPath, reqFields, opSpec.Params, false)
	}

	// No required fields: use top-level (depth-0) non-clearable MostCommon params if any
	// are curated; otherwise no example. Clearable_object params (e.g. --address="") and
	// depth-1 sub-fields are excluded — examples should show scalar params only.
	var candidates []string
	for name, p := range opSpec.Params {
		if p.MostCommon && !strings.Contains(name, ".") && p.Type != "clearable_object" {
			candidates = append(candidates, name)
		}
	}
	sort.Strings(candidates)

	if len(candidates) == 0 {
		return ""
	}
	ellipsis := len(candidates) > 2
	if ellipsis {
		candidates = candidates[:2]
	}
	return buildExampleLine(cmdPath, candidates, opSpec.Params, ellipsis)
}

// buildExampleLine constructs a single example command line for the given fields.
// If ellipsis is true, " ..." is appended to indicate additional params exist.
func buildExampleLine(cmdPath string, fields []string, params map[string]*ParamSpec, ellipsis bool) string {
	var tokens []string
	for _, field := range fields {
		ps, ok := params[field]
		if !ok {
			continue
		}
		tokens = append(tokens, fmt.Sprintf("--%s %s", paramFlagName(field), exampleValue(ps)))
	}
	if len(tokens) == 0 {
		return ""
	}
	line := fmt.Sprintf("  $ %s %s", cmdPath, strings.Join(tokens, " "))
	if ellipsis {
		line += " ..."
	}
	return line
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
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{AIAgentHelp .}}{{if .HasAvailableLocalFlags}}

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
			val := *stringVal
			// For clearable_object flags, "{}" is accepted as an alias for "" for
			// compatibility with other tools. The v1 API requires an empty string to
			// clear the field, so translate "{}" accordingly.
			if val == "{}" {
				if f := oc.Cmd.Flags().Lookup(stringProp); f != nil {
					if apitype, ok := f.Annotations["apitype"]; ok && len(apitype) > 0 && apitype[0] == "clearable_object" {
						val = ""
					}
				}
			}
			if strings.Contains(paramName, ".") {
				constructedNestedStringParams(requestParams, strings.Split(paramName, "."), &val)
			} else {
				requestParams[paramName] = val
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
