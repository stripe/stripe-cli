package p400

import (
	"context"
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
func AttemptRegisterReader(ctx context.Context, tsCtx TerminalSessionContext, tries int) (Reader, error) {
	regcode, err := ReaderRegistrationCodePrompt()
	var newReader Reader

	if err != nil {
		return newReader, err
	}

	spinner := ansi.StartNewSpinner("Registering your reader with Stripe...", os.Stdout)

	newReader, err = RegisterReader(ctx, regcode, tsCtx)

	if err != nil {
		tries++
		ansi.StopSpinner(spinner, "", os.Stdout)
		fmt.Println("Could not register the Reader - please try your code again.")

		if tries < 3 {
			return AttemptRegisterReader(ctx, tsCtx, tries)
		}

		return newReader, ErrRegisterReaderFailed
	}

	// we need to get the reader list again in order to source the base_url attr which you don't get back after registering it
	readerList, err := DiscoverReaders(ctx, tsCtx)

	if err != nil {
		return newReader, err
	}

	for _, reader := range readerList {
		if reader.Label == regcode {
			newReader = reader
			break
		}
	}

	ansi.StopSpinner(spinner, "", os.Stdout)

	return newReader, nil
}

// RegisterAndActivateReader prompts the user to either add a new reader or choose an existing reader on their account
// it then calls the appropriate method to set them up and activate a reader session for them to take a test payment
// it returns an updated TerminalSessionContext containing the session's connection token and rpc session token
func RegisterAndActivateReader(ctx context.Context, tsCtx TerminalSessionContext) (TerminalSessionContext, error) {
	var (
		reader         Reader
		activationType string
	)

	spinner := ansi.StartNewSpinner("Requesting connection token...", os.Stdout)
	pstToken, err := GetNewConnectionToken(ctx, tsCtx)

	if err != nil {
		return tsCtx, err
	}

	tsCtx.PstToken = pstToken
	ansi.StopSpinner(spinner, ansi.Faint("Received new connection token"), os.Stdout)

	// check if user has a reader already registered that they might want to use
	readerList, err := DiscoverReaders(ctx, tsCtx)

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
		reader, err = RegisteredReaderChoicePrompt(readerList, tsCtx)

		if err != nil {
			return tsCtx, err
		}
	} else {
		reader, err = AttemptRegisterReader(ctx, tsCtx, 0)

		if err != nil {
			return tsCtx, err
		}

		fmt.Printf("> %s", ansi.Faint("Finished registering reader\n"))
	}

	tsCtx.IPAddress = reader.IPAddress
	tsCtx.BaseURL = reader.BaseURL

	spinner = ansi.StartNewSpinner("Connecting to Reader...", os.Stdout)
	tsCtx.TransactionContext = SetTransactionContext(tsCtx)
	tsCtx.SessionToken, err = ActivateTerminalRPCSession(tsCtx)

	if err != nil {
		return tsCtx, err
	}

	ansi.StopSpinner(spinner, ansi.Faint("Reader connected"), os.Stdout)

	return tsCtx, nil
}
