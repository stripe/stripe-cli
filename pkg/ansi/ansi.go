package ansi

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/briandowns/spinner"
	"github.com/logrusorgru/aurora"
	"github.com/tidwall/pretty"
	"golang.org/x/term"
)

var darkTerminalStyle = &pretty.Style{
	Key:    [2]string{"\x1B[34m", "\x1B[0m"},
	String: [2]string{"\x1B[30m", "\x1B[0m"},
	Number: [2]string{"\x1B[94m", "\x1B[0m"},
	True:   [2]string{"\x1B[35m", "\x1B[0m"},
	False:  [2]string{"\x1B[35m", "\x1B[0m"},
	Null:   [2]string{"\x1B[31m", "\x1B[0m"},
}

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

// Bold returns bolded text if the writer supports colors
func Bold(text string) string {
	color := Color(os.Stdout)
	return color.Sprintf(color.Bold(text))
}

// Color returns an aurora.Aurora instance with colors enabled or disabled
// depending on whether the writer supports colors.
func Color(w io.Writer) aurora.Aurora {
	return aurora.NewAurora(shouldUseColors(w))
}

// ColorizeJSON returns a colorized version of the input JSON, if the writer
// supports colors.
func ColorizeJSON(json string, darkStyle bool, w io.Writer) string {
	if !shouldUseColors(w) {
		return json
	}

	style := (*pretty.Style)(nil)
	if darkStyle {
		style = darkTerminalStyle
	}

	return string(pretty.Color([]byte(json), style))
}

// ColorizeStatus returns a colorized number for HTTP status code
func ColorizeStatus(status int) aurora.Value {
	color := Color(os.Stdout)

	switch {
	case status >= 500:
		return color.Red(status).Bold()
	case status >= 300:
		return color.Yellow(status).Bold()
	default:
		return color.Green(status).Bold()
	}
}

// Faint returns slightly offset color text if the writer supports it
func Faint(text string) string {
	color := Color(os.Stdout)
	return color.Sprintf(color.Faint(text))
}

// Italic returns italicized text if the writer supports it.
func Italic(text string) string {
	color := Color(os.Stdout)
	return color.Sprintf(color.Italic(text))
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

type charset = []string

func getCharset() charset {
	// See https://github.com/briandowns/spinner#available-character-sets for
	// list of available charsets
	if runtime.GOOS == "windows" {
		// Less fancy, but uses ASCII characters so works with Windows default
		// console.
		return spinner.CharSets[8]
	}
	return spinner.CharSets[11]
}

const duration = time.Duration(100) * time.Millisecond

// StartNewSpinner starts a new spinner with the given message. If the writer is not
// a terminal or doesn't support colors, it simply prints the message.
func StartNewSpinner(msg string, w io.Writer) *spinner.Spinner {
	if !isTerminal(w) || !shouldUseColors(w) {
		fmt.Fprintln(w, msg)
		return nil
	}

	s := spinner.New(getCharset(), duration)
	s.Writer = w

	if msg != "" {
		s.Suffix = " " + msg
	}

	s.Start()

	return s
}

// StartSpinner updates an existing spinner's message, and starts it if it was stopped
func StartSpinner(s *spinner.Spinner, msg string, w io.Writer) {
	if s == nil {
		fmt.Fprintln(w, msg)
		return
	}
	if msg != "" {
		s.Suffix = " " + msg
	}
	if !s.Active() {
		s.Start()
	}
}

// StopSpinner stops a spinner with the given message. If the writer is not
// a terminal or doesn't support colors, it simply prints the message.
func StopSpinner(s *spinner.Spinner, msg string, w io.Writer) {
	if !isTerminal(w) || !shouldUseColors(w) {
		fmt.Fprintln(w, msg)
		return
	}

	if msg != "" {
		s.FinalMSG = "> " + msg + "\n"
	}

	s.Stop()
}

// StrikeThrough returns struck though text if the writer supports colors
func StrikeThrough(text string) string {
	color := Color(os.Stdout)
	return color.Sprintf(color.StrikeThrough(text))
}

//
// Private functions
//

func isTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return term.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}

func shouldUseColors(w io.Writer) bool {
	useColors := ForceColors || isTerminal(w)

	if EnvironmentOverrideColors {
		force, ok := os.LookupEnv("CLICOLOR_FORCE")

		switch {
		case ok && force != "0":
			useColors = true
		case ok && force == "0":
			useColors = false
		case os.Getenv("CLICOLOR") == "0":
			useColors = false
		}
	}

	return useColors && !DisableColors
}
