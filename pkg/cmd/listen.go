package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/proxy"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

const webhooksWebSocketFeature = "webhooks"
const timeLayout = "2006-01-02 15:04:05"
const outputFormatJSON = "JSON"

type listenCmd struct {
	cmd *cobra.Command

	forwardURL            string
	forwardHeaders        []string
	forwardConnectHeaders []string
	forwardConnectURL     string
	events                []string
	latestAPIVersion      bool
	livemode              bool
	useConfiguredWebhooks bool
	printJSON             bool
	format                string
	skipVerify            bool
	onlyPrintSecret       bool
	skipUpdate            bool
	apiBaseURL            string
	noWSS                 bool
}

func newListenCmd() *listenCmd {
	lc := &listenCmd{}

	lc.cmd = &cobra.Command{
		Use:   "listen",
		Args:  validators.NoArgs,
		Short: "Listen for webhook events",
		Long: `The listen command watches and forwards webhook events from Stripe to your
local machine by connecting directly to Stripe's API. You can test the latest
API version, filter events, or even load your saved webhook endpoints from your
Stripe account.`,
		Example: `stripe listen
  stripe listen --events charge.captured,charge.updated \
    --forward-to localhost:3000/events`,
		RunE: lc.runListenCmd,
	}

	lc.cmd.Flags().StringSliceVar(&lc.forwardConnectHeaders, "connect-headers", []string{}, "A comma-separated list of custom headers to forward for Connect. Ex: \"Key1:Value1, Key2:Value2\"")
	lc.cmd.Flags().StringSliceVarP(&lc.events, "events", "e", []string{"*"}, "A comma-separated list of specific events to listen for. For a list of all possible events, see: https://stripe.com/docs/api/events/types")
	lc.cmd.Flags().StringVarP(&lc.forwardURL, "forward-to", "f", "", "The URL to forward webhook events to")
	lc.cmd.Flags().StringSliceVarP(&lc.forwardHeaders, "headers", "H", []string{}, "A comma-separated list of custom headers to forward. Ex: \"Key1:Value1, Key2:Value2\"")
	lc.cmd.Flags().StringVarP(&lc.forwardConnectURL, "forward-connect-to", "c", "", "The URL to forward Connect webhook events to (default: same as normal events)")
	lc.cmd.Flags().BoolVarP(&lc.latestAPIVersion, "latest", "l", false, "Receive events formatted with the latest API version (default: your account's default API version)")
	lc.cmd.Flags().BoolVar(&lc.livemode, "live", false, "Receive live events (default: test)")
	lc.cmd.Flags().BoolVarP(&lc.printJSON, "print-json", "j", false, "Print full JSON objects to stdout.")
	lc.cmd.Flags().MarkDeprecated("print-json", "Please use `--format JSON` instead and use `jq` if you need to process the JSON in the terminal.")
	lc.cmd.Flags().StringVar(&lc.format, "format", "", `Specifies the output format of webhook events
	Acceptable values:
		'JSON' - Output webhook events in JSON format`)
	lc.cmd.Flags().BoolVarP(&lc.useConfiguredWebhooks, "use-configured-webhooks", "a", false, "Load webhook endpoint configuration from the webhooks API/dashboard")
	lc.cmd.Flags().BoolVarP(&lc.skipVerify, "skip-verify", "", false, "Skip certificate verification when forwarding to HTTPS endpoints")
	lc.cmd.Flags().BoolVar(&lc.onlyPrintSecret, "print-secret", false, "Only print the webhook signing secret and exit")
	lc.cmd.Flags().BoolVarP(&lc.skipUpdate, "skip-update", "s", false, "Skip checking latest version of Stripe CLI")

	// Hidden configuration flags, useful for dev/debugging
	lc.cmd.Flags().StringVar(&lc.apiBaseURL, "api-base", "", "Sets the API base URL")
	lc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	lc.cmd.Flags().BoolVar(&lc.noWSS, "no-wss", false, "Force unencrypted ws:// protocol instead of wss://")
	lc.cmd.Flags().MarkHidden("no-wss") // #nosec G104

	// renamed --load-from-webhooks-api to --use-configured-webhooks,  but want to keep backward compatibility
	lc.cmd.Flags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == "load-from-webhooks-api" {
			name = "use-configured-webhooks"
		}
		return pflag.NormalizedName(name)
	})

	return lc
}

