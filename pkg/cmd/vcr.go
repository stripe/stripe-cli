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
}

func newVcrCmd() *vcrCmd {
	vc := &vcrCmd{}

	vc.cmd = &cobra.Command{
		Use:     "vcr",
		Args:    validators.NoArgs,
		Short:   "Start a VCR server",
		Long:    `TODO.`,
		Example: `TODO`,
		RunE:    vc.runVcrCmd,
	}

	vc.cmd.Flags().BoolVar(&vc.replayMode, "replaymode", false, "Replay events (default: record)")
	vc.cmd.Flags().StringVar(&vc.address, "address", ":8080", "Address to serve on")
	vc.cmd.Flags().StringVar(&vc.filepath, "cassette", "default_cassette.yaml", "The cassette file to use")

	// // Hidden configuration flags, useful for dev/debugging
	vc.cmd.Flags().StringVar(&vc.apiBaseURL, "api-base", "https://api.stripe.com", "Sets the API base URL")

	return vc
}

func (vc *vcrCmd) runVcrCmd(cmd *cobra.Command, args []string) error {
	// filepath := "main_result.yaml"
	// addressString := "localhost:8080"
	// recordMode := true
	// remoteURL := "https://api.stripe.com"
	// remoteURL := "https://gobyexample.com"

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

	fmt.Println()
	fmt.Printf("===\nUsing cassette \"%v\".\nListening via HTTPS on %v\nRecordMode: %v\n===", filepath, addressString, recordMode)

	fmt.Println()

	server := httpWrapper.InitializeServer(addressString)

	return server.ListenAndServeTLS("pkg/vcr/cert.pem", "pkg/vcr/key.pem")
}
