package requests

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"
)

// Diagnostics opts a request into summaries and retains its correlation ID for
// callers that decode or validate the response after the request completes.
type Diagnostics struct {
	ShowHeaders bool
	Operation   string
	Stage       string
	RequestID   string
}

var safeRequestID = regexp.MustCompile(`^req_[A-Za-z0-9]{1,128}$`)
var safeErrorCode = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)
var safeContext = regexp.MustCompile(`^(acct|play|wksp)_[A-Za-z0-9_]+(/(acct|play|wksp)_[A-Za-z0-9_]+)*$`)

func safeDiagnosticValue(value string, pattern *regexp.Regexp) string {
	if pattern.MatchString(value) {
		return value
	}
	return ""
}

func (d *Diagnostics) configure(fields log.Fields, request *http.Request) {
	if d == nil {
		return
	}
	fields["endpoint"] = request.URL.EscapedPath()
	fields["stripe_context"] = safeDiagnosticValue(request.Header.Get("Stripe-Context"), safeContext)
	switch mode := request.Header.Get("Stripe-Livemode"); mode {
	case "true", "false":
		fields["livemode"] = mode
	}
}

func (d *Diagnostics) response(fields log.Fields, response RequestError, readErr error, started time.Time) {
	outcome := "success"
	if response.StatusCode >= 300 {
		outcome = "http_error"
	} else if readErr != nil {
		outcome = "read_error"
	}
	d.complete(fields, response.StatusCode, response.RequestID, response.ErrorCode, outcome, started)
}

func (d *Diagnostics) complete(fields log.Fields, status int, requestID, code, outcome string, started time.Time) {
	if d == nil {
		return
	}
	d.RequestID = safeDiagnosticValue(requestID, safeRequestID)
	fields["operation"] = d.Operation
	fields["stage"] = d.Stage
	fields["elapsed"] = time.Since(started)
	fields["status"] = status
	fields["request_id"] = d.RequestID
	fields["error_code"] = safeDiagnosticValue(code, safeErrorCode)
	fields["outcome"] = outcome
	log.WithFields(fields).Debug("Sandbox management request")
}

// WithRequestID annotates a safe caller-generated error without changing its category.
func (d *Diagnostics) WithRequestID(err error) error {
	if d == nil {
		return err
	}
	return WithRequestID(err, d.RequestID)
}

type correlatedError struct {
	err       error
	requestID string
}

func (e correlatedError) Error() string {
	return fmt.Sprintf("%s (request ID: %s)", e.err, e.requestID)
}

func (e correlatedError) Unwrap() error { return e.err }

func (e correlatedError) StripeRequestID() string { return e.requestID }

// WithRequestID adds only a validated Stripe request ID to an error.
func WithRequestID(err error, requestID string) error {
	requestID = safeDiagnosticValue(requestID, safeRequestID)
	if err == nil || requestID == "" || RequestIDFromError(err) == requestID {
		return err
	}
	return correlatedError{err: err, requestID: requestID}
}

// RequestIDFromError extracts correlation metadata without rendering the error body.
func RequestIDFromError(err error) string {
	var correlated interface{ StripeRequestID() string }
	if errors.As(err, &correlated) {
		return safeDiagnosticValue(correlated.StripeRequestID(), safeRequestID)
	}
	return ""
}

func firstDiagnostics(options []*Diagnostics) *Diagnostics {
	if len(options) > 0 {
		return options[0]
	}
	return nil
}
