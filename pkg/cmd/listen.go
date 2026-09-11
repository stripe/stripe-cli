package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/proxy"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

// thinEventPattern matches thin event types, which are namespaced by API
// version (e.g. v1.billing.meter.no_meter_found, v2.core.account.created).
var thinEventPattern = regexp.MustCompile(`^v\d+\.`)

const (
	webhooksWebSocketFeature     = "webhooks"
	destinationsWebSocketFeature = "v2_events"
	timeLayout                   = "2006-01-02 15:04:05"
	outputFormatJSON             = "JSON"

	// Accepted values for --events-from.
	eventsFromSelf     = "@self"
	eventsFromAccounts = "@accounts"
	eventsFromAll      = "all"
)

type listenCmd struct {
	cmd *cobra.Command

	forwardURL            string
	forwardHeaders        []string
	forwardConnectHeaders []string
	forwardConnectURL     string
	eventsFrom            string
	events                []string
	allSnapshot           bool
	allThin               bool

	// Deprecated event flags. Still functional so that published commands keep
	// working; they also cover the one case --forward-to can't express, a
	// separate destination per payload style.
	thinEvents            []string
	forwardThinURL        string
	forwardThinConnectURL string

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
	timeout               int64
	deviceToken           string
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
  stripe listen --all-snapshot --forward-to localhost:3000/events
  stripe listen --all-thin --forward-to localhost:3000/events
  stripe listen --events charge.captured,customer.created \
    --forward-to localhost:3000/events
  stripe listen --events v1.billing.meter.no_meter_found \
    --forward-to localhost:3000/events
  stripe listen --all-thin --events-from @accounts \
    --forward-to localhost:3000/events`,
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Use `--forward-to` to specify where events are sent, e.g. localhost:4242/webhook.\n" +
				"  Use `--events` to filter to specific event types, e.g. `--events checkout.session.completed`.\n" +
				"  Use `--format json` and pipe to `jq` for machine-readable event output.\n" +
				"  Use `--print-secret` to retrieve the webhook signing secret for signature verification.",
		},
		RunE: lc.runListenCmd,
	}

	lc.cmd.Flags().StringSliceVar(&lc.forwardConnectHeaders, "connect-headers", []string{}, "A comma-separated list of custom headers to forward for Connect. Ex: \"Key1:Value1, Key2:Value2\"")
	lc.cmd.Flags().StringSliceVarP(&lc.events, "events", "e", []string{}, "A comma-separated list of specific events to listen for. Accepts both snapshot events (e.g. charge.captured) and thin events (e.g. v1.billing.meter.no_meter_found). For a list of all possible events, see: https://stripe.com/docs/api/events/types")
	lc.cmd.Flags().BoolVar(&lc.allSnapshot, "all-snapshot", false, "Subscribe to all snapshot events")
	lc.cmd.Flags().BoolVar(&lc.allThin, "all-thin", false, "Subscribe to all thin events")
	lc.cmd.Flags().StringVar(&lc.eventsFrom, "events-from", eventsFromAll, "Which accounts to receive events from: '@self' (your account only), '@accounts' (connected accounts only), or 'all'")
	lc.cmd.Flags().StringVarP(&lc.forwardURL, "forward-to", "f", "", "The URL to forward events to")
	lc.cmd.Flags().StringSliceVarP(&lc.forwardHeaders, "headers", "H", []string{}, "A comma-separated list of custom headers to forward. Ex: \"Key1:Value1, Key2:Value2\"")
	lc.cmd.Flags().StringVarP(&lc.forwardConnectURL, "forward-connect-to", "c", "", "The URL to forward Connect events to (default: same as --forward-to)")

	// Deprecated flags. MarkDeprecated warns on use and hides them from help,
	// but they keep working; they'll be removed once the docs have migrated.
	lc.cmd.Flags().StringSliceVar(&lc.thinEvents, "thin-events", []string{}, "A comma-separated list of thin events to listen for.")
	lc.cmd.Flags().MarkDeprecated("thin-events", "use --events, which accepts both snapshot and thin event types, or --all-thin for all thin events.")
	lc.cmd.Flags().StringVar(&lc.forwardThinURL, "forward-thin-to", "", "The URL to forward thin events to")
	lc.cmd.Flags().MarkDeprecated("forward-thin-to", "use --forward-to together with --all-thin (or --events naming thin event types). Snapshot and thin events cannot share one destination.")
	lc.cmd.Flags().StringVar(&lc.forwardThinConnectURL, "forward-thin-connect-to", "", "The URL to forward thin Connect events to")
	lc.cmd.Flags().MarkDeprecated("forward-thin-connect-to", "use --events-from @accounts with --all-thin (or --events naming thin event types) and --forward-to <url>.")

	lc.cmd.Flags().BoolVarP(&lc.latestAPIVersion, "latest", "l", false, "Receive events formatted with the latest API version (default: your account's default API version)")
	lc.cmd.Flags().BoolVar(&lc.livemode, "live", false, "Receive live events (default: test)")
	lc.cmd.Flags().BoolVarP(&lc.printJSON, "print-json", "j", false, "Print full JSON objects to stdout.")
	lc.cmd.Flags().MarkDeprecated("print-json", "Please use `--format json` instead and use `jq` if you need to process the JSON in the terminal.")
	lc.cmd.Flags().StringVar(&lc.format, "format", "", `Specifies the output format of webhook events
	Acceptable values:
		'JSON' - Output webhook events in JSON format`)
	lc.cmd.Flags().BoolVarP(&lc.useConfiguredWebhooks, "use-configured-webhooks", "a", false, "Load webhook endpoint configuration from the webhooks API/dashboard")
	lc.cmd.Flags().BoolVarP(&lc.skipVerify, "skip-verify", "", false, "Skip certificate verification when forwarding to HTTPS endpoints")
	lc.cmd.Flags().BoolVar(&lc.onlyPrintSecret, "print-secret", false, "Only print the webhook signing secret and exit")
	lc.cmd.Flags().BoolVarP(&lc.skipUpdate, "skip-update", "s", false, "Skip checking latest version of Stripe CLI")

	// Hidden configuration flags, useful for dev/debugging
	lc.cmd.Flags().StringVar(&lc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	lc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	lc.cmd.Flags().BoolVar(&lc.noWSS, "no-wss", false, "Force unencrypted ws:// protocol instead of wss://")
	lc.cmd.Flags().MarkHidden("no-wss") // #nosec G104

	lc.cmd.Flags().Int64Var(&lc.timeout, "timeout", 30, "Sets timeout duration")
	lc.cmd.Flags().MarkHidden("timeout")

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
	// Validate flags before touching credentials or the network, so a bad flag
	// combination reports itself rather than failing on something downstream.
	if err := lc.validateFlags(); err != nil {
		return err
	}

	if err := stripe.ValidateAPIBaseURL(lc.apiBaseURL); err != nil {
		return err
	}

	if !lc.printJSON && !lc.onlyPrintSecret && !lc.skipUpdate {
		version.CheckLatestVersion()
	}

	deviceName, err := Config.Profile.GetDeviceName()
	if err != nil {
		return err
	}

	creds, err := Config.Profile.ResolveCredentials(lc.livemode)
	var mismatch *config.ActiveContextLivemodeMismatchError
	if errors.As(err, &mismatch) {
		if mismatch.ActiveLivemode {
			return errorcategory.UserInputErrorf("You're in live mode. Add --live to run the command, or run 'stripe switch context' to select a sandbox.")
		}
		return errorcategory.UserInputErrorf("You're in a sandbox. Remove --live to run the command, or run 'stripe switch context' to select a live account.")
	}
	if err != nil {
		return err
	}

	if strings.Contains(creds.Token, "sk_org") {
		log.Errorf("The listen command is not supported in an organization sandbox at this time.")
		return nil
	}

	apiBase, err := url.Parse(lc.apiBaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse API base url: %w", err)
	}

	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "proxy.Proxy.Run",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	client := &stripe.Client{
		BaseURL:     apiBase,
		Credentials: creds,
	}

	// --print-secret option
	if lc.onlyPrintSecret {
		secret, err := proxy.GetSessionSecret(ctx, client, deviceName)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", secret)
		return nil
	}

	accountID, _ := Config.Profile.GetAccountID()

	logger := log.StandardLogger()
	proxyVisitor := lc.createVisitor(logger, lc.format, lc.printJSON)
	proxyOutCh := make(chan websocket.IElement)

	snapshotEvents, thinEvents := lc.resolveEvents()
	directURL, connectURL := lc.resolveForwardURLs()
	thinURL, thinConnectURL := lc.resolveThinForwardURLs(directURL, connectURL)
	lc.warnUnforwardedThinEvents(logger)

	p, err := proxy.Init(ctx, &proxy.Config{
		Client:                client,
		DeviceName:            deviceName,
		DeviceToken:           &lc.deviceToken,
		ForwardURL:            directURL,
		ForwardThinURL:        thinURL,
		ForwardHeaders:        lc.forwardHeaders,
		ForwardConnectURL:     connectURL,
		ForwardThinConnectURL: thinConnectURL,
		ForwardConnectHeaders: lc.forwardConnectHeaders,
		UseConfiguredWebhooks: lc.useConfiguredWebhooks,
		WebSocketFeatures:     lc.getFeatures(),
		PrintJSON:             lc.printJSON,
		UseLatestAPIVersion:   lc.latestAPIVersion,
		SkipVerify:            lc.skipVerify,
		Log:                   logger,
		NoWSS:                 lc.noWSS,
		Timeout:               lc.timeout,
		Events:                snapshotEvents,
		ThinEvents:            thinEvents,
		OutCh:                 proxyOutCh,
		LoggedInAccountID:     accountID,
		EventsFrom:            lc.eventsFrom,
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

func (lc *listenCmd) createVisitor(logger *log.Logger, format string, printJSON bool) *websocket.Visitor {
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
				log.Errorf("%s", errStr)

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
			case proxy.V2EventPayload:
				if strings.ToUpper(format) == outputFormatJSON || printJSON {
					fmt.Println(de.Marshaled)
					return nil
				}

				maybeConnect := ""
				if data.IsConnect() {
					maybeConnect = "connect "
				}

				localTime := time.Now().Format(timeLayout)

				color := ansi.Color(os.Stdout)
				outputStr := fmt.Sprintf("%s   --> %s%s [%s]",
					color.Faint(localTime),
					color.BrightBlue(maybeConnect),
					ansi.Bold(data.Type),
					ansi.Linkify(data.ID, data.URLForEventID(lc.deviceToken), logger.Out),
				)
				fmt.Println(outputStr)
				return nil
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
						color.BrightBlue(maybeConnect),
						ansi.Linkify(ansi.Bold(data.Type), data.URLForEventType(), logger.Out),
						ansi.Linkify(data.ID, data.URLForEventID(), logger.Out),
					)
					fmt.Println(outputStr)
				}
				return nil
			case proxy.EndpointResponse:
				event := data.Event
				resp := data.Resp
				v2Event := data.V2Event
				var link string
				if event != nil {
					link = ansi.Linkify(event.ID, event.URLForEventID(), logger.Out)
				} else if v2Event != nil {
					link = ansi.Linkify(v2Event.ID, v2Event.URLForEventID(lc.deviceToken), logger.Out)
				}
				localTime := time.Now().Format(timeLayout)

				color := ansi.Color(os.Stdout)
				outputStr := fmt.Sprintf("%s  <--  [%d] %s %s [%s]",
					color.Faint(localTime),
					ansi.ColorizeStatus(resp.StatusCode),
					resp.Request.Method,
					resp.Request.URL,
					link,
				)
				fmt.Println(outputStr)
				return nil
			default:
				return errorcategory.Errorf(errorcategory.Internal, "VisitData received unexpected type for DataElement, got %T", de)
			}
		},
	}
}

// getFeatures derives the websocket features from the resolved subscription, so
// that the channels opened always match the events actually subscribed to.
func (lc *listenCmd) getFeatures() []string {
	snapshotEvents, thinEvents := lc.resolveEvents()

	features := []string{}
	if len(snapshotEvents) > 0 {
		features = append(features, webhooksWebSocketFeature)
	}
	if len(thinEvents) > 0 {
		features = append(features, destinationsWebSocketFeature)
	}

	return features
}

func isThinEvent(eventType string) bool {
	return thinEventPattern.MatchString(eventType)
}

// usesDeprecatedThinFlags reports whether the invocation drives the thin side
// through the deprecated flags rather than --events / --forward-to.
func (lc *listenCmd) usesDeprecatedThinFlags() bool {
	return len(lc.thinEvents) > 0 || lc.forwardThinURL != "" || lc.forwardThinConnectURL != ""
}

// namesSnapshotEvents reports whether --events named any snapshot event type.
func (lc *listenCmd) namesSnapshotEvents() bool {
	for _, e := range lc.events {
		if !isThinEvent(e) {
			return true
		}
	}

	return false
}

// resolveEvents splits the event flags into the snapshot and thin event lists
// the proxy subscribes with.
func (lc *listenCmd) resolveEvents() (snapshotEvents []string, thinEvents []string) {
	allSnapshot := lc.allSnapshot
	allThin := lc.allThin

	// Deprecated --thin-events folds into the thin subscription. Its values are
	// thin by declaration, so they skip the event-type sniffing below.
	var legacyThin []string
	for _, e := range lc.thinEvents {
		if e == "*" {
			allThin = true
			continue
		}
		legacyThin = append(legacyThin, e)
	}

	// The deprecated flags ran alongside an --events default of "*", so a legacy
	// invocation that never named snapshot events still subscribed to them all.
	if lc.usesDeprecatedThinFlags() && !allSnapshot && !lc.namesSnapshotEvents() {
		allSnapshot = true
	}

	return splitEventsByType(lc.events, allSnapshot, allThin, legacyThin...)
}

// splitEventsByType separates an event list into snapshot and thin event lists.
// extraThin holds event types already known to be thin.
func splitEventsByType(events []string, allSnapshot, allThin bool, extraThin ...string) (snapshotEvents []string, thinEvents []string) {
	if allSnapshot {
		snapshotEvents = append(snapshotEvents, "*")
	}
	if allThin {
		thinEvents = append(thinEvents, "*")
	}

	for _, e := range events {
		if isThinEvent(e) {
			thinEvents = append(thinEvents, e)
		} else {
			snapshotEvents = append(snapshotEvents, e)
		}
	}

	thinEvents = append(thinEvents, extraThin...)

	// A bare "stripe listen" with no event flags subscribes to everything.
	if len(snapshotEvents) == 0 && len(thinEvents) == 0 {
		snapshotEvents = []string{"*"}
		thinEvents = []string{"*"}
	}

	return
}

// resolveForwardURLs determines the direct and Connect forwarding URLs from
// --events-from, --forward-to, and --forward-connect-to.
func (lc *listenCmd) resolveForwardURLs() (directURL, connectURL string) {
	switch lc.eventsFrom {
	case eventsFromSelf:
		directURL = lc.forwardURL
		connectURL = ""
	case eventsFromAccounts:
		directURL = ""
		connectURL = lc.forwardURL
		if lc.forwardConnectURL != "" {
			connectURL = lc.forwardConnectURL
		}
	default: // eventsFromAll
		directURL = lc.forwardURL
		connectURL = lc.forwardConnectURL
		if connectURL == "" {
			connectURL = lc.forwardURL
		}
	}

	return
}

// resolveThinForwardURLs determines where thin events go. Normally they follow
// --forward-to alongside snapshot events.
//
// A deprecated invocation instead forwards thin events only to the destinations
// --forward-thin-to / --forward-thin-connect-to name. That covers the one
// arrangement the unified --forward-to can't express, a separate endpoint per
// payload style, and it keeps --thin-events with a bare --forward-to behaving as
// it always did: thin events were printed but never forwarded there.
func (lc *listenCmd) resolveThinForwardURLs(directURL, connectURL string) (thinURL, thinConnectURL string) {
	if !lc.usesDeprecatedThinFlags() {
		return directURL, connectURL
	}

	thinURL = lc.forwardThinURL
	thinConnectURL = lc.forwardThinConnectURL
	if thinConnectURL == "" {
		thinConnectURL = thinURL
	}

	return
}

// warnUnforwardedThinEvents flags the one deprecated shape that silently drops
// events: --thin-events with a forwarding destination that only snapshot events
// reach. Previously this printed thin events and forwarded nothing; say so
// rather than leaving the user to notice the gap.
func (lc *listenCmd) warnUnforwardedThinEvents(logger *log.Logger) {
	if !lc.usesDeprecatedThinFlags() || lc.forwardThinURL != "" || lc.forwardThinConnectURL != "" {
		return
	}

	directURL, connectURL := lc.resolveForwardURLs()
	if directURL == "" && connectURL == "" {
		return
	}

	logger.Warn("--thin-events without --forward-thin-to does not forward thin events; they are only printed here. Use --all-thin (or --events with thin event types) and --forward-to to forward them.")
}

// validateFlags rejects flag combinations that are contradictory, ambiguous, or
// no longer supported.
func (lc *listenCmd) validateFlags() error {
	if err := lc.validateEvents(); err != nil {
		return err
	}

	if err := lc.validateEventsFrom(); err != nil {
		return err
	}

	// --print-secret exits before forwarding anything, so don't hold it to the
	// forwarding rules.
	if lc.onlyPrintSecret {
		return nil
	}

	return lc.validateForwardingConfig()
}

// validateEvents rejects the "*" wildcard, which used to mean "all snapshot
// events" and is ambiguous now that --events also accepts thin events.
func (lc *listenCmd) validateEvents() error {
	for _, e := range lc.events {
		if e == "*" {
			return errorcategory.UserInputErrorf("--events '*' is no longer supported. Use --all-snapshot for all snapshot events, or --all-thin for all thin events.")
		}
	}

	return nil
}

func (lc *listenCmd) validateEventsFrom() error {
	switch lc.eventsFrom {
	case eventsFromSelf, eventsFromAccounts, eventsFromAll:
		return nil
	default:
		return errorcategory.UserInputErrorf("invalid --events-from value %q: must be one of '@self', '@accounts', or 'all'", lc.eventsFrom)
	}
}

// validateForwardingConfig checks that the forwarding flags name exactly one
// destination for exactly one payload style.
func (lc *listenCmd) validateForwardingConfig() error {
	if lc.forwardURL == "" && lc.forwardConnectURL == "" {
		return nil
	}

	// Forwarding requires an explicit subscription. Defaulting to everything
	// would POST every event on the account to the user's endpoint. The
	// deprecated flags carry their own subscription, so they satisfy this too.
	if !lc.allSnapshot && !lc.allThin && len(lc.events) == 0 && !lc.usesDeprecatedThinFlags() {
		return errorcategory.UserInputErrorf("must specify events to forward using --events, --all-snapshot, or --all-thin")
	}

	// --events-from @self drops Connect events, so a Connect destination would
	// never receive anything.
	if lc.eventsFrom == eventsFromSelf && lc.forwardConnectURL != "" {
		return errorcategory.UserInputErrorf("--forward-connect-to cannot be used with --events-from @self, which excludes events from connected accounts")
	}

	// Under @accounts both flags name the same destination, so two different
	// URLs is ambiguous.
	if lc.eventsFrom == eventsFromAccounts &&
		lc.forwardURL != "" && lc.forwardConnectURL != "" && lc.forwardURL != lc.forwardConnectURL {
		return errorcategory.UserInputErrorf("--forward-to and --forward-connect-to name the same destination when --events-from is @accounts: specify only one")
	}

	// The two payload styles are framed differently, so a single endpoint can't
	// receive both. Distinct destinations are fine, which is what the deprecated
	// --forward-thin-to still buys.
	snapshotEvents, thinEvents := lc.resolveEvents()
	if len(snapshotEvents) > 0 && len(thinEvents) > 0 {
		directURL, connectURL := lc.resolveForwardURLs()
		thinURL, thinConnectURL := lc.resolveThinForwardURLs(directURL, connectURL)
		if directURL != "" && directURL == thinURL {
			return errorcategory.UserInputErrorf("cannot forward both snapshot and thin events to the same destination")
		}
		if connectURL != "" && connectURL == thinConnectURL {
			return errorcategory.UserInputErrorf("cannot forward both snapshot and thin events to the same connect destination")
		}
	}

	return nil
}
