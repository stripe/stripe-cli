// Package reporting provides error reporting via Sentry and telemetry.
package reporting

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	sentry "github.com/getsentry/sentry-go"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// errorTelemetryEventName is the telemetry event used to mirror errors
// reported to Sentry, so error rates can be tracked in Prometheus without
// Sentry access. The event value is just the category (e.g. "api",
// "internal") — never the error message, which is unbounded free text and
// would blow up tag cardinality downstream.
const errorTelemetryEventName = "CLI Error"

var accountIDProvider func() (string, error)

// SetAccountIDProvider registers a function used to look up the current account
// ID, which is attached as a tag on every captured exception.
func SetAccountIDProvider(fn func() (string, error)) {
	accountIDProvider = fn
}

var commandPath string

// SetCommandPath records the cobra command path (e.g. "stripe webhooks create")
// to be attached as a tag on every captured exception. Only the command name is
// recorded — never args or flag values, which may contain sensitive data.
func SetCommandPath(path string) {
	commandPath = path
}

// Init initializes the error reporter with the given DSN and release version.
func Init(dsn, release string) error {
	return sentry.Init(sentry.ClientOptions{
		Dsn:                    dsn,
		Release:                release,
		BeforeSend:             scrubEvent,
		DisableTelemetryBuffer: true, // workaround: race in v0.48.0 telemetry scheduler can drop events on flush
	})
}

// CaptureException reports err to the error reporting backends (Sentry and
// telemetry).
func CaptureException(ctx context.Context, err error) {
	category := classifyError(err)
	if !shouldCapture(category) {
		return
	}

	// Walk to the root cause so wrapped context ("failed to create customer:
	// EOF") doesn't prevent grouping on the underlying error.
	root := err
	for e := errors.Unwrap(root); e != nil; e = errors.Unwrap(e) {
		root = e
	}

	// Include the call site so that identical generic errors (e.g.
	// *errors.errorString "EOF") from different code paths land in separate
	// Sentry issues without requiring callers to use custom error types.
	caller := "unknown"
	if _, file, line, ok := runtime.Caller(1); ok {
		caller = fmt.Sprintf("%s:%d", file, line)
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("error_category", string(category))
		if accountIDProvider != nil {
			if accountID, _ := accountIDProvider(); accountID != "" {
				scope.SetTag("account_id", accountID)
			}
		}
		if commandPath != "" {
			scope.SetTag("command", commandPath)
		}
		scope.SetFingerprint([]string{caller, fmt.Sprintf("%T", root), root.Error()})
		sentry.CaptureException(err)
	})

	sendErrorTelemetry(ctx, category)
}

// sendErrorTelemetry mirrors a captured Sentry event's category to
// telemetry. The value is the bare category string (e.g. "api") so that AEL
// can key a Prometheus tag directly off it with a fixed values allowlist —
// nothing free-form (error message, call site) is sent, since that would be
// unbounded cardinality if ever wired into a tag/gauge/set.
func sendErrorTelemetry(ctx context.Context, category errorcategory.Category) {
	telemetryClient := stripe.GetTelemetryClient(ctx)
	if telemetryClient == nil {
		return
	}
	metadata := stripe.GetEventMetadata(ctx)
	if metadata == nil {
		// CaptureException always runs with metadata already on ctx (set once
		// in cmd.Execute). RecoverAndReport can run before that, e.g. a panic
		// during setup, so fall back to freshly built metadata.
		metadata = stripe.NewEventMetadata()
	}
	bucketedMetadata := *metadata
	bucketedMetadata.CommandPath = commandBucket(metadata)
	ctx = stripe.WithEventMetadata(ctx, &bucketedMetadata)

	// Sent synchronously (not fire-and-forget): callers on the error/panic
	// path exit via os.Exit right after this, which would otherwise race
	// the request and drop it nondeterministically.
	telemetryClient.SendEvent(ctx, errorTelemetryEventName, string(category))
}

// shouldCapture defines the reporting policy for classified errors. Auth covers
// expected credential or authorization outcomes, not defects in authentication code,
// and RateLimit covers quotas the caller can resolve on their own (closing sessions,
// retrying later); callers can explicitly categorize actionable failures as
// internal, network, or API.
func shouldCapture(category errorcategory.Category) bool {
	switch category {
	case errorcategory.UserInput, errorcategory.Auth, errorcategory.RateLimit:
		return false
	default:
		return true
	}
}

// RecoverAndReport captures a recovered panic value to the error reporting backend.
// The caller is responsible for re-panicking and calling Flush before the process exits.
func RecoverAndReport(ctx context.Context, r any) {
	sentry.CurrentHub().WithScope(func(scope *sentry.Scope) {
		scope.SetTag("error_category", string(errorcategory.Panic))
		sentry.CurrentHub().Recover(r)
	})

	sendErrorTelemetry(ctx, errorcategory.Panic)
}

// Flush blocks until all buffered events are delivered or the timeout elapses.
func Flush() {
	sentry.Flush(2 * time.Second)
}