// Normally, this function would be listed alphabetically with the others declared in this file,
// but since it's acting as the core functionality for the cmd above, I'm keeping it close.
func (lc *listenCmd) runListenCmd(cmd *cobra.Command, args []string) error {
	if !lc.printJSON && !lc.onlyPrintSecret && !lc.skipUpdate {
		version.CheckLatestVersion()
	}

	deviceName, err := Config.Profile.GetDeviceName()
	if err != nil {
		return err
	}

	key, err := Config.Profile.GetAPIKey(lc.livemode)
	if err != nil {
		return err
	}

	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "proxy.Proxy.Run",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	// --print-secret option
	if lc.onlyPrintSecret {
		secret, err := proxy.GetSessionSecret(ctx, deviceName, key, lc.apiBaseURL)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", secret)
		return nil
	}

	logger := log.StandardLogger()
	proxyVisitor := createVisitor(logger, lc.format, lc.printJSON)
	proxyOutCh := make(chan websocket.IElement)

	p, err := proxy.Init(ctx, &proxy.Config{
		DeviceName:            deviceName,
		Key:                   key,
		ForwardURL:            lc.forwardURL,
		ForwardHeaders:        lc.forwardHeaders,
		ForwardConnectURL:     lc.forwardConnectURL,
		ForwardConnectHeaders: lc.forwardConnectHeaders,
		UseConfiguredWebhooks: lc.useConfiguredWebhooks,
		APIBaseURL:            lc.apiBaseURL,
		WebSocketFeature:      webhooksWebSocketFeature,
		PrintJSON:             lc.printJSON,
		UseLatestAPIVersion:   lc.latestAPIVersion,
		SkipVerify:            lc.skipVerify,
		Log:                   logger,
		NoWSS:                 lc.noWSS,
		Events:                lc.events,
		OutCh:                 proxyOutCh,
	})
	if err != nil {
		return err
	}

	go p.Run(ctx)

	for el := range proxyOutCh {
		err := el.Accept(proxyVisitor)
		if err != nil {
			return err
		}
	}

	return nil
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

func createVisitor(logger *log.Logger, format string, printJSON bool) *websocket.Visitor {
	var s *spinner.Spinner

	return &websocket.Visitor{
		VisitError: func(ee websocket.ErrorElement) error {
			ansi.StopSpinner(s, "", logger.Out)
			switch ee.Error.(type) {
			case proxy.FailedToPostError:
				color := ansi.Color(os.Stdout)
				localTime := time.Now().Format(timeLayout)

				errStr := fmt.Sprintf("%s            [%s] Failed to POST: %v\n",
					color.Faint(localTime),
					color.Red("ERROR"),
					ee.Error,
				)
				fmt.Println(errStr)

				// Don't exit program
				return nil
			case proxy.FailedToReadResponseError:
				color := ansi.Color(os.Stdout)
				localTime := time.Now().Format(timeLayout)

				errStr := fmt.Sprintf("%s            [%s] Failed to read response from endpoint, error = %v\n",
					color.Faint(localTime),
					color.Red("ERROR"),
					ee.Error,
				)
				log.Errorf(errStr)

				// Don't exit program
				return nil
			default:
				logger.Fatal(ee.Error)
				return ee.Error
			}
		},
		VisitStatus: func(se websocket.StateElement) error {
			switch se.State {
			case websocket.Loading:
				s = ansi.StartNewSpinner("Getting ready...", logger.Out)
			case websocket.Reconnecting:
				ansi.StartSpinner(s, "Session expired, reconnecting...", logger.Out)
			case websocket.Ready:
				ansi.StopSpinner(s, fmt.Sprintf("Ready! %sYour webhook signing secret is %s (^C to quit)", se.Data[0], ansi.Bold(se.Data[1])), logger.Out)
			case websocket.Done:
				ansi.StopSpinner(s, "", logger.Out)
			}
			return nil
		},
		VisitData: func(de websocket.DataElement) error {
			switch data := de.Data.(type) {
			case proxy.StripeEvent:
				if strings.ToUpper(format) == outputFormatJSON || printJSON {
					fmt.Println(de.Marshaled)
				} else {
					maybeConnect := ""
					if data.IsConnect() {
						maybeConnect = "connect "
					}

					localTime := time.Now().Format(timeLayout)

					color := ansi.Color(os.Stdout)
					outputStr := fmt.Sprintf("%s   --> %s%s [%s]",
						color.Faint(localTime),
						maybeConnect,
						ansi.Linkify(ansi.Bold(data.Type), data.URLForEventType(), logger.Out),
						ansi.Linkify(data.ID, data.URLForEventID(), logger.Out),
					)
					fmt.Println(outputStr)
				}
				return nil
			case proxy.EndpointResponse:
				event := data.Event
				resp := data.Resp
				localTime := time.Now().Format(timeLayout)

				color := ansi.Color(os.Stdout)
				outputStr := fmt.Sprintf("%s  <--  [%d] %s %s [%s]",
					color.Faint(localTime),
					ansi.ColorizeStatus(resp.StatusCode),
					resp.Request.Method,
					resp.Request.URL,
					ansi.Linkify(event.ID, event.URLForEventID(), logger.Out),
				)
				fmt.Println(outputStr)
				return nil
			default:
				return fmt.Errorf("VisitData received unexpected type for DataElement, got %T", de)
			}
		},
	}
}
