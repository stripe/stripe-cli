package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/stripe/stripe-cli/pkg/proxy"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
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
	deviceName, err := Config.Profile.GetDeviceName()
	if err != nil {
		return err
	}

	endpointRoutes := make([]proxy.EndpointRoute, 0)

	key, err := Config.Profile.GetAPIKey(lc.livemode)
	if err != nil {
		return err
	}

	if !lc.printJSON && !lc.onlyPrintSecret && !lc.skipUpdate {
		version.CheckLatestVersion()
	}

	for _, event := range lc.events {
		if _, found := validEvents[event]; !found {
			fmt.Printf("Warning: You're attempting to listen for \"%s\", which isn't a valid event\n", event)
		}
	}

	if len(lc.events) == 0 {
		lc.events = []string{"*"}
	}

	if len(lc.forwardConnectURL) == 0 {
		lc.forwardConnectURL = lc.forwardURL
	}

	// default to non connect headers if no forward connect headers
	if len(lc.forwardConnectHeaders) == 0 {
		lc.forwardConnectHeaders = lc.forwardHeaders
	}

	if len(lc.forwardURL) > 0 {
		endpointRoutes = append(endpointRoutes, proxy.EndpointRoute{
			URL:            parseURL(lc.forwardURL),
			ForwardHeaders: lc.forwardHeaders,
			Connect:        false,
			EventTypes:     lc.events,
		})
	}

	if len(lc.forwardConnectURL) > 0 {
		endpointRoutes = append(endpointRoutes, proxy.EndpointRoute{
			URL:            parseURL(lc.forwardConnectURL),
			ForwardHeaders: lc.forwardConnectHeaders,
			Connect:        true,
			EventTypes:     lc.events,
		})
	}

	if lc.useConfiguredWebhooks && len(lc.forwardURL) > 0 {
		if strings.HasPrefix(lc.forwardURL, "/") {
			return errors.New("--forward-to cannot be a relative path when loading webhook endpoints from the API")
		}

		if strings.HasPrefix(lc.forwardConnectURL, "/") {
			return errors.New("--forward-connect-to cannot be a relative path when loading webhook endpoints from the API")
		}

		endpoints := lc.getEndpointsFromAPI(key)
		if len(endpoints.Data) == 0 {
			return errors.New("You have not defined any webhook endpoints on your account. Go to the Stripe Dashboard to add some: https://dashboard.stripe.com/test/webhooks")
		}

		endpointRoutes = buildEndpointRoutes(endpoints, parseURL(lc.forwardURL), parseURL(lc.forwardConnectURL), lc.forwardHeaders, lc.forwardConnectHeaders)
	} else if lc.useConfiguredWebhooks && len(lc.forwardURL) == 0 {
		return errors.New("--load-from-webhooks-api requires a location to forward to with --forward-to")
	}

	p := proxy.New(&proxy.Config{
		DeviceName:          deviceName,
		Key:                 key,
		EndpointRoutes:      endpointRoutes,
		APIBaseURL:          lc.apiBaseURL,
		WebSocketFeature:    webhooksWebSocketFeature,
		PrintJSON:           lc.printJSON,
		UseLatestAPIVersion: lc.latestAPIVersion,
		SkipVerify:          lc.skipVerify,
		Log:                 log.StandardLogger(),
		NoWSS:               lc.noWSS,
	}, lc.events)

	if lc.onlyPrintSecret {
		secret, err := p.GetSessionSecret(context.Background())
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", secret)
		return nil
	}

	err = p.Run(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (lc *listenCmd) getEndpointsFromAPI(secretKey string) requests.WebhookEndpointList {
	apiBaseURL := lc.apiBaseURL
	if apiBaseURL == "" {
		apiBaseURL = stripe.DefaultAPIBaseURL
	}

	return requests.WebhookEndpointsList(apiBaseURL, "2019-03-14", secretKey, &Config.Profile)
}

func buildEndpointRoutes(endpoints requests.WebhookEndpointList, forwardURL, forwardConnectURL string, forwardHeaders []string, forwardConnectHeaders []string) []proxy.EndpointRoute {
	endpointRoutes := make([]proxy.EndpointRoute, 0)

	for _, endpoint := range endpoints.Data {
		u, err := url.Parse(endpoint.URL)
		// Silently skip over invalid paths
		if err == nil {
			// Since webhooks in the dashboard may have a more generic url, only extract
			// the path. We'll use this with `localhost` or with the `--forward-to` flag
			if endpoint.Application == "" {
				endpointRoutes = append(endpointRoutes, proxy.EndpointRoute{
					URL:            buildForwardURL(forwardURL, u),
					ForwardHeaders: forwardHeaders,
					Connect:        false,
					EventTypes:     endpoint.EnabledEvents,
				})
			} else {
				endpointRoutes = append(endpointRoutes, proxy.EndpointRoute{
					URL:            buildForwardURL(forwardConnectURL, u),
					ForwardHeaders: forwardConnectHeaders,
					Connect:        true,
					EventTypes:     endpoint.EnabledEvents,
				})
			}
		}
	}

	return endpointRoutes
}

// parseURL parses the potentially incomplete URL provided in the configuration
// and returns a full URL
func parseURL(url string) string {
	_, err := strconv.Atoi(url)
	if err == nil {
		// If the input is just a number, assume it's a port number
		url = "localhost:" + url
	}

	if strings.HasPrefix(url, "/") {
		// If the input starts with a /, assume it's a relative path
		url = "localhost" + url
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		// Add the protocol if it's not already there
		url = "http://" + url
	}

	return url
}

func buildForwardURL(forwardURL string, destination *url.URL) string {
	f, err := url.Parse(forwardURL)
	if err != nil {
		log.Fatalf("Provided forward url cannot be parsed: %s", forwardURL)
	}

	return fmt.Sprintf(
		"%s://%s%s%s",
		f.Scheme,
		f.Host,
		strings.TrimSuffix(f.Path, "/"), // avoids having a double "//"
		destination.Path,
	)
}
