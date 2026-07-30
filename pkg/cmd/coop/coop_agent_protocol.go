package coopcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/workflow"
)

func (c *coopAgentActionCmd) addSessionStepFlags() {
	c.cmd.Flags().StringVar(&c.session, "session", "", "Session ID")
	c.cmd.Flags().IntVar(&c.step, "step", 0, "1-based node number")
}

func (c *coopAgentActionCmd) validateSessionStep(action string) error {
	if strings.TrimSpace(c.session) != "" && c.step > 0 {
		return nil
	}
	return outputCoopError(
		"--session and a positive --step are required",
		"Provide the Co-op session ID and 1-based node number.",
		coop.SessionStepTemplate(action),
	)
}

func configureAgentCommand(cmd *cobra.Command) {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return outputCoopError(
			err.Error(),
			"Correct the command flags and retry.",
			coop.Continue(coop.StatusCommand("")),
		)
	})
}

func agentNoArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return outputCoopError(
		fmt.Sprintf("%s does not accept positional arguments", cmd.CommandPath()),
		"Remove the unexpected positional arguments and retry.",
		coop.Continue(coop.StatusCommand("")),
	)
}

func newWorkflowService() (*workflow.Service, error) {
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return nil, fmt.Errorf("creating store: %w", err)
	}
	return workflow.NewService(store), nil
}

func parseReportedOutputs(values []string) (coop.NodeOutputs, error) {
	if len(values) == 0 {
		return nil, nil
	}
	outputs := coop.NodeOutputs{}
	for _, value := range values {
		selector, rawValue, ok := strings.Cut(value, "=")
		if !ok {
			return nil, fmt.Errorf("--output must be in field=value or source:field=value format: %q", value)
		}
		selector = strings.TrimSpace(selector)
		if selector == "" {
			return nil, fmt.Errorf("--output field cannot be empty: %q", value)
		}
		if rawValue == "" {
			return nil, fmt.Errorf("--output value cannot be empty: %q", value)
		}

		source := coop.DefaultOutputSource
		field := selector
		if parsedSource, parsedField, hasSource := strings.Cut(selector, ":"); hasSource {
			source = strings.TrimSpace(parsedSource)
			field = strings.TrimSpace(parsedField)
			if source == "" || field == "" {
				return nil, fmt.Errorf("--output source and field cannot be empty: %q", value)
			}
		}

		raw := json.RawMessage(rawValue)
		if !json.Valid(raw) {
			encoded, err := json.Marshal(rawValue)
			if err != nil {
				return nil, fmt.Errorf("encoding --output %q: %w", selector, err)
			}
			raw = encoded
		}
		if outputs[source] == nil {
			outputs[source] = map[string]json.RawMessage{}
		}
		outputs[source][field] = append(json.RawMessage(nil), raw...)
	}
	return outputs, nil
}

// outputAgentError renders err as a structured agent JSON response. Used for
// failures that happen before a workflow CommandResponse exists (e.g.
// newWorkflowService / store creation), so agent commands never emit a bare
// plain-text error on that path.
func outputAgentError(err error) error {
	return outputAgentResponse(coop.CommandResponse{}, err)
}

func outputCoopError(message, hint string, continuation coop.Continuation) error {
	return outputAgentResponse(coop.CommandResponse{
		OK:       false,
		Error:    message,
		Recovery: continuation.Recovery(hint),
	}, nil)
}

func outputAgentResponse(resp coop.CommandResponse, err error) error {
	if err != nil {
		resp = protocolFailure(err.Error())
	}
	if validationErr := resp.Validate(); validationErr != nil {
		resp = protocolFailure("invalid Co-op protocol response: " + validationErr.Error())
	}
	if !resp.OK {
		if resp.Recovery == nil {
			resp.Recovery = defaultAgentRecovery()
		}
		if outErr := outputJSONTo(os.Stderr, resp); outErr != nil {
			return outErr
		}
		return RenderedError{}
	}
	return outputJSON(resp)
}

func protocolFailure(message string) coop.CommandResponse {
	return coop.CommandResponse{
		OK:       false,
		Error:    message,
		Recovery: defaultAgentRecovery(),
	}
}

func defaultAgentRecovery() *coop.Recovery {
	return coop.Continue(coop.StatusCommand("")).
		Recovery("Inspect the current Co-op session before retrying.")
}
