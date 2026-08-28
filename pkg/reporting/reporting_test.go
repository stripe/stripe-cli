package reporting

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	sentry "github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
)

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

			CaptureException(test.err)

			require.Empty(t, transport.Events())
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

			CaptureException(errorcategory.With(errors.New("actionable error"), category))

			events := transport.Events()
			require.Len(t, events, 1)
			require.Equal(t, string(category), events[0].Tags["error_category"])
		})
	}
}

// The Sentry issue title comes from the type of the outermost exception, which
// is the errorcategory wrapper. Without the substitution in scrubEvent every
// categorized error would be titled "errorcategory.categorizedError".
func TestCaptureExceptionTitlesExceptionsWithTheCategory(t *testing.T) {
	transport, restore := bindTestClient(t)
	defer restore()

	CaptureException(errorcategory.With(errors.New("actionable error"), errorcategory.API))

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

	CaptureException(&os.PathError{Op: "open", Path: "config.toml", Err: errors.New("permission denied")})

	events := transport.Events()
	require.Len(t, events, 1)

	exceptions := events[0].Exception
	require.NotEmpty(t, exceptions)
	require.Equal(t, "*fs.PathError", exceptions[len(exceptions)-1].Type)
}

func TestCaptureExceptionCapturesUnknownErrorsAsInternal(t *testing.T) {
	transport, restore := bindTestClient(t)
	defer restore()

	CaptureException(errors.New("unknown error"))

	events := transport.Events()
	require.Len(t, events, 1)
	require.Equal(t, string(errorcategory.Internal), events[0].Tags["error_category"])
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

	RecoverAndReport("panic value")
	CaptureException(errors.New("ordinary error"))

	events := transport.Events()
	require.Len(t, events, 2)
	require.Equal(t, string(errorcategory.Panic), events[0].Tags["error_category"])
	require.Equal(t, string(errorcategory.Internal), events[1].Tags["error_category"])
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
