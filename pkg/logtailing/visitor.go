package logtailing

/**
 * This file contains types for processing streamed logs outside of this package. This is useful for
 * packages that want to define their own handler, such as
 * - a Cobra command, which wants to pretty-print logs to the terminal
 * - an RPC service, which wants to stream logs to a client
 */

// StreamElementVisitor should implement the handlers for each type of element
type StreamElementVisitor struct {
	VisitError   func(ErrorElement) error
	VisitLog     func(LogElement) error
	VisitStatus  func(StatusElement) error
	VisitWarning func(WarningElement) error
}

// ErrorElement is an error from the log tailer
type ErrorElement struct {
	Error error
}

// LogElement is the log received on the stream
type LogElement struct {
	Log EventPayload

	MarshalledLog string
}

// StatusElement is the current status of the stream: loading, ready, etc.
type StatusElement struct {
	Status status
}

// WarningElement is a worning from the log tailer
type WarningElement struct {
	Warning string
}

// StreamElement is an element that can be visited. This is visitor pattern boilerplate.
type StreamElement interface {
	Visit(v StreamElementVisitor) error
}

// Visit is visitor pattern boilerplate
func (ee ErrorElement) Visit(v StreamElementVisitor) error {
	if v.VisitError == nil {
		return nil
	}
	return v.VisitError(ee)
}

// Visit is visitor pattern boilerplate
func (le LogElement) Visit(v StreamElementVisitor) error {
	if v.VisitLog == nil {
		return nil
	}
	return v.VisitLog(le)
}

// Visit is visitor pattern boilerplate
func (we WarningElement) Visit(v StreamElementVisitor) error {
	if v.VisitWarning == nil {
		return nil
	}
	return v.VisitWarning(we)
}

// Visit is visitor pattern boilerplate
func (se StatusElement) Visit(v StreamElementVisitor) error {
	if v.VisitStatus == nil {
		return nil
	}
	return v.VisitStatus(se)
}

type status int

const (
	// Loading means the stream is being set up
	Loading status = iota

	// Reconnecting means the stream is reconnecting
	Reconnecting

	// Ready means we are ready to receive logs
	Ready

	// Done means log streaming is done
	Done
)
