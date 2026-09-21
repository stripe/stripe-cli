package login

import (
	"io"

	"github.com/briandowns/spinner"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// startSpinnerAfterSignal starts a spinner once opened is closed, so it only
// appears once the user has actually opened the browser rather than for the
// full duration of the poll. If opened is nil (there's no browser-open signal
// to wait for), the spinner starts immediately. If the work finishes before
// opened is ever closed, the spinner never appears. It returns a func that
// stops the spinner (a no-op if it never started).
func startSpinnerAfterSignal(msg string, w io.Writer, opened <-chan struct{}) func() {
	if opened == nil {
		s := ansi.StartNewSpinner(msg, w)
		return func() { ansi.StopSpinner(s, "", w) }
	}

	done := make(chan struct{})
	spinnerCh := make(chan *spinner.Spinner, 1)
	go func() {
		select {
		case <-opened:
			spinnerCh <- ansi.StartNewSpinner(msg, w)
		case <-done:
			spinnerCh <- nil
		}
	}()

	return func() {
		close(done)
		if s := <-spinnerCh; s != nil {
			ansi.StopSpinner(s, "", w)
		}
	}
}
