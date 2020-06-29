package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/vcr"
)

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

VCR Server Control via HTTP Endpoints
/vcr/stop: stops recording and writes the current session to cassette`,
		Example: `stripe vcr
  stripe vcr --replaymode
  stripe vcr --https --cassette "my_cassette.yaml"`,
		RunE: vc.runVcrCmd,
	}

	vc.cmd.Flags().BoolVar(&vc.replayMode, "replaymode", false, "Replay events (default: record)")
	vc.cmd.Flags().StringVar(&vc.address, "address", ":8080", "Address to serve on")
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

	var httpWrapper vcr.VcrHttpServer

	if recordMode {
		// delete file if exists
		if _, err := os.Stat(filepath); !os.IsNotExist(err) {
			err = os.Remove(filepath)
			if err != nil {
				return err
			}
		}

		fileWriteHandle, err := os.Create(filepath)
		defer fileWriteHandle.Close()
		if err != nil {
			return err
		}

		httpRecorder, err := vcr.NewHttpRecorder(fileWriteHandle, remoteURL)
		if err != nil {
			return err
		}
		httpWrapper = &httpRecorder
	} else {
		// Make sure file exists
		_, err := os.Stat(filepath)
		if err != nil {
			return err
		}

		fileReadHandle, err := os.Open(filepath)
		defer fileReadHandle.Close()
		if err != nil {
			return err
		}

		httpReplayer, err := vcr.NewHttpReplayer(fileReadHandle)
		if err != nil {
			return err
		}
		httpWrapper = &httpReplayer
	}

	server := httpWrapper.InitializeServer(addressString)

	if vc.serveHTTPS {
		fmt.Println()
		fmt.Printf("===\nUsing cassette \"%v\".\nListening via HTTPS on %v\nRecordMode: %v\n===", filepath, addressString, recordMode)

		fmt.Println()
		return server.ListenAndServeTLS("pkg/vcr/cert.pem", "pkg/vcr/key.pem")

	} else {
		fmt.Println()
		fmt.Printf("===\nUsing cassette \"%v\".\nListening via HTTP on %v\nRecordMode: %v\n===", filepath, addressString, recordMode)

		fmt.Println()
		return server.ListenAndServe()
	}
}
