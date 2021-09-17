package terminal

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/manifoldco/promptui"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/terminal/p400"
)

// QuickstartP400 runs the quickstart interactive prompt sequence to walk the user through setting up a P400 reader
func QuickstartP400(ctx context.Context, cfg *config.Config) error {
	tsCtx := SetTerminalSessionContext(cfg)

	tsCtx, err := p400.RegisterAndActivateReader(ctx, tsCtx)

	if err != nil {
		if err.Error() == promptui.ErrInterrupt.Error() {
			os.Exit(1)
		} else {
			return fmt.Errorf(err.Error())
		}
	}

	fmt.Println("Got it!")

	tsCtx, err = p400.SetUpTestPayment(ctx, tsCtx)

	if err != nil {
		p400.ClearReaderDisplay(tsCtx)
		if err.Error() == promptui.ErrInterrupt.Error() {
			os.Exit(1)
		} else {
			return fmt.Errorf(err.Error())
		}
	}

	tsCtx, err = p400.CompleteTestPayment(ctx, tsCtx)

	if err != nil {
		p400.ClearReaderDisplay(tsCtx)
		if err.Error() == promptui.ErrInterrupt.Error() {
			os.Exit(1)
		} else {
			return fmt.Errorf(err.Error())
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
