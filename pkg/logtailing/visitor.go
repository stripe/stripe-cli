package logtailing

/**
 * This file contains types for processing streamed logs outside of this package. This is useful for
 * packages that want to define their own handler, such as
 * - a Cobra command, which wants to pretty-print logs to the terminal
 * - an RPC service, which wants to stream logs to a client
 */

// Visitor should implement the handlers for each type of element
type Visitor struct {
	VisitError   func(ErrorElement) error
	VisitLog     func(LogElement) error
	VisitStatus  func(StateElement) error
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

// StateElement is the current state of the stream: loading, ready, etc.
type StateElement struct {
	State state
}

// WarningElement is a warning from the log tailer
type WarningElement struct {
	Warning string
}

// IElement is an element that can be visited. This is visitor pattern boilerplate.
type IElement interface {
	Accept(v *Visitor) error
}

// Accept is visitor pattern boilerplate
func (ee ErrorElement) Accept(v *Visitor) error {
	// This null check prevents segfaults. There isn't a good way to enforce the visitor method
	// exists at compile time.
	if v.VisitError == nil {
		return nil
	}
	return v.VisitError(ee)
}

// Accept is visitor pattern boilerplate
func (le LogElement) Accept(v *Visitor) error {
	if v.VisitLog == nil {
		return nil
	}
	return v.VisitLog(le)
}

// Accept is visitor pattern boilerplate
func (we WarningElement) Accept(v *Visitor) error {
	if v.VisitWarning == nil {
		return nil
	}
	return v.VisitWarning(we)
}

// Accept is visitor pattern boilerplate
func (se StateElement) Accept(v *Visitor) error {
	if v.VisitStatus == nil {
		return nil
	}
	return v.VisitStatus(se)
}

type state int

const (
	// Loading means the stream is being set up
	Loading state = iota

	// Reconnecting means the stream is reconnecting
	Reconnecting

	// Ready means we are ready to receive logs
	Ready

	// Done means log streaming is done
	Done
)
