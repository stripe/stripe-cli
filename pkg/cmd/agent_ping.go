package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/validators"
)

const agentPingEventName = "agent.ping"

type agentPingCmd struct {
	cmd *cobra.Command

	skill string
}

type agentPingEvent struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func newAgentPingCmd() *agentPingCmd {
	apc := &agentPingCmd{}
	apc.cmd = &cobra.Command{
		Use:           "ping",
		Short:         "Record Stripe skill usage",
		Long:          "Record Stripe skill usage telemetry from an automated agent hook.",
		Args:          validators.NoArgs,
		Run:           apc.runPing,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	apc.cmd.Flags().StringVar(&apc.skill, "skill", "", "Name of the Stripe skill that was used")

	return apc
}

func (apc *agentPingCmd) runPing(cmd *cobra.Command, _ []string) {
	if apc.skill == "" {
		return
	}

	payload, err := json.Marshal(agentPingEvent{
		Type: "skill",
		Name: apc.skill,
	})
	if err != nil {
		return
	}

	sendAgentEvent(commandContextOrBackground(cmd), agentPingEventName, string(payload))
}
