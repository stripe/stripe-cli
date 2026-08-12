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
)

func TestCaptureExceptionSetsErrorCategory(t *testing.T) {
	transport, restore := bindTestClient(t)
	defer restore()

	validationError := errorcategory.New(errorcategory.UserInput, "a profile name is required")
	filesystemError := fmt.Errorf("loading configuration: %w", &os.PathError{Op: "open", Path: "config.toml", Err: errors.New("failed")})

	CaptureException(validationError)
	CaptureException(filesystemError)
	CaptureException(context.Canceled)

	events := transport.Events()
	require.Len(t, events, 3)
	require.Equal(t, string(errorcategory.UserInput), events[0].Tags["error_category"])
	require.Equal(t, string(errorcategory.Filesystem), events[1].Tags["error_category"])
	require.Equal(t, string(errorcategory.UserInput), events[2].Tags["error_category"])
	require.Equal(t, validationError.Error(), events[0].Exception[0].Value)
	require.Equal(t, filesystemError.Error(), events[1].Exception[len(events[1].Exception)-1].Value)
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
