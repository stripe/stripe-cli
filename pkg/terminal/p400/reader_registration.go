package p400

import (
	"fmt"
	"os"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// ActivationTypeLabels defines the string values offered as options to the user to prompt them to choose a new or registered reader
// We use these strings in a few places so defining them here cuts down on brittleness if they change
var ActivationTypeLabels = []string{
	"New",
	"Already registered",
}

// this is a lookup for ActivationTypeLabels
const (
	NewReaderChoice = iota
	RegisteredReaderChoice
)

// AttemptRegisterReader prompt the user for their p400 registration code, and tries to register the reader via Stripe API
// it tries three times before returning an error
// returns the registered reader's IP address if successful
func AttemptRegisterReader(tsCtx TerminalSessionContext, tries int) (string, error) {
	regcode, err := ReaderRegistrationCodePrompt()

	if err != nil {
		return "", err
	}

	spinner := ansi.StartNewSpinner("Registering your reader with Stripe...", os.Stdout)

	IPAddress, err := RegisterReader(regcode, tsCtx)

	ansi.StopSpinner(spinner, "", os.Stdout)

	if err != nil {
		tries++
		fmt.Println("Could not register the Reader - please try your code again.")

		if tries < 3 {
			return AttemptRegisterReader(tsCtx, tries)
		}

		return "", ErrRegisterReaderFailed
	}

	return IPAddress, nil
}

// RegisterAndActivateReader prompts the user to either add a new reader or choose an existing reader on their account
// it then calls the appropriate method to set them up and activate a reader session for them to take a test payment
// it returns an updated TerminalSessionContext containing the session's connection token and rpc session token
func RegisterAndActivateReader(tsCtx TerminalSessionContext) (TerminalSessionContext, error) {
	var (
		IPAddress      string
		activationType string
	)

	// check if user has a reader already registered that they might want to use
	readerList, err := DiscoverReaders(tsCtx)

	if err != nil {
		return tsCtx, err
	}

	// if user had at least one reader registered
	if len(readerList) > 0 {
		activationType, err = ReaderNewOrExistingPrompt()

		if err != nil {
			return tsCtx, err
		}
	}

	if activationType == ActivationTypeLabels[RegisteredReaderChoice] {
		IPAddress, err = RegisteredReaderChoicePrompt(readerList, tsCtx)

		if err != nil {
			return tsCtx, err
		}
	} else {
		IPAddress, err = AttemptRegisterReader(tsCtx, 0)

		if err != nil {
			return tsCtx, err
		}

		fmt.Printf("> %s", ansi.Faint("Finished registering reader\n"))
	}

	tsCtx.IPAddress = IPAddress

	spinner := ansi.StartNewSpinner("Requesting connection token...", os.Stdout)
	tsCtx.PstToken, err = GetNewConnectionToken(tsCtx)

	if err != nil {
		return tsCtx, err
	}

	ansi.StopSpinner(spinner, ansi.Faint("Received new connection token"), os.Stdout)

	spinner = ansi.StartNewSpinner("Connecting to Reader...", os.Stdout)
	tsCtx.TransactionContext = SetTransactionContext(tsCtx)
	tsCtx.SessionToken, err = ActivateTerminalRPCSession(tsCtx)

	if err != nil {
		return tsCtx, err
	}

	ansi.StopSpinner(spinner, ansi.Faint("Reader connected"), os.Stdout)

	return tsCtx, nil
}
