package resource

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
)

// EventsResendCmd represents the event resend API operation command. This
// command is manually defined because it has a custom behavior.
type EventsResendCmd struct {
	opCmd *OperationCmd
}

func (erc *EventsResendCmd) runEventsResendCmd(cmd *cobra.Command, args []string) error {
	// If the `webhook-endpoint` flag was not passed, then add
	// `for_stripecli=true` to the request so the event is replayed to the
	// Stripe CLI.
	if !erc.opCmd.Cmd.Flags().Changed("webhook-endpoint") {
		erc.opCmd.Parameters.AppendData([]string{"for_stripecli=true"})
	}

	return erc.opCmd.runOperationCmd(cmd, args)
}

// NewEventsResendCmd returns a new EventsResendCmd.
func NewEventsResendCmd(parentCmd *cobra.Command, cfg *config.Config) *EventsResendCmd {
	eventsResendCmd := &EventsResendCmd{
		opCmd: NewOperationCmd(parentCmd, "resend", "/v1/events/{event}/retry", http.MethodPost, map[string]string{
			"account":          "string",
			"webhook_endpoint": "string",
		}, cfg),
	}

	eventsResendCmd.opCmd.Cmd.RunE = eventsResendCmd.runEventsResendCmd

	return eventsResendCmd
}
