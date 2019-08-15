package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/proxy"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const webhooksWebSocketFeature = "webhooks"

type listenCmd struct {
	cmd *cobra.Command

	forwardURL          string
	forwardConnectURL   string
	events              []string
	latestAPIVersion    bool
	loadFromWebhooksAPI bool
	printJSON           bool
	skipVerify          bool

	apiBaseURL string
	noWSS      bool
}

func newListenCmd() *listenCmd {
	lc := &listenCmd{}

	lc.cmd = &cobra.Command{
		Use:   "listen",
		Args:  validators.NoArgs,
		Short: "Listen for webhook events",
		Long: fmt.Sprintf(`%s

The listen command lets you watch for and forward webhook events from Stripe.
The command establishes a direct connection with Stripe to send the webhook
events to your local machine. With that, you can either leave it open to see
webhooks come in, filter on specific events, or forward the events to a local
instance of your application.

Watch for all events sent from Stripe:

  $ stripe listen

Start listening for 'charge.created' and 'charge.updated' events and forward
to your localhost:

  $ stripe listen --events charge.created,charge.updated --forward-to localhost:9000/events`,
			getBanner(),
		),
		RunE: lc.runListenCmd,
	}

	lc.cmd.Flags().StringSliceVarP(&lc.events, "events", "e", []string{"*"}, "A comma-separated list of which webhook events\nto listen for. For a list of all possible events, see:\nhttps://stripe.com/docs/api/events/types")
	lc.cmd.Flags().StringVarP(&lc.forwardURL, "forward-to", "f", "", "The URL to forward webhook events to")
	lc.cmd.Flags().StringVarP(&lc.forwardConnectURL, "forward-connect-to", "c", "", "The URL to forward Connect webhook events to (default: same as normal events)")
	lc.cmd.Flags().BoolVarP(&lc.latestAPIVersion, "latest", "l", false, "Receive events formatted with the latest API version (default: your account's default API version)")
	lc.cmd.Flags().BoolVarP(&lc.printJSON, "print-json", "p", false, "Print full JSON objects to stdout")
	lc.cmd.Flags().BoolVarP(&lc.loadFromWebhooksAPI, "load-from-webhooks-api", "a", false, "Load webhook endpoint configuration from the webhooks API")
	lc.cmd.Flags().BoolVarP(&lc.skipVerify, "skip-verify", "", false, "Skip certificate verification when forwarding to HTTPS endpoints")

	// Hidden configuration flags, useful for dev/debugging
	lc.cmd.Flags().StringVar(&lc.apiBaseURL, "api-base", "", "Sets the API base URL")
	lc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	lc.cmd.Flags().BoolVar(&lc.noWSS, "no-wss", false, "Force unencrypted ws:// protocol instead of wss://")
	lc.cmd.Flags().MarkHidden("no-wss") // #nosec G104

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

	key, err := Config.Profile.GetAPIKey()
	if err != nil {
		return err
	}

	if len(lc.events) == 0 {
		lc.events = []string{"*"}
	}

	if len(lc.forwardConnectURL) == 0 {
		lc.forwardConnectURL = lc.forwardURL
	}

	if len(lc.forwardURL) > 0 {
		endpointRoutes = append(endpointRoutes, proxy.EndpointRoute{
			URL:        parseURL(lc.forwardURL),
			Connect:    false,
			EventTypes: lc.events,
		})
		endpointRoutes = append(endpointRoutes, proxy.EndpointRoute{
			URL:        parseURL(lc.forwardConnectURL),
			Connect:    true,
			EventTypes: lc.events,
		})
	}

	if lc.loadFromWebhooksAPI && len(lc.forwardURL) > 0 {
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

		endpointRoutes = buildEndpointRoutes(endpoints, parseURL(lc.forwardURL), parseURL(lc.forwardConnectURL))
	} else if lc.loadFromWebhooksAPI && len(lc.forwardURL) == 0 {
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
	})

	err = p.Run()
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

	examples := requests.Examples{
		Profile:    Config.Profile,
		APIVersion: "2019-03-14",
		APIKey:     secretKey,
		APIBaseURL: apiBaseURL,
	}
	return examples.WebhookEndpointsList()
}

func buildEndpointRoutes(endpoints requests.WebhookEndpointList, forwardURL, forwardConnectURL string) []proxy.EndpointRoute {
	endpointRoutes := make([]proxy.EndpointRoute, 0)
	for _, endpoint := range endpoints.Data {
		u, err := url.Parse(endpoint.URL)
		// Silently skip over invalid paths
		if err == nil {
			// Since webhooks in the dashboard may have a more generic url, only extract
			// the path. We'll use this with `localhost` or with the `--forward-to` flag
			if endpoint.Application == "" {
				endpointRoutes = append(endpointRoutes, proxy.EndpointRoute{
					URL:        buildForwardURL(forwardURL, u),
					Connect:    false,
					EventTypes: endpoint.EnabledEvents,
				})
			} else {
				endpointRoutes = append(endpointRoutes, proxy.EndpointRoute{
					URL:        buildForwardURL(forwardConnectURL, u),
					Connect:    true,
					EventTypes: endpoint.EnabledEvents,
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

	return fmt.Sprintf("%s://%s", f.Scheme, path.Join(f.Host, destination.Path))
}
