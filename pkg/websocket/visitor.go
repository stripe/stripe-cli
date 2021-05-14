package websocket

/**
 * This file contains types for processing streamed logs outside of this package. This is useful for
 * packages that want to define their own handler, such as
 * - a Cobra command, which wants to pretty-print logs to the terminal
 * - an RPC service, which wants to stream logs to a client
 */

// Visitor should implement the handlers for each type of element
type Visitor struct {
	VisitError   func(ErrorElement) error
	VisitData    func(DataElement) error
	VisitStatus  func(StateElement) error
	VisitWarning func(WarningElement) error
}

// ErrorElement is an error from the log tailer
type ErrorElement struct {
	Error error
}

// DataElement is the data received on the stream.
// It represents the main data model between communicated.
type DataElement struct {
	Data      interface{}
	Marshaled string
}

// StateElement is the current state of the stream: loading, ready, etc.
type StateElement struct {
	State state
	Data  []string
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
func (le DataElement) Accept(v *Visitor) error {
	if v.VisitData == nil {
		return nil
	}
	return v.VisitData(le)
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
