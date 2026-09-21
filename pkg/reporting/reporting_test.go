package reporting

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	sentry "github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// syncTelemetryClient records SendEvent calls and lets tests block until the
// CaptureException/RecoverAndReport goroutine has delivered its event.
type syncTelemetryClient struct {
	mu     sync.Mutex
	events []struct{ name, value string }
	done   chan struct{}
}

func newSyncTelemetryClient() *syncTelemetryClient {
	return &syncTelemetryClient{done: make(chan struct{}, 8)}
}

func (c *syncTelemetryClient) SendAPIRequestEvent(_ context.Context, _ string, _ bool) (*http.Response, error) {
	return nil, nil
}

func (c *syncTelemetryClient) SendEvent(_ context.Context, eventName string, eventValue string) {
	c.mu.Lock()
	c.events = append(c.events, struct{ name, value string }{eventName, eventValue})
	c.mu.Unlock()
	c.done <- struct{}{}
}

func (c *syncTelemetryClient) waitForEvent(t *testing.T) {
	t.Helper()
	select {
	case <-c.done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for a telemetry event to be sent")
	}
}

func (c *syncTelemetryClient) lastEvent() (name string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.events) == 0 {
		return "", ""
	}
	last := c.events[len(c.events)-1]
	return last.name, last.value
}

func telemetryContext(client *syncTelemetryClient) context.Context {
	ctx := stripe.WithTelemetryClient(context.Background(), client)
	return stripe.WithEventMetadata(ctx, stripe.NewEventMetadata())
}

func TestCaptureExceptionSuppressesExpectedCategories(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "user input", err: context.Canceled},
		{name: "auth", err: requests.RequestError{StatusCode: 401}},
		{name: "rate limit", err: requests.RequestError{StatusCode: 429}},
		{name: "explicit rate limit", err: errorcategory.New(errorcategory.RateLimit, "you have too many `stripe listen` sessions open, please close some and try again")},
		{name: "wrapped user input", err: fmt.Errorf("wrapped: %w", context.Canceled)},
		{name: "wrapped auth", err: fmt.Errorf("wrapped: %w", requests.RequestError{StatusCode: 403})},
		{name: "wrapped rate limit", err: fmt.Errorf("wrapped: %w", requests.RequestError{StatusCode: 429})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport, restore := bindTestClient(t)
			defer restore()
			telemetryClient := newSyncTelemetryClient()

			CaptureException(telemetryContext(telemetryClient), test.err)

			require.Empty(t, transport.Events())
			require.Empty(t, telemetryClient.events, "suppressed categories must not be mirrored to telemetry either")
		})
	}
}

func TestCaptureExceptionCapturesActionableCategories(t *testing.T) {
	categories := []errorcategory.Category{
		errorcategory.Network,
		errorcategory.API,
		errorcategory.Filesystem,
		errorcategory.Internal,
	}

	for _, category := range categories {
		t.Run(string(category), func(t *testing.T) {
			transport, restore := bindTestClient(t)
			defer restore()
			telemetryClient := newSyncTelemetryClient()

			CaptureException(telemetryContext(telemetryClient), errorcategory.With(errors.New("actionable error"), category))

			events := transport.Events()
			require.Len(t, events, 1)
			require.Equal(t, string(category), events[0].Tags["error_category"])

			telemetryClient.waitForEvent(t)
			eventName, eventValue := telemetryClient.lastEvent()
			require.Equal(t, errorTelemetryEventName, eventName)
			require.Equal(t, string(category), eventValue, "telemetry value must be the bare category, never the error message")
		})
	}
}

// The Sentry issue title comes from the type of the outermost exception, which
// is the errorcategory wrapper. Without the substitution in scrubEvent every
// categorized error would be titled "errorcategory.categorizedError".
func TestCaptureExceptionTitlesExceptionsWithTheCategory(t *testing.T) {
	transport, restore := bindTestClient(t)
	defer restore()

	CaptureException(telemetryContext(newSyncTelemetryClient()), errorcategory.With(errors.New("actionable error"), errorcategory.API))

	events := transport.Events()
	require.Len(t, events, 1)

	exceptions := events[0].Exception
	require.NotEmpty(t, exceptions)
	require.Equal(t, string(errorcategory.API), exceptions[len(exceptions)-1].Type)
	require.NotContains(t, exceptions[len(exceptions)-1].Type, "categorizedError")
}

