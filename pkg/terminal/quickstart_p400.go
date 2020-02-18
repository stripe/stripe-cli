package terminal

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/terminal/p400"
)

// QuickstartP400 runs the quickstart interactive prompt sequence to walk the user through setting up a P400 reader
func QuickstartP400(cfg *config.Config) error {
	tsCtx := SetTerminalSessionContext(cfg)

	// reset the reader's state on SIGINT
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt)

	go func() {
		// block unless SIGINT occurs
		<-interruptChannel
		p400.ClearReaderDisplay(tsCtx)
		os.Exit(1)
	}()

	tsCtx, err := p400.RegisterAndActivateReader(tsCtx)

	if err != nil {
		switch e := err.Error(); e {
		case p400.ErrRegisterReaderFailed.Error():
			fallthrough

		case p400.ErrActivateReaderFailed.Error():
			fallthrough

		case p400.ErrConnectionTokenFailed.Error():
			fallthrough

		case p400.ErrNewRPCSessionFailed.Error():
			return fmt.Errorf(err.Error())

		default:
			os.Exit(1)
		}
	}

	fmt.Println("Got it!")

	tsCtx, err = p400.SetUpTestPayment(tsCtx)

	if err != nil {
		switch e := err.Error(); e {
		case p400.ErrNewPaymentIntentFailed.Error():
			p400.ClearReaderDisplay(tsCtx)
			fallthrough

		case p400.ErrSetReaderDisplayFailed.Error():
			return fmt.Errorf(err.Error())

		default:
			os.Exit(1)
		}
	}

	tsCtx, err = p400.CompleteTestPayment(tsCtx)

	if err != nil {
		p400.ClearReaderDisplay(tsCtx)

		switch e := err.Error(); e {
		case p400.ErrCapturePaymentIntentFailed.Error():
			fallthrough

		case p400.ErrCollectPaymentFailed.Error():
			fallthrough

		case p400.ErrCollectPaymentTimeout.Error():
			fallthrough

		case p400.ErrConfirmPaymentFailed.Error():
			fallthrough

		case p400.ErrQueryPaymentFailed.Error():
			return fmt.Errorf(err.Error())

		default:
			os.Exit(1)
		}
	}

	p400.SummarizeQuickstartCompletion(tsCtx)

	return nil
}

// SetTerminalSessionContext creates a data struct that contains the context of the user's current quickstart session
// it returns a TerminalSessionContext interface that is passed into most of the P400 reader related functions in the quickstart flow
func SetTerminalSessionContext(cfg *config.Config) p400.TerminalSessionContext {
	apiKey, _ := cfg.Profile.GetAPIKey(false)
	posID := cfg.Profile.GetTerminalPOSDeviceID()

	if posID == "" {
		seed := time.Now().UnixNano()
		posID = p400.GeneratePOSDeviceID(seed)
		cfg.Profile.WriteConfigField("terminal_pos_device_id", posID)
	}

	hostOSVersion := p400.GetOSString()
	POSInfoDescription := fmt.Sprintf("%v:StripeCLI", hostOSVersion)

	tsCtx := p400.TerminalSessionContext{
		APIKey: apiKey,
		DeviceInfo: p400.DeviceInfo{
			DeviceClass:   "POS",
			DeviceUUID:    posID,
			HostOSVersion: hostOSVersion,
			HardwareModel: p400.HardwareModel{
				POSInfo: p400.POSInfo{
					Description: POSInfoDescription,
				},
			},
			AppModel: p400.AppModel{
				AppID:      "Stripe-CLI-Terminal-Quickstart",
				AppVersion: "https://stripe.com/docs/stripe-cli",
			},
		},
	}

	return tsCtx
}
