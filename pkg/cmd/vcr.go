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

type vcrCmd struct {
	cmd *cobra.Command

	apiBaseURL string
	filepath   string
	address    string
	replayMode bool
	serveHTTPS bool
}

func newVcrCmd() *vcrCmd {
	vc := &vcrCmd{}

	vc.cmd = &cobra.Command{
		Use:   "vcr",
		Args:  validators.NoArgs,
		Short: "Start a VCR server",
		Long: `The vcr command starts a local proxy server that intercepts outgoing requests to the Stripe API.

If run in record mode, this server acts as a transparent layer between the client and Stripe,
recording the request/response pairs in a cassette file.

If run in replay mode, requests are terminated at the VCR server, and responses are played back from a cassette file.

VCR Server Control via HTTP Endpoints:
/vcr/stop: stops recording and writes the current session to cassette`,
		Example: `stripe vcr
  stripe vcr --replaymode
  stripe vcr --https --cassette "my_cassette.yaml"`,
		RunE: vc.runVcrCmd,
	}

	vc.cmd.Flags().BoolVar(&vc.replayMode, "replaymode", false, "Replay events (default: record)")
	vc.cmd.Flags().StringVar(&vc.address, "address", fmt.Sprintf(":%d", defaultVCRPort), "Address to serve on")
	vc.cmd.Flags().StringVar(&vc.filepath, "cassette", "default_cassette.yaml", "The cassette file to use")
	vc.cmd.Flags().BoolVar(&vc.serveHTTPS, "https", false, "Serve over HTTPS (default: HTTP")

	// // Hidden configuration flags, useful for dev/debugging
	vc.cmd.Flags().StringVar(&vc.apiBaseURL, "api-base", "https://api.stripe.com", "Sets the API base URL")

	return vc
}

func (vc *vcrCmd) runVcrCmd(cmd *cobra.Command, args []string) error {
	filepath := vc.filepath
	addressString := vc.address
	recordMode := !vc.replayMode
	remoteURL := vc.apiBaseURL

	httpWrapper, err := vcr.NewRecordReplayServer(remoteURL)
	if err != nil {
		return nil
	}

	server := httpWrapper.InitializeServer(addressString)

	if vc.serveHTTPS {
		fmt.Println()
		fmt.Printf("===\nUsing cassette \"%v\".\nListening via HTTPS on %v\nRecordMode: %v\n===", filepath, addressString, recordMode)

		fmt.Println()

		go func() {
			server.ListenAndServeTLS("pkg/vcr/cert.pem", "pkg/vcr/key.pem")
		}()

	} else {
		fmt.Println()
		fmt.Printf("===\nUsing cassette \"%v\".\nListening via HTTP on %v\nRecordMode: %v\n===", filepath, addressString, recordMode)

		fmt.Println()
		go func() {
			server.ListenAndServe()
		}()
	}

	fullAddressString := "localhost" + addressString
	if vc.serveHTTPS {
		fullAddressString = "https://" + fullAddressString
	} else {
		fullAddressString = "http://" + fullAddressString
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

	select {}
}
