// Package reporting provides error reporting via Sentry and telemetry.
package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	sentry "github.com/getsentry/sentry-go"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// errorTelemetryEventName is the telemetry event name used to mirror errors
// reported to Sentry, so error rates can be tracked without Sentry access.
const errorTelemetryEventName = "CLI Error"

// errorTelemetryPayload mirrors the data attached to the corresponding Sentry
// event: the classification tag and the fingerprint (call site, root error
// type, and root error message).
type errorTelemetryPayload struct {
	Category string `json:"category"`
	Type     string `json:"type"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

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

	sendErrorTelemetry(ctx, category, root, caller)
}

// sendErrorTelemetry mirrors a captured Sentry event to telemetry: the same
// category tag, and the same root error type/message/call site used for the
// Sentry fingerprint. account_id and command are omitted here since they're
// already attached to every telemetry event via CLIAnalyticsEventMetadata.
func sendErrorTelemetry(ctx context.Context, category errorcategory.Category, root error, caller string) {
	telemetryClient := stripe.GetTelemetryClient(ctx)
	if telemetryClient == nil {
		return
	}
	if stripe.GetEventMetadata(ctx) == nil {
		// CaptureException always runs with metadata already on ctx (set once
		// in cmd.Execute). RecoverAndReport can run before that, e.g. a panic
		// during setup, so fall back to freshly built metadata.
		ctx = stripe.WithEventMetadata(ctx, stripe.NewEventMetadata())
	}

	payload := errorTelemetryPayload{
		Category: string(category),
		Type:     fmt.Sprintf("%T", root),
		Message:  redactSensitiveStrings(root.Error()),
		Location: caller,
	}
	value, err := json.Marshal(payload)
	if err != nil {
		return
	}

	go telemetryClient.SendEvent(ctx, errorTelemetryEventName, string(value))
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

	sendErrorTelemetry(ctx, errorcategory.Panic, fmt.Errorf("%v", r), "")
}

// Flush blocks until all buffered events are delivered or the timeout elapses.
func Flush() {
	sentry.Flush(2 * time.Second)
}
