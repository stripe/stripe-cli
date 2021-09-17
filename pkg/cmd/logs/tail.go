package logs

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"context"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/logtailing"
	logTailing "github.com/stripe/stripe-cli/pkg/logtailing"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

const outputFormatJSON = "JSON"

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

func withSIGTERMCancel(ctx context.Context, onCancel func()) context.Context {
	// Create a context that will be canceled when Ctrl+C is pressed
	ctx, cancel := context.WithCancel(ctx)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-interruptCh
		onCancel()
		cancel()
	}()
	return ctx
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

	logger := log.StandardLogger()

	logtailingVisitor := createVisitor(logger, tailCmd.format)

	logtailingOutCh := make(chan websocket.IElement)

	tailer := logTailing.New(&logTailing.Config{
		APIBaseURL: tailCmd.apiBaseURL,
		DeviceName: deviceName,
		Filters:    tailCmd.LogFilters,
		Key:        key,
		Log:        logger,
		NoWSS:      tailCmd.noWSS,
		OutCh:      logtailingOutCh,
	})

	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "logtailing.Tailer.Run",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	go tailer.Run(ctx)

	for el := range logtailingOutCh {
		err := el.Accept(logtailingVisitor)
		if err != nil {
			return err
		}
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

func createVisitor(logger *log.Logger, format string) *websocket.Visitor {
	var s *spinner.Spinner

	return &websocket.Visitor{
		VisitError: func(ee websocket.ErrorElement) error {
			ansi.StopSpinner(s, "", logger.Out)
			return ee.Error
		},
		VisitWarning: func(we websocket.WarningElement) error {
			color := ansi.Color(os.Stdout)
			fmt.Printf("%s %s\n", color.Yellow("Warning"), we.Warning)
			return nil
		},
		VisitStatus: func(se websocket.StateElement) error {
			switch se.State {
			case websocket.Loading:
				s = ansi.StartNewSpinner("Getting ready...", logger.Out)
			case websocket.Reconnecting:
				ansi.StartSpinner(s, "Session expired, reconnecting...", logger.Out)
			case websocket.Ready:
				ansi.StopSpinner(s, "Ready! You're now waiting to receive API request logs (^C to quit)", logger.Out)
			case websocket.Done:
				ansi.StopSpinner(s, "", logger.Out)
			}
			return nil
		},
		VisitData: func(de websocket.DataElement) error {
			log, ok := de.Data.(logtailing.EventPayload)
			if !ok {
				return fmt.Errorf("VisitData received unexpected type for DataElement, got %T expected %T", de, logtailing.EventPayload{})
			}

			if strings.ToUpper(format) == outputFormatJSON {
				fmt.Println(ansi.ColorizeJSON(de.Marshaled, false, os.Stdout))
				return nil
			}

			coloredStatus := ansi.ColorizeStatus(log.Status)

			url := urlForRequestID(&log)
			requestLink := ansi.Linkify(log.RequestID, url, os.Stdout)

			if log.URL == "" {
				log.URL = "[View path in dashboard]"
			}

			exampleLayout := "2006-01-02 15:04:05"
			localTime := time.Unix(int64(log.CreatedAt), 0).Format(exampleLayout)

			color := ansi.Color(os.Stdout)
			outputStr := fmt.Sprintf("%s [%d] %s %s [%s]", color.Faint(localTime), coloredStatus, log.Method, log.URL, requestLink)
			fmt.Println(outputStr)

			errorValues := reflect.ValueOf(&log.Error).Elem()
			errType := errorValues.Type()

			for i := 0; i < errorValues.NumField(); i++ {
				fieldValue := errorValues.Field(i).Interface()
				if fieldValue != "" {
					fmt.Printf("%s: %s\n", errType.Field(i).Name, fieldValue)
				}
			}
			return nil
		},
	}
}

func urlForRequestID(payload *logtailing.EventPayload) string {
	maybeTest := ""
	if !payload.Livemode {
		maybeTest = "/test"
	}

	return fmt.Sprintf("https://dashboard.stripe.com%s/logs/%s", maybeTest, payload.RequestID)
}
