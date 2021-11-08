package stripe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/version"
)

//
// Public types
//

// telemetryMetadataKey is the key for the telemetry context
type telemetryMetadataKey struct{}

// TelemetryClientKey is the key for the telemetry client
type telemetryClientKey struct{}

// DefaultTelemetryEndpoint is the default URL for the telemetry destination
const DefaultTelemetryEndpoint = "https://r.stripe.com/0"

// CLIAnalyticsEventMetadata is the structure that holds telemetry data context that is ultimately sent to the Stripe Analytics Service.
type CLIAnalyticsEventMetadata struct {
	InvocationID      string `url:"invocation_id"`      // The invocation id is unique to each context object and represents all events coming from one command / gRPC method call
	UserAgent         string `url:"user_agent"`         // the application that is used to create this request
	CommandPath       string `url:"command_path"`       // the command or gRPC method that initiated this request
	Merchant          string `url:"merchant"`           // the merchant ID: ex. acct_xxxx
	CLIVersion        string `url:"cli_version"`        // the version of the CLI
	OS                string `url:"os"`                 // the OS of the system
	GeneratedResource bool   `url:"generated_resource"` // whether or not this was a generated resource
}

// TelemetryClient is an interface that can send two types of events: an API request, and just general events.
type TelemetryClient interface {
	SendAPIRequestEvent(ctx context.Context, requestID string, livemode bool) (*http.Response, error)
	SendEvent(ctx context.Context, eventName string, eventValue string)
}

// AnalyticsTelemetryClient sends event information to r.stripe.com
type AnalyticsTelemetryClient struct {
	BaseURL    *url.URL
	wg         sync.WaitGroup
	HTTPClient *http.Client
}

// NoOpTelemetryClient does not call any endpoint and returns an empty response
type NoOpTelemetryClient struct {
}

//
// Public functions
//

// NewEventMetadata initializes an instance of CLIAnalyticsEventContext
func NewEventMetadata() *CLIAnalyticsEventMetadata {
	return &CLIAnalyticsEventMetadata{
		InvocationID: uuid.NewString(),
		CLIVersion:   version.Version,
		OS:           runtime.GOOS,
	}
}

// WithEventMetadata returns a new copy of context.Context with the provided CLIAnalyticsEventMetadata
func WithEventMetadata(ctx context.Context, metadata *CLIAnalyticsEventMetadata) context.Context {
	return context.WithValue(ctx, telemetryMetadataKey{}, metadata)
}

// GetEventMetadata returns the CLIAnalyticsEventMetadata from the provided context
func GetEventMetadata(ctx context.Context) *CLIAnalyticsEventMetadata {
	metadata := ctx.Value(telemetryMetadataKey{})
	if metadata != nil {
		return metadata.(*CLIAnalyticsEventMetadata)
	}
	return nil
}

// WithTelemetryClient returns a new copy of context.Context with the provided telemetryClient
func WithTelemetryClient(ctx context.Context, client TelemetryClient) context.Context {
	return context.WithValue(ctx, telemetryClientKey{}, client)
}

// GetTelemetryClient returns the CLIAnalyticsEventMetadata from the provided context
func GetTelemetryClient(ctx context.Context) TelemetryClient {
	client := ctx.Value(telemetryClientKey{})
	if client != nil {
		return client.(TelemetryClient)
	}
	return nil
}

// SetCobraCommandContext sets the telemetry values for the command being executed.
func (e *CLIAnalyticsEventMetadata) SetCobraCommandContext(cmd *cobra.Command) {
	e.CommandPath = cmd.CommandPath()
	e.GeneratedResource = false

	if cmd.HasParent() {
		for key, value := range cmd.Parent().Annotations {
			// Generated commands have an annotation called "operation", we can
			// search for that to let us know it's generated
			if key == cmd.Use && value == "operation" {
				e.GeneratedResource = true
			}
		}
	}
}

// SetMerchant sets the merchant on the CLIAnalyticsEventContext object
func (e *CLIAnalyticsEventMetadata) SetMerchant(merchant string) {
	e.Merchant = merchant
}

// SetUserAgent sets the userAgent on the CLIAnalyticsEventContext object
func (e *CLIAnalyticsEventMetadata) SetUserAgent(userAgent string) {
	e.UserAgent = userAgent
}

// SetCommandPath sets the commandPath on the CLIAnalyticsEventContext object
func (e *CLIAnalyticsEventMetadata) SetCommandPath(commandPath string) {
	e.CommandPath = commandPath
}

// SendAPIRequestEvent is a special function for API requests
func (a *AnalyticsTelemetryClient) SendAPIRequestEvent(ctx context.Context, requestID string, livemode bool) (*http.Response, error) {
	a.wg.Add(1)
	defer a.wg.Done()
	telemetryMetadata := GetEventMetadata(ctx)
	if telemetryMetadata != nil {
		data, _ := query.Values(telemetryMetadata)

		data.Set("client_id", "stripe-cli")
		data.Set("request_id", requestID)
		data.Set("livemode", strconv.FormatBool(livemode))
		data.Set("event_id", uuid.NewString())
		data.Set("event_name", "API Request")
		data.Set("event_value", "")
		data.Set("created", fmt.Sprint((time.Now().Unix())))

		return a.sendData(ctx, data)
	}
	return nil, nil
}

// SendEvent sends a telemetry event to r.stripe.com
func (a *AnalyticsTelemetryClient) SendEvent(ctx context.Context, eventName string, eventValue string) {
	a.wg.Add(1)
	defer a.wg.Done()
	telemetryMetadata := GetEventMetadata(ctx)
	if telemetryMetadata != nil {
		data, _ := query.Values(telemetryMetadata)

		data.Set("client_id", "stripe-cli")
		data.Set("event_id", uuid.NewString())
		data.Set("event_name", eventName)
		data.Set("event_value", eventValue)
		data.Set("created", fmt.Sprint((time.Now().Unix())))

		resp, err := a.sendData(ctx, data)
		// Don't throw exception if we fail to send the event
		if err != nil {
			log.Debugf("Error while sending telemetry data: %v\n", err)
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
}

func (a *AnalyticsTelemetryClient) sendData(ctx context.Context, data url.Values) (*http.Response, error) {
	a.wg.Add(1)
	defer a.wg.Done()
	if a.BaseURL == nil {
		analyticsURL, err := url.Parse(DefaultTelemetryEndpoint)
		if err != nil {
			return nil, err
		}
		a.BaseURL = analyticsURL
	}

	req, err := http.NewRequest(http.MethodPost, a.BaseURL.String(), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("origin", "stripe-cli")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Wait will return when all in-flight telemetry requests are complete.
func (a *AnalyticsTelemetryClient) Wait() {
	a.wg.Wait()
}

// SendAPIRequestEvent does nothing
func (a *NoOpTelemetryClient) SendAPIRequestEvent(ctx context.Context, requestID string, livemode bool) (*http.Response, error) {
	return nil, nil
}

// SendEvent does nothing
func (a *NoOpTelemetryClient) SendEvent(ctx context.Context, eventName string, eventValue string) {
}

// TelemetryOptedOut returns true if the user has opted out of telemetry,
// false otherwise.
func TelemetryOptedOut(optoutVar string) bool {
	optoutVar = strings.ToLower(optoutVar)

	return optoutVar == "1" || optoutVar == "true"
}