func TestCaptureExceptionLeavesUncategorizedExceptionTypes(t *testing.T) {
	transport, restore := bindTestClient(t)
	defer restore()

	CaptureException(telemetryContext(newSyncTelemetryClient()), &os.PathError{Op: "open", Path: "config.toml", Err: errors.New("permission denied")})

	events := transport.Events()
	require.Len(t, events, 1)

	exceptions := events[0].Exception
	require.NotEmpty(t, exceptions)
	require.Equal(t, "*fs.PathError", exceptions[len(exceptions)-1].Type)
}

func TestCaptureExceptionCapturesUnknownErrorsAsInternal(t *testing.T) {
	transport, restore := bindTestClient(t)
	defer restore()

	CaptureException(telemetryContext(newSyncTelemetryClient()), errors.New("unknown error"))

	events := transport.Events()
	require.Len(t, events, 1)
	require.Equal(t, string(errorcategory.Internal), events[0].Tags["error_category"])
}

func TestCaptureExceptionNeverSendsTheErrorMessageToTelemetry(t *testing.T) {
	_, restore := bindTestClient(t)
	defer restore()
	telemetryClient := newSyncTelemetryClient()

	CaptureException(telemetryContext(telemetryClient), errors.New("failed for account acct_1abcDEF at https://example.com/secret-path?token=xyz"))

	telemetryClient.waitForEvent(t)
	_, eventValue := telemetryClient.lastEvent()
	require.Equal(t, string(errorcategory.Internal), eventValue)
}

func TestShouldCapture(t *testing.T) {
	tests := []struct {
		category errorcategory.Category
		expected bool
	}{
		{category: errorcategory.UserInput, expected: false},
		{category: errorcategory.Auth, expected: false},
		{category: errorcategory.RateLimit, expected: false},
		{category: errorcategory.Network, expected: true},
		{category: errorcategory.API, expected: true},
		{category: errorcategory.Filesystem, expected: true},
		{category: errorcategory.Internal, expected: true},
		{category: errorcategory.Panic, expected: true},
		{category: errorcategory.Category("unknown"), expected: true},
	}

	for _, test := range tests {
		t.Run(string(test.category), func(t *testing.T) {
			require.Equal(t, test.expected, shouldCapture(test.category))
		})
	}
}

func TestRecoverAndReportSetsIsolatedPanicCategory(t *testing.T) {
	transport, restore := bindTestClient(t)
	defer restore()
	telemetryClient := newSyncTelemetryClient()
	ctx := telemetryContext(telemetryClient)

	RecoverAndReport(ctx, "panic value")
	CaptureException(ctx, errors.New("ordinary error"))

	events := transport.Events()
	require.Len(t, events, 2)
	require.Equal(t, string(errorcategory.Panic), events[0].Tags["error_category"])
	require.Equal(t, string(errorcategory.Internal), events[1].Tags["error_category"])

	telemetryClient.waitForEvent(t)
	telemetryClient.waitForEvent(t)
	require.Len(t, telemetryClient.events, 2)
	require.Equal(t, string(errorcategory.Panic), telemetryClient.events[0].value)
	require.Equal(t, string(errorcategory.Internal), telemetryClient.events[1].value)
}

func bindTestClient(t *testing.T) (*sentry.MockTransport, func()) {
	t.Helper()

	transport := &sentry.MockTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:        "https://public@localhost/1",
		Transport:  transport,
		BeforeSend: scrubEvent,
	})
	require.NoError(t, err)

	hub := sentry.CurrentHub()
	previousClient := hub.Client()
	previousAccountIDProvider := accountIDProvider
	previousCommandPath := commandPath
	hub.BindClient(client)
	accountIDProvider = nil
	commandPath = ""

	return transport, func() {
		hub.BindClient(previousClient)
		accountIDProvider = previousAccountIDProvider
		commandPath = previousCommandPath
	}
}
