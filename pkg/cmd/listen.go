package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/stripe/stripe-cli/pkg/proxy"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

const webhooksWebSocketFeature = "webhooks"

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

	lc.cmd.Flags().StringSliceVar(&lc.forwardConnectHeaders, "connect-headers", []string{}, "A comma-separated list of custom headers to forward for Connect")
	lc.cmd.Flags().StringSliceVarP(&lc.events, "events", "e", []string{"*"}, "A comma-separated list of specific events to listen for. For a list of all possible events, see: https://stripe.com/docs/api/events/types")
	lc.cmd.Flags().StringVarP(&lc.forwardURL, "forward-to", "f", "", "The URL to forward webhook events to")
	lc.cmd.Flags().StringSliceVarP(&lc.forwardHeaders, "headers", "H", []string{}, "A comma-separated list of custom headers to forward")
	lc.cmd.Flags().StringVarP(&lc.forwardConnectURL, "forward-connect-to", "c", "", "The URL to forward Connect webhook events to (default: same as normal events)")
	lc.cmd.Flags().BoolVarP(&lc.latestAPIVersion, "latest", "l", false, "Receive events formatted with the latest API version (default: your account's default API version)")
	lc.cmd.Flags().BoolVar(&lc.livemode, "live", false, "Receive live events (default: test)")
	lc.cmd.Flags().BoolVarP(&lc.printJSON, "print-json", "j", false, "Print full JSON objects to stdout")
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

	// --print-secret option
	if lc.onlyPrintSecret {
		secret, err := proxy.GetSessionSecret(deviceName, key, lc.apiBaseURL)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", secret)
		return nil
	}

	// validate forward-urls args
	if lc.useConfiguredWebhooks && len(lc.forwardURL) > 0 {
		if strings.HasPrefix(lc.forwardURL, "/") {
			return errors.New("--forward-to cannot be a relative path when loading webhook endpoints from the API")
		}
		if strings.HasPrefix(lc.forwardConnectURL, "/") {
			return errors.New("--forward-connect-to cannot be a relative path when loading webhook endpoints from the API")
		}
	} else if lc.useConfiguredWebhooks && len(lc.forwardURL) == 0 {
		return errors.New("--load-from-webhooks-api requires a location to forward to with --forward-to")
	}

	p, err := proxy.Init(&proxy.Config{
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
		Log:                   log.StandardLogger(),
		NoWSS:                 lc.noWSS,
		Events:                lc.events,
	})
	if err != nil {
		return err
	}

	ctx := withSIGTERMCancel(context.Background(), func() {
		log.WithFields(log.Fields{
			"prefix": "proxy.Proxy.Run",
		}).Debug("Ctrl+C received, cleaning up...")
	})
	err = p.Run(ctx)
	if err != nil {
		return err
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
