package ansi

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/logrusorgru/aurora"
	"github.com/tidwall/pretty"

	"golang.org/x/crypto/ssh/terminal"
)

//
// Public variables
//

// ForceColors forces the use of colors and other ANSI sequences.
var ForceColors = false

// DisableColors disables all colors and other ANSI sequences.
var DisableColors = false

// EnvironmentOverrideColors overs coloring based on `CLICOLOR` and
// `CLICOLOR_FORCE`. Cf. https://bixense.com/clicolors/
var EnvironmentOverrideColors = true

//
// Public functions
//

// Color returns an aurora.Aurora instance with colors enabled or disabled
// depending on whether the writer supports colors.
func Color(w io.Writer) aurora.Aurora {
	return aurora.NewAurora(shouldUseColors(w))
}

// ColorizeJSON returns a colorized version of the input JSON, if the writer
// supports colors.
func ColorizeJSON(json string, w io.Writer) string {
	if !shouldUseColors(w) {
		return json
	}

	return string(pretty.Color([]byte(json), nil))
}

// Linkify returns an ANSI escape sequence with an hyperlink, if the writer
// supports colors.
func Linkify(text, url string, w io.Writer) string {
	if !shouldUseColors(w) {
		return text
	}

	// See https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda
	// for more information about this escape sequence.
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, text)
}

// StartSpinner starts a spinner with the given message. If the writer doesn't
// support colors, it simply prints the message.
func StartSpinner(msg string, w io.Writer) *spinner.Spinner {
	if !shouldUseColors(w) {
		fmt.Fprintln(w, msg)
		return nil
	}

	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Writer = w
	s.Suffix = " " + msg
	s.Start()
	return s
}

// StopSpinner stops a spinner with the given message. If the writer doesn't
// support colors, it simply prints the message.
func StopSpinner(s *spinner.Spinner, msg string, w io.Writer) {
	if !shouldUseColors(w) {
		fmt.Fprintln(w, msg)
		return
	}

	s.FinalMSG = "> " + msg + "\n"
	s.Stop()
}

//
// Private functions
//

func checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}

func shouldUseColors(w io.Writer) bool {
	useColors := ForceColors || checkIfTerminal(w)

	if EnvironmentOverrideColors {
		if force, ok := os.LookupEnv("CLICOLOR_FORCE"); ok && force != "0" {
			useColors = true
		} else if ok && force == "0" {
			useColors = false
		} else if os.Getenv("CLICOLOR") == "0" {
			useColors = false
		}
	}

	return useColors && !DisableColors
}
