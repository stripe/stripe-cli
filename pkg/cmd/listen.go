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

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/proxy"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type listenCmd struct {
	cmd *cobra.Command

	forwardURL          string
	events              []string
	latestAPIVersion    bool
	loadFromWebhooksAPI bool
	printJSON           bool

	apiBaseURL   string
	noWSS        bool
	webSocketURL string
}

func newListenCmd() *listenCmd {
	lc := &listenCmd{}

	lc.cmd = &cobra.Command{
		Use:   "listen",
		Args:  validators.NoArgs,
		Short: "Listens for webhook events sent from Stripe to help test your integration.",
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
			ansi.Italic("⚠️  The Stripe CLI is in beta! Have feedback? Let us know, run: 'stripe feedback'. ⚠️"),
		),
		RunE: lc.runListenCmd,
	}

	lc.cmd.Flags().StringSliceVarP(&lc.events, "events", "e", []string{"*"}, "A comma-seperated list of which webhook events\nto listen for. For a list of all possible events, see:\nhttps://stripe.com/docs/api/events/types")
	lc.cmd.Flags().StringVarP(&lc.forwardURL, "forward-to", "f", "", "The URL to forward webhook events to")
	lc.cmd.Flags().BoolVarP(&lc.latestAPIVersion, "latest", "l", false, "Receive events formatted with the latest API version (default: your account's default API version)")
	lc.cmd.Flags().BoolVarP(&lc.printJSON, "print-json", "p", false, "Print full JSON objects to stdout")
	lc.cmd.Flags().BoolVarP(&lc.loadFromWebhooksAPI, "load-from-webhooks-api", "a", false, "Load webhook endpoint configuration from the webhooks API")

	// Hidden configuration flags, useful for dev/debugging
	lc.cmd.Flags().StringVar(&lc.apiBaseURL, "api-base", "", "Sets the API base URL")
	lc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	lc.cmd.Flags().BoolVar(&lc.noWSS, "no-wss", false, "Force unencrypted ws:// protocol instead of wss://")
	lc.cmd.Flags().MarkHidden("no-wss") // #nosec G104

	lc.cmd.Flags().StringVar(&lc.webSocketURL, "ws-url", "", "Sets the websocket URL")
	lc.cmd.Flags().MarkHidden("ws-url") // #nosec G104

	return lc
}

// Normally, this function would be listed alphabetically with the others declared in this file,
// but since it's acting as the core functionality for the cmd above, I'm keeping it close.
func (lc *listenCmd) runListenCmd(cmd *cobra.Command, args []string) error {
	deviceName, err := Profile.GetDeviceName()
	if err != nil {
		return err
	}

	endpointsMap := make(map[string][]string)

	key, err := Profile.GetSecretKey()
	if err != nil {
		return err
	}

	if len(lc.events) == 0 {
		lc.events = []string{"*"}
	}

	if len(lc.forwardURL) > 0 {
		endpointsMap[parseURL(lc.forwardURL)] = lc.events
	}

	if lc.loadFromWebhooksAPI && len(lc.forwardURL) > 0 {
		if strings.HasPrefix(lc.forwardURL, "/") {
			return errors.New("--forward-to cannot be a relative path when loading webhook endpoints from the API")
		}

		endpoints := lc.getEndpointsFromAPI(key)
		if len(endpoints.Data) == 0 {
			return errors.New("You have not defined any webhook endpoints on your account. Go to the Stripe Dashboard to add some: https://dashboard.stripe.com/test/webhooks")
		}

		endpointsMap = buildEndpointsMap(endpoints, parseURL(lc.forwardURL))
	} else if lc.loadFromWebhooksAPI && len(lc.forwardURL) == 0 {
		return errors.New("--load-from-webhooks-api requires a location to forward to with --forward-to")
	}

	p := proxy.New(&proxy.Config{
		DeviceName:          deviceName,
		Key:                 key,
		EndpointsMap:        endpointsMap,
		APIBaseURL:          lc.apiBaseURL,
		WebSocketURL:        lc.webSocketURL,
		PrintJSON:           lc.printJSON,
		UseLatestAPIVersion: lc.latestAPIVersion,
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
	examples := requests.Examples{
		Profile:    Profile,
		APIVersion: "2019-03-14",
		SecretKey:  secretKey,
	}
	return examples.WebhookEndpointsList()
}

func buildEndpointsMap(endpoints requests.WebhookEndpointList, forwardURL string) map[string][]string {
	endpointsMap := make(map[string][]string)
	for _, endpoint := range endpoints.Data {
		u, err := url.Parse(endpoint.URL)
		// Silently skip over invalid paths
		if err == nil {
			// Since webhooks in the dashboard may have a more generic url, only extract
			// the path. We'll use this with `localhost` or with the `--forward-to` flag
			endpointsMap[path.Join(forwardURL, u.Path)] = endpoint.EnabledEvents
		}
	}

	return endpointsMap
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
