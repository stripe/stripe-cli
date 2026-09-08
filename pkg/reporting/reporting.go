// Package reporting provides error reporting via Sentry.
package reporting

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	sentry "github.com/getsentry/sentry-go"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

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

// CaptureException reports err to the error reporting backend.
func CaptureException(err error) {
	category := classifyError(err)
	if !shouldCapture(category) {
		return
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
		scope.SetFingerprint([]string{caller, fmt.Sprintf("%T", root), root.Error()})
		sentry.CaptureException(err)
	})
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
func RecoverAndReport(r any) {
	sentry.CurrentHub().WithScope(func(scope *sentry.Scope) {
		scope.SetTag("error_category", string(errorcategory.Panic))
		sentry.CurrentHub().Recover(r)
	})
}

// Flush blocks until all buffered events are delivered or the timeout elapses.
func Flush() {
	sentry.Flush(2 * time.Second)
}
