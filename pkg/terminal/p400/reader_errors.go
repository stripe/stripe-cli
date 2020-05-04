package p400

import (
	"errors"
)

var (
	// ErrActivateReaderFailed is for when a new RPC session could not be created for the reader
	ErrActivateReaderFailed = errors.New("couldn't communicate with the Reader. Please make sure your reader is online and on the same network as your device.\nSee our troubleshooting docs here: https://stripe.com/docs/terminal/readers/verifone-p400#troubleshooting")
	// ErrRegisterReaderFailed is for when adding the reader via the Stripe API could not be completed (likely bad reg code)
	ErrRegisterReaderFailed = errors.New("could not register the Reader due to an invalid reader code")
	// ErrReaderSelectionFailed is for when the user quit the CLI at the reader choice prompt
	ErrReaderSelectionFailed = errors.New("reader choice failed, no selection made")
	// ErrDiscoverReadersFailed is for when the call to Stripe failed for listing the readers registered to user's account
	ErrDiscoverReadersFailed = errors.New("reader discovery was unable to list readers")
	// ErrNoReadersRegistered is for when a user has elected to use a registered reader but the list for their account returns empty
	ErrNoReadersRegistered = errors.New("no readers currently registered")
	// ErrConnectionTokenFailed is for when the call to Stripe for a new Terminal connection token went wonky
	ErrConnectionTokenFailed = errors.New("could not create new connection token")
	// ErrNewRPCSessionFailed is for when a new RPC Session (via the Stripe API not Rabbit) could not be created
	ErrNewRPCSessionFailed = errors.New("could not create new Terminal session")
	// ErrNewPaymentIntentFailed is for when calling Stripe for a new shiny Payment Intent failed
	ErrNewPaymentIntentFailed = errors.New("could not create new Payment Intent")
	// ErrCapturePaymentIntentFailed is for when you need to manually collect the Payment Intent after a Payment Method is attached and it failed
	ErrCapturePaymentIntentFailed = errors.New("could not capture the Payment Intent")
	// ErrSetReaderDisplayFailed is for when the Rabbit call to update the reader display didn't work as planned
	ErrSetReaderDisplayFailed = errors.New("could not set the Reader's display")
	// ErrClearReaderDisplayFailed is for when you're canceling a payment collection and need to reset the display back to default splash but it failed
	ErrClearReaderDisplayFailed = errors.New("could not clear the Reader's display")
	// ErrCollectPaymentFailed is for when the Rabbit call for the reader to go into collect payment state failed
	ErrCollectPaymentFailed = errors.New("could not collect payment method")
	// ErrCollectPaymentTimeout is for when the user didn't boop the card on the reader in a reasonable time
	ErrCollectPaymentTimeout = errors.New("timed out waiting for payment to be presented")
	// ErrConfirmPaymentFailed is for when the Rabbit call to confirm the payment method collected has failed
	ErrConfirmPaymentFailed = errors.New("could not confirm the payment")
	// ErrQueryPaymentFailed is for when you're polling Rabbit to see if the user has booped their card on the reader yet but something went wrong
	ErrQueryPaymentFailed = errors.New("could not query the payment")
	// ErrDNSFailed is for when a reader's address could not be resolved by DNS while attempting to contact it via Rabbit Service
	ErrDNSFailed = errors.New("couldn't find your reader on the network. We think it's probably a DNS issue.\n See our troubleshooting docs here: https://stripe.com/docs/terminal/readers/verifone-p400#troubleshooting")
	// ErrRabbitRequestCreationFailed is for when a Rabbit Service request is being rolled and the first stage of setting it up with the http client instance fails
	ErrRabbitRequestCreationFailed = errors.New("could not prepare request for Reader")
	// ErrStripeForbiddenResponse is for when a Stripe API call fails due to Terminal resource calls not supporting restricted keys
	ErrStripeForbiddenResponse = errors.New("it seems that your Stripe API Key is either restricted or out of date. Are you using a valid Stripe Secret Key with the --api-key global flag for this command?")
	// ErrStripeGenericResponse is for when any non status code happens for a Stripe call that isn't a 200 or a 403. This should be exceedingly rare (famous last words)
	ErrStripeGenericResponse = errors.New("could not connect to Stripe, perhaps try again?")
)
