package login

import (
	"context"
	"fmt"
	"os"

	"github.com/briandowns/spinner"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/login/keys"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

var openBrowser = open.Browser
var canOpenBrowser = open.CanOpenBrowser

const stripeCLIAuthPath = "/stripecli/auth"

// TODO
/*
4. Observability and associated alerting? Business metrics (how many users use this flow)?
5. Rate limiting for each operation?
6. Audit trail for key generation
7. Move configuration changes to profile package
*/

// Authenticator handles the login flow
type Authenticator struct {
	keytransfer      keys.KeyTransfer
	asyncInputReader AsyncInputReader
}

// NewAuthenticator creates a new authenticator object
func NewAuthenticator(keytransfer keys.KeyTransfer) *Authenticator {
	return &Authenticator{
		keytransfer:      keytransfer,
		asyncInputReader: AsyncStdinReader{},
	}
}

// Login function is used to obtain credentials via stripe dashboard.
func (a *Authenticator) Login(ctx context.Context, links *Links) error {
	color := ansi.Color(os.Stdout)
	fmt.Printf("Your pairing code is: %s\n", color.Bold(links.VerificationCode))
	fmt.Println(ansi.Faint("This pairing code verifies your authentication with Stripe."))

	var s *spinner.Spinner

	pollResultCh := make(chan keys.AsyncPollResult)
	inputCh := make(chan int)

	if isSSH() || !canOpenBrowser() {
		fmt.Printf("To authenticate with Stripe, please go to: %s\n", links.BrowserURL)
		s = ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)
		go a.keytransfer.AsyncPollKey(ctx, links.PollURL, 0, 0, pollResultCh)
	} else {
		fmt.Printf("Press Enter to open the browser or visit %s (^C to quit)", links.BrowserURL)
		go a.asyncInputReader.scanln(inputCh)
		go a.keytransfer.AsyncPollKey(ctx, links.PollURL, 0, 0, pollResultCh)
	}

	for {
		select {
		case <-inputCh:
			s = ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)

			err := openBrowser(links.BrowserURL)
			if err != nil {
				msg := fmt.Sprintf("Failed to open browser, please go to %s manually.", links.BrowserURL)
				ansi.StopSpinner(s, msg, os.Stdout)
				s = ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)
			}
		case res := <-pollResultCh:
			if res.Err != nil {
				return res.Err
			}

			message, err := SuccessMessage(ctx, res.Account, stripe.DefaultAPIBaseURL, res.TestModeAPIKey)
			if err != nil {
				fmt.Printf("> Error verifying the CLI was set up successfully: %s\n", err)
				return err
			}

			if s == nil {
				fmt.Printf("\n> %s\n", message)
			} else {
				ansi.StopSpinner(s, message, os.Stdout)
			}
			fmt.Println(ansi.Italic("Please note: this key will expire after 90 days, at which point you'll need to re-authenticate."))
			return nil
		}
	}
}

func isSSH() bool {
	if os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "" {
		return true
	}

	return false
}
