// Package pluginlogin provides login recovery after failed plugin discovery.
package pluginlogin

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/useragent"
)

// CanPrompt excludes headless callers and agents, including agents with a TTY.
func CanPrompt(getEnv func(string) string, stdinIsTerminal bool) bool {
	return stdinIsTerminal && useragent.DetectAIAgent(getEnv) == ""
}

type loginRequiredError struct {
	message string
	cause   error
}

func (e loginRequiredError) Error() string { return e.message }

func (e loginRequiredError) Unwrap() error { return e.cause }

// Offer only starts login after an interactive human explicitly accepts it.
// The caller controls whether to retry discovery after a successful login.
func Offer(name string, cause error, stdin io.Reader, stdout io.Writer, interactive bool, login func() error) error {
	message := fmt.Sprintf("You're not logged in.\nRun 'stripe login', check the account with 'stripe whoami', then retry 'stripe plugin install %s'.", name)
	log.WithField("prefix", "pluginlogin.Offer").Debugf("plugin discovery requires login: %v", cause)
	if !interactive {
		return errorcategory.With(loginRequiredError{message: message, cause: cause}, errorcategory.Auth)
	}

	fmt.Fprintf(stdout, "%s\n\n", message)
	fmt.Fprint(stdout, "Press Enter to run 'stripe login', or type anything to cancel: ")
	input, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil || strings.TrimSpace(input) != "" {
		return errorcategory.New(errorcategory.UserInput, "login canceled")
	}
	if err := login(); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Run 'stripe whoami' to verify the active account before retrying your command.")
	return nil
}
