// Package reporting provides error reporting via Sentry.
package reporting

import (
	"time"

	sentry "github.com/getsentry/sentry-go"
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
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		if accountIDProvider != nil {
			if accountID, _ := accountIDProvider(); accountID != "" {
				scope.SetTag("account_id", accountID)
			}
		}
		if commandPath != "" {
			scope.SetTag("command", commandPath)
		}
	})
	sentry.CaptureException(err)
}

// RecoverAndReport captures a recovered panic value to the error reporting backend.
// The caller is responsible for re-panicking and calling Flush before the process exits.
func RecoverAndReport(r any) {
	sentry.CurrentHub().Recover(r)
}

// Flush blocks until all buffered events are delivered or the timeout elapses.
func Flush() {
	sentry.Flush(2 * time.Second)
}
