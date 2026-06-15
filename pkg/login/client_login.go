// Package login implements Stripe authentication flows.
package login

import (
	"context"
	"fmt"
	"os"

	"github.com/briandowns/spinner"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/login/keys"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

var openBrowser = open.Browser
var canOpenBrowser = open.CanOpenBrowser

// SetOpenBrowserForTesting overrides the browser-opening function used by
// the login flow. It returns a restore function that resets to the default.
func SetOpenBrowserForTesting(fn func(string) error) (restore func()) {
	orig := openBrowser
	openBrowser = fn
	return func() { openBrowser = orig }
}

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
	fmt.Print(i18n.Tf("login_flow.output.pairing_code", i18n.Args{"code": fmt.Sprint(color.Bold(links.VerificationCode))}))
	fmt.Println(ansi.Faint(i18n.T("login_flow.output.pairing_code_note")))

	var s *spinner.Spinner

	pollResultCh := make(chan keys.AsyncPollResult)
	inputCh := make(chan int)

	if isSSH() || !canOpenBrowser() {
		fmt.Print(i18n.Tf("login_flow.output.go_to_url", i18n.Args{"url": links.BrowserURL}))
		s = ansi.StartNewSpinner(i18n.T("login_flow.output.waiting_for_confirmation"), os.Stdout)
		go a.keytransfer.AsyncPollKey(ctx, links.PollURL, 0, 0, pollResultCh)
	} else {
		fmt.Print(i18n.Tf("login_flow.output.press_enter_to_open", i18n.Args{"url": links.BrowserURL}))
		go a.asyncInputReader.scanln(inputCh)
		go a.keytransfer.AsyncPollKey(ctx, links.PollURL, 0, 0, pollResultCh)
	}

	for {
		select {
		case <-inputCh:
			s = ansi.StartNewSpinner(i18n.T("login_flow.output.waiting_for_confirmation"), os.Stdout)

			err := openBrowser(links.BrowserURL)
			if err != nil {
				msg := i18n.Tf("login_flow.output.browser_failed", i18n.Args{"url": links.BrowserURL})
				ansi.StopSpinner(s, msg, os.Stdout)
				s = ansi.StartNewSpinner(i18n.T("login_flow.output.waiting_for_confirmation"), os.Stdout)
			}
		case res := <-pollResultCh:
			if res.Err != nil {
				return res.Err
			}

			if res.IsAboutToSaveCreds {
				if s != nil {
					s.Stop()
				}
				continue
			}

			message, err := SuccessMessage(ctx, res.Account, stripe.DefaultAPIBaseURL, res.TestModeAPIKey)
			if err != nil {
				fmt.Print(i18n.Tf("login_flow.output.error_verifying", i18n.Args{"error": err.Error()}))
				return err
			}

			fmt.Print(i18n.Tf("login_flow.output.success", i18n.Args{"message": message}))
			fmt.Println(ansi.Italic(i18n.T("login_flow.output.key_expiry_notice")))
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
