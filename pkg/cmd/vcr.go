package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/vcr"
)

const defaultVCRPort = 13111
const defaultWebhookPort = 13112

type vcrCmd struct {
	cmd *cobra.Command

	apiBaseURL     string
	filepath       string
	address        string
	webhookAddress string
	replayMode     bool
	serveHTTPS     bool
}

func newVcrCmd() *vcrCmd {
	vc := &vcrCmd{}

	vc.cmd = &cobra.Command{
		Use:   "playback",
		Args:  validators.NoArgs,
		Short: "Start a `playback` server",
		Long: `The playback command starts a local proxy server that intercepts outgoing requests to the Stripe API.

If run in record mode, this server acts as a transparent layer between the client and Stripe,
recording the request/response pairs in a cassette file.

If run in replay mode, requests are terminated at the playback server, and responses are played back from a cassette file.

Playback Server Control via HTTP Endpoints:
/vcr/stop: stops recording and writes the current session to cassette`,
		Example: `stripe playback
  stripe playback --replaymode
  stripe playback --https --cassette "my_cassette.yaml"`,
		RunE: vc.runVcrCmd,
	}

	vc.cmd.Flags().BoolVar(&vc.replayMode, "replaymode", false, "Replay events (default: record)")
	vc.cmd.Flags().StringVar(&vc.address, "address", fmt.Sprintf("localhost:%d", defaultVCRPort), "Address to serve on")
	vc.cmd.Flags().StringVar(&vc.webhookAddress, "forward-to", fmt.Sprintf("localhost:%d", defaultWebhookPort), "Address to forward webhooks to")
	vc.cmd.Flags().StringVar(&vc.filepath, "cassette", "default_cassette.yaml", "The cassette file to use")
	vc.cmd.Flags().BoolVar(&vc.serveHTTPS, "https", false, "Serve over HTTPS (default: HTTP")

	// // Hidden configuration flags, useful for dev/debugging
	vc.cmd.Flags().StringVar(&vc.apiBaseURL, "api-base", "https://api.stripe.com", "Sets the API base URL")

	return vc
}

func (vc *vcrCmd) runVcrCmd(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("Seting up playback server...")
	fmt.Println()

	filepath := vc.filepath
	addressString := vc.address
	recordMode := !vc.replayMode
	remoteURL := vc.apiBaseURL

	var webhookAddress string
	if vc.serveHTTPS {
		webhookAddress = "https://" + vc.webhookAddress
	} else {
		webhookAddress = "http://" + vc.webhookAddress
	}

	// TODO: figure out interface for webhook / stripe listen configuration
	httpWrapper, err := vcr.NewRecordReplayServer(remoteURL, webhookAddress)
	if err != nil {
		return nil
	}

	server := httpWrapper.InitializeServer(addressString)

	if vc.serveHTTPS {
		go func() {
			server.ListenAndServeTLS("pkg/vcr/cert.pem", "pkg/vcr/key.pem")
		}()

	} else {
		go func() {
			server.ListenAndServe()
		}()
	}

	var fullAddressString string
	if vc.serveHTTPS {
		fullAddressString = "https://" + addressString
	} else {
		fullAddressString = "http://" + addressString
	}

	if recordMode {
		resp, err := http.Get(fullAddressString + "/vcr/mode/record")
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return errors.New("Non 200 status code received during VCR startup: " + string(resp.Status))
		}
	} else {
		resp, err := http.Get(fullAddressString + "/vcr/mode/replay")
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return errors.New("Non 200 status code received during VCR startup: " + string(resp.Status))
		}
	}

	resp, err := http.Get(fullAddressString + "/vcr/cassette/load?filepath=" + filepath)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("Non 200 status code received during VCR startup: " + string(resp.Status))
	}

	fmt.Println()
	fmt.Println("------ Server Running ------")

	if recordMode {
		fmt.Printf("Recording...\n")
	} else {
		fmt.Printf("Replaying...\n")
	}

	fmt.Printf("Using cassette: \"%v\".\n", filepath)
	fmt.Println()

	if vc.serveHTTPS {
		fmt.Printf("Listening via HTTPS on %v\n", addressString)
	} else {
		fmt.Printf("Listening via HTTP on %v\n", addressString)
	}

	fmt.Printf("Forwarding webhooks to %v\n", webhookAddress)

	fmt.Println("-----------------------------")
	fmt.Println()

	select {}
}
