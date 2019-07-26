package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/logs"
)

const requestLogsWebSocketFeature = "request_logs"

// LogsTailCmd wraps the configuration for the tail command
type LogsTailCmd struct {
	Cmd *cobra.Command

	apiBaseURL   string
	noWSS        bool
	webSocketURL string
}

// NewLogsTailCmd creates and initializes the tail command for the logs package
func NewLogsTailCmd() *LogsTailCmd {
	tailCmd := &LogsTailCmd{}

	tailCmd.Cmd = &cobra.Command{
		Use:   "tail",
		Args:  validators.NoArgs,
		Short: "Listens for API request logs sent from Stripe to help test your integration.",
		Long: fmt.Sprintf(`
The tail command lets you tail API request logs from Stripe.
The command establishes a direct connection with Stripe to send the request logs to your local machine.

Watch for all request logs sent from Stripe:

  $ stripe logs tail`),
		RunE: tailCmd.runTailCmd,
	}

	// Hidden configuration flags, useful for dev/debugging
	tailCmd.Cmd.Flags().StringVar(&tailCmd.apiBaseURL, "api-base", "", "Sets the API base URL")
	tailCmd.Cmd.Flags().MarkHidden("api-base") // #nosec G104

	tailCmd.Cmd.Flags().BoolVar(&tailCmd.noWSS, "no-wss", false, "Force unencrypted ws:// protocol instead of wss://")
	tailCmd.Cmd.Flags().MarkHidden("no-wss") // #nosec G104

	return tailCmd
}

func (tailCmd *LogsTailCmd) runTailCmd(cmd *cobra.Command, args []string) error {
	deviceName, err := Config.Profile.GetDeviceName()
	if err != nil {
		return err
	}

	key, err := Config.Profile.GetSecretKey()
	if err != nil {
		return err
	}

	tailer := logs.New(&logs.Config{
		DeviceName:          deviceName,
		Key:                 key,
		APIBaseURL:          tailCmd.apiBaseURL,
		WebSocketFeature:    requestLogsWebSocketFeature,
		Log:                 log.StandardLogger(),
		NoWSS:               tailCmd.noWSS,
	})

	err = tailer.Run()
	if err != nil {
		return err
	}

	return nil
}
