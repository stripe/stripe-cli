package logs

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"context"

	"github.com/stripe/stripe-cli/pkg/config"
	logTailing "github.com/stripe/stripe-cli/pkg/logtailing"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

const requestLogsWebSocketFeature = "request_logs"

// TailCmd wraps the configuration for the tail command
type TailCmd struct {
	apiBaseURL string
	cfg        *config.Config
	Cmd        *cobra.Command
	format     string
	LogFilters *logTailing.LogFilters
	noWSS      bool
}

// NewTailCmd creates and initializes the tail command for the logs package
func NewTailCmd(config *config.Config) *TailCmd {
	tailCmd := &TailCmd{
		cfg:        config,
		LogFilters: &logTailing.LogFilters{},
	}

	tailCmd.Cmd = &cobra.Command{
		Use:   "tail",
		Args:  validators.NoArgs,
		Short: "Tail API request logs from your Stripe requests.",
		Long: `View API request logs in real-time as they are made to your Stripe account.
Log tailing allows you to filter data similarly to the Stripe Dashboard; filter
HTTP methods, IP addresses, paths, response status, and more.`,
		Example: `stripe logs tail
  stripe logs tail --filter-http-methods GET
  stripe logs tail --filter-status-code-type 4XX`,
		RunE: tailCmd.runTailCmd,
	}

	tailCmd.Cmd.Flags().StringVar(
		&tailCmd.format,
		"format",
		"",
		`Specifies the output format of request logs
Acceptable values:
	'JSON' - Output logs in JSON format`,
	)

	// Log filters
	tailCmd.Cmd.Flags().StringSliceVar(
		&tailCmd.LogFilters.FilterAccount,
		"filter-account",
		[]string{},
		`*CONNECT ONLY* Filter request logs by source and destination account
Acceptable values:
	'connect_in'  - Incoming connect requests
	'connect_out' - Outgoing connect requests
	'self'        - Non-connect requests`,
	)
	tailCmd.Cmd.Flags().StringSliceVar(&tailCmd.LogFilters.FilterIPAddress, "filter-ip-address", []string{}, "Filter request logs by ip address")
	tailCmd.Cmd.Flags().StringSliceVar(
		&tailCmd.LogFilters.FilterHTTPMethod,
		"filter-http-method",
		[]string{},
		`Filter request logs by http method
Acceptable values:
	'GET'    - HTTP get requests
	'POST'   - HTTP post requests
	'DELETE' - HTTP delete requests`,
	)
	tailCmd.Cmd.Flags().StringSliceVar(&tailCmd.LogFilters.FilterRequestPath, "filter-request-path", []string{}, "Filter request logs by request path")
	tailCmd.Cmd.Flags().StringSliceVar(
		&tailCmd.LogFilters.FilterRequestStatus,
		"filter-request-status",
		[]string{},
		`Filter request logs by request status
Acceptable values:
	'SUCCEEDED' - Requests that succeeded (status codes 200, 201, 202)
	'FAILED'    - Requests that failed`,
	)
	tailCmd.Cmd.Flags().StringSliceVar(
		&tailCmd.LogFilters.FilterSource,
		"filter-source",
		[]string{},
		`Filter request logs by source
Acceptable values:
	'API'       - Requests that came through the Stripe API
	'DASHBOARD' - Requests that came through the Stripe Dashboard`,
	)
	tailCmd.Cmd.Flags().StringSliceVar(&tailCmd.LogFilters.FilterStatusCode, "filter-status-code", []string{}, "Filter request logs by status code")
	tailCmd.Cmd.Flags().StringSliceVar(
		&tailCmd.LogFilters.FilterStatusCodeType,
		"filter-status-code-type",
		[]string{},
		`Filter request logs by status code type
Acceptable values:
	'2XX' - All 2XX status codes
	'4XX' - All 4XX status codes
	'5XX' - All 5XX status codes`,
	)

	// Hidden configuration flags, useful for dev/debugging
	tailCmd.Cmd.Flags().StringVar(&tailCmd.apiBaseURL, "api-base", "", "Sets the API base URL")
	tailCmd.Cmd.Flags().MarkHidden("api-base") // #nosec G104

	tailCmd.Cmd.Flags().BoolVar(&tailCmd.noWSS, "no-wss", false, "Force unencrypted ws:// protocol instead of wss://")
	tailCmd.Cmd.Flags().MarkHidden("no-wss") // #nosec G104

	return tailCmd
}

func (tailCmd *TailCmd) runTailCmd(cmd *cobra.Command, args []string) error {
	err := tailCmd.validateArgs()
	if err != nil {
		return err
	}

	err = tailCmd.convertArgs()
	if err != nil {
		return err
	}

	deviceName, err := tailCmd.cfg.Profile.GetDeviceName()
	if err != nil {
		return err
	}

	key, err := tailCmd.cfg.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	version.CheckLatestVersion()

	tailer := logTailing.New(&logTailing.Config{
		APIBaseURL:       tailCmd.apiBaseURL,
		DeviceName:       deviceName,
		Filters:          tailCmd.LogFilters,
		Key:              key,
		Log:              log.StandardLogger(),
		NoWSS:            tailCmd.noWSS,
		OutputFormat:     strings.ToUpper(tailCmd.format),
		WebSocketFeature: requestLogsWebSocketFeature,
	})

	err = tailer.Run(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (tailCmd *TailCmd) validateArgs() error {
	err := validators.CallNonEmptyArray(validators.Account, tailCmd.LogFilters.FilterAccount)
	if err != nil {
		return err
	}

	err = validators.CallNonEmptyArray(validators.HTTPMethod, tailCmd.LogFilters.FilterHTTPMethod)
	if err != nil {
		return err
	}

	err = validators.CallNonEmptyArray(validators.StatusCode, tailCmd.LogFilters.FilterStatusCode)
	if err != nil {
		return err
	}

	err = validators.CallNonEmptyArray(validators.StatusCodeType, tailCmd.LogFilters.FilterStatusCodeType)
	if err != nil {
		return err
	}

	err = validators.CallNonEmptyArray(validators.RequestSource, tailCmd.LogFilters.FilterSource)
	if err != nil {
		return err
	}

	err = validators.CallNonEmptyArray(validators.RequestStatus, tailCmd.LogFilters.FilterRequestStatus)
	if err != nil {
		return err
	}

	return nil
}

func (tailCmd *TailCmd) convertArgs() error {
	// The backend expects to receive the status code type as a string representing the start of the range (e.g., '200')
	if len(tailCmd.LogFilters.FilterStatusCodeType) > 0 {
		for i, code := range tailCmd.LogFilters.FilterStatusCodeType {
			tailCmd.LogFilters.FilterStatusCodeType[i] = strings.ReplaceAll(strings.ToUpper(code), "X", "0")
		}
	}

	return nil
}
