// Package pluginlogin provides login recovery after failed plugin discovery.
package pluginlogin

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/useragent"
)

// CanPrompt excludes headless callers and agents, including agents with a TTY.
func CanPrompt(getEnv func(string) string, stdinIsTerminal bool) bool {
	return stdinIsTerminal && useragent.DetectAIAgent(getEnv) == ""
}

// Offer only starts login after an interactive human explicitly accepts it.
// The caller controls whether to retry discovery after a successful login.
func Offer(name string, cause error, stdin io.Reader, stdout io.Writer, interactive bool, login func() error) error {
	guidance := fmt.Sprintf("Run 'stripe login', then 'stripe whoami' to verify the active account, then retry 'stripe plugin install %s'.", name)
	if !interactive {
		return errorcategory.With(fmt.Errorf("%w. %s", cause, guidance), errorcategory.Auth)
	}

	fmt.Fprintf(stdout, "%v. %s\n\n", cause, guidance)
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
