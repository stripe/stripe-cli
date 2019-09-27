package logtailing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/stripeauth"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

const outputFormatJSON = "JSON"

// LogFilters contains all of the potential user-provided filters for log tailing
type LogFilters struct {
	FilterAccount        []string `json:"filter_account,omitempty"`
	FilterIPAddress      []string `json:"filter_ip_address,omitempty"`
	FilterHTTPMethod     []string `json:"filter_http_method,omitempty"`
	FilterRequestPath    []string `json:"filter_request_path,omitempty"`
	FilterRequestStatus  []string `json:"filter_request_status,omitempty"`
	FilterSource         []string `json:"filter_source,omitempty"`
	FilterStatusCode     []string `json:"filter_status_code,omitempty"`
	FilterStatusCodeType []string `json:"filter_status_code_type,omitempty"`
}

// Config provides the configuration of a log tailer
type Config struct {
	APIBaseURL string

	// DeviceName is the name of the device sent to Stripe to help identify the device
	DeviceName string

	// Filters for API request logs
	Filters *LogFilters

	// Key is the API key used to authenticate with Stripe
	Key string

	// Info, error, etc. logger. Unrelated to API request logs.
	Log *log.Logger

	// Force use of unencrypted ws:// protocol instead of wss://
	NoWSS bool

	// Output format for request logs
	OutputFormat string

	// WebSocketFeature is the feature specified for the websocket connection
	WebSocketFeature string
}

// Tailer is the main interface for running the log tailing session
type Tailer struct {
	cfg *Config

	stripeAuthClient *stripeauth.Client
	webSocketClient  *websocket.Client

	interruptCh chan os.Signal
}

// EventPayload is the mapping for fields in event payloads from request log tailing
type EventPayload struct {
	CreatedAt int    `json:"created_at"`
	Livemode  bool   `json:"livemode"`
	Method    string `json:"method"`
	RequestID string `json:"request_id"`
	Status    int    `json:"status"`
	URL       string `json:"url"`
	Error     RedactedError `json:"error"`
}

// RedactedError is the mapping for fields in error from an EventPayload
type RedactedError struct {
	Type        string `json:"type"`
	Charge      string `json:"charge"`
	Code        string `json:"code"`
	DeclineCode string `json:"decline_code"`
	Message     string `json:"message"`
	Param       string `json:"param"`
}

// New creates a new Tailer
func New(cfg *Config) *Tailer {
	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}
	return &Tailer{
		cfg: cfg,
		stripeAuthClient: stripeauth.NewClient(cfg.Key, &stripeauth.Config{
			Log:        cfg.Log,
			APIBaseURL: cfg.APIBaseURL,
		}),
		interruptCh: make(chan os.Signal, 1),
	}
}

// Run sets the websocket connection
func (tailer *Tailer) Run() error {
	s := ansi.StartSpinner("Getting ready...", tailer.cfg.Log.Out)

	// Intercept Ctrl+c so we can do some clean up
	signal.Notify(tailer.interruptCh, os.Interrupt, syscall.SIGTERM)

	filters, err := jsonifyFilters(tailer.cfg.Filters)
	if err != nil {
		tailer.cfg.Log.Fatalf("Error while converting log filters to JSON encoding: %v", err)
	}

	session, err := tailer.stripeAuthClient.Authorize(tailer.cfg.DeviceName, tailer.cfg.WebSocketFeature, &filters)
	if err != nil {
		tailer.cfg.Log.Fatalf("Error while authenticating with Stripe: %v", err)
	}

	tailer.webSocketClient = websocket.NewClient(
		session.WebSocketURL,
		session.WebSocketID,
		session.WebSocketAuthorizedFeature,
		&websocket.Config{
			EventHandler:      websocket.EventHandlerFunc(tailer.processRequestLogEvent),
			Log:               tailer.cfg.Log,
			NoWSS:             tailer.cfg.NoWSS,
			ReconnectInterval: time.Duration(session.ReconnectDelay) * time.Second,
		},
	)
	go tailer.webSocketClient.Run()

	ansi.StopSpinner(s, "Ready! You're now waiting to receive API request logs (^C to quit)", tailer.cfg.Log.Out)

	if session.DisplayConnectFilterWarning {
		color := ansi.Color(os.Stdout)
		fmt.Println(fmt.Sprintf("%s you specified the 'account' filter for connect accounts but are not a connect merchant, so the filter will not be applied.", color.Yellow("Warning")))
	}

	// Block until Ctrl+C is received
	<-tailer.interruptCh

	log.WithFields(log.Fields{
		"prefix": "logs.Tailer.Run",
	}).Debug("Ctrl+C received, cleaning up...")

	if tailer.webSocketClient != nil {
		tailer.webSocketClient.Stop()
	}

	log.WithFields(log.Fields{
		"prefix": "logs.Tailer.Run",
	}).Debug("Bye!")

	return nil
}

func (tailer *Tailer) processRequestLogEvent(msg websocket.IncomingMessage) {
	if msg.RequestLogEvent == nil {
		tailer.cfg.Log.Warn("WebSocket specified for request logs received non-request-logs event")
		return
	}

	requestLogEvent := msg.RequestLogEvent

	tailer.cfg.Log.WithFields(log.Fields{
		"prefix":     "logs.Tailer.processRequestLogEvent",
		"webhook_id": requestLogEvent.RequestLogID,
	}).Debugf("Processing request log event")

	var payload EventPayload
	if err := json.Unmarshal([]byte(requestLogEvent.EventPayload), &payload); err != nil {
		tailer.cfg.Log.Warn("Received malformed payload: ", err)
	}

	// Don't show stripecli/sessions logs since they're generated by the CLI
	if payload.URL == "/v1/stripecli/sessions" {
		tailer.cfg.Log.Debug("Filtering out /v1/stripecli/sessions from logs")
		return
	}

	if tailer.cfg.OutputFormat == outputFormatJSON {
		fmt.Println(ansi.ColorizeJSON(requestLogEvent.EventPayload, false, os.Stdout))
		return
	}

	coloredStatus := ansi.ColorizeStatus(payload.Status)

	url := urlForRequestID(&payload)
	requestLink := ansi.Linkify(payload.RequestID, url, os.Stdout)

	if payload.URL == "" {
		payload.URL = "[View path in dashboard]"
	}

	exampleLayout := "2006-01-02 15:04:05"
	localTime := time.Unix(int64(payload.CreatedAt), 0).Format(exampleLayout)

	color := ansi.Color(os.Stdout)
	outputStr := fmt.Sprintf("%s [%d] %s %s [%s]", color.Faint(localTime), coloredStatus, payload.Method, payload.URL, requestLink)
	fmt.Println(outputStr)

	errorValues := reflect.ValueOf(&payload.Error).Elem()
	errType := errorValues.Type()
	for i := 0; i < errorValues.NumField(); i++ {
		fieldValue := errorValues.Field(i).Interface()
		if fieldValue != "" {
			fmt.Printf("%s: %s\n", errType.Field(i).Name, fieldValue)
		}
	}
}

func jsonifyFilters(logFilters *LogFilters) (string, error) {
	bytes, err := json.Marshal(logFilters)
	if err != nil {
		return "", err
	}

	jsonStr := string(bytes)
	return jsonStr, nil
}

func urlForRequestID(payload *EventPayload) string {
	maybeTest := ""
	if !payload.Livemode {
		maybeTest = "/test"
	}

	return fmt.Sprintf("https://dashboard.stripe.com%s/logs/%s", maybeTest, payload.RequestID)
}
