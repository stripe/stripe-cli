package resource

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/cmdutil"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

var (
	climateEnableSpec = OperationSpec{
		Name:   "enable",
		Path:   "/v1/climate/program",
		Method: http.MethodPost,
		// No Params: rate comes from positional arg, synthesized into the request body in RunE.
	}
	climateShowSpec = OperationSpec{
		Name:   "show",
		Path:   "/v1/climate/program",
		Method: http.MethodGet,
	}
	climateDisableSpec = OperationSpec{
		Name:   "disable",
		Path:   "/v1/climate/program",
		Method: http.MethodDelete,
	}
)

// AddClimateCommitmentsSubCmds patches the commitments command tree into the
// auto-generated `climate` namespace.
func AddClimateCommitmentsSubCmds(rootCmd *cobra.Command, cfg *config.Config) {
	climateCmd, ok := cmdutil.FindSubCmd(rootCmd, "climate")
	if !ok {
		return
	}

	rCommitmentCmd := NewResourceCmd(climateCmd, "commitment")

	newClimateEnableCmd(rCommitmentCmd.Cmd, cfg)
	newClimateShowCmd(rCommitmentCmd.Cmd, cfg)
	newClimateDisableCmd(rCommitmentCmd.Cmd, cfg)
}

func newClimateEnableCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := NewOperationCmd(parentCmd, &climateEnableSpec, cfg)
	opCmd.Cmd.Short = "Enable or update a rate-based Climate commitment"
	opCmd.Cmd.Use = "enable <rate>"
	opCmd.Cmd.Long = `Enable or update your Climate commitment program at the given rate.

RATE is a percentage of pay-in volume (e.g. "1.5" for 1.5%).
If a program is already active, this replaces the existing rate.`
	opCmd.Cmd.Args = cobra.ExactArgs(1)
	opCmd.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runClimateOperation(cmd, opCmd, map[string]interface{}{
			"contribution_method":       "rate",
			"contribution_rate_percent": args[0],
		})
	}
	return opCmd.Cmd
}

func newClimateShowCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := NewOperationCmd(parentCmd, &climateShowSpec, cfg)
	opCmd.Cmd.Short = "Show the current Climate commitment program"
	opCmd.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runClimateOperation(cmd, opCmd, nil)
	}
	return opCmd.Cmd
}

func newClimateDisableCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := NewOperationCmd(parentCmd, &climateDisableSpec, cfg)
	opCmd.Cmd.Short = "Disable the current Climate commitment program"
	opCmd.SuppressOutput = true
	return opCmd.Cmd
}

func runClimateOperation(cmd *cobra.Command, opCmd *OperationCmd, requestParams map[string]interface{}) error {
	if err := stripe.ValidateAPIBaseURL(opCmd.APIBaseURL); err != nil {
		return err
	}

	// Suppress base output so we can transform the response before printing.
	opCmd.SuppressOutput = true

	apiKey, apiKeyErr := opCmd.Profile.GetAPIKey(opCmd.Livemode)
	if opCmd.DryRun {
		dryRunKey := apiKey
		if apiKeyErr != nil {
			dryRunKey = ""
		}
		output, err := opCmd.BuildDryRunOutput(dryRunKey, opCmd.APIBaseURL, opCmd.Path, &opCmd.Parameters, requestParams)
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

	body, err := opCmd.MakeRequest(cmd.Context(), apiKey, opCmd.Path, &opCmd.Parameters, requestParams, false, nil)
	if err != nil || len(body) == 0 {
		return err
	}

	transformed, err := transformClimateProgram(body)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), ansi.ColorizeJSON(string(transformed), opCmd.DarkStyle, cmd.OutOrStdout()))
	return nil
}

// transformClimateProgram maps verbose API field names to shorter CLI-friendly
// names, keeping only the fields relevant to the program type.
func transformClimateProgram(body []byte) ([]byte, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return body, nil
	}

	var method string
	if raw, ok := obj["contribution_method"]; ok {
		method, _ = raw.(string)
		obj["type"] = method
		delete(obj, "contribution_method")
	}

	switch method {
	case "rate":
		if v, ok := obj["contribution_rate_percent"]; ok {
			obj["rate"] = v
			delete(obj, "contribution_rate_percent")
		}
		delete(obj, "contribution_flat_amount")
		delete(obj, "contribution_flat_currency")
	case "flat":
		if v, ok := obj["contribution_flat_amount"]; ok {
			obj["amount"] = v
			delete(obj, "contribution_flat_amount")
		}
		if v, ok := obj["contribution_flat_currency"]; ok {
			obj["currency"] = v
			delete(obj, "contribution_flat_currency")
		}
		delete(obj, "contribution_rate_percent")
	default:
		delete(obj, "contribution_rate_percent")
		delete(obj, "contribution_flat_amount")
		delete(obj, "contribution_flat_currency")
	}

	delete(obj, "initial_start_date")
	return json.Marshal(obj)
}
