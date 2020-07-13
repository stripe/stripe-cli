package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/playback"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const defaultPort = 13111
const defaultWebhookPort = 13112

type playbackCmd struct {
	cmd *cobra.Command

	mode string

	apiBaseURL     string
	filepath       string
	address        string
	webhookAddress string
	serveHTTPS     bool
}

func newPlaybackCmd() *playbackCmd {
	pc := &playbackCmd{}

	pc.cmd = &cobra.Command{
		Use:   "playback",
		Args:  validators.NoArgs,
		Short: "Start a `playback` server",
		Long: `The playback command starts a local proxy server that intercepts outgoing requests to the Stripe API.

There are three modes of operation:

- "record": Any requests received are forwarded to the api.stripe.com, and the response is returned. All interactions
are written to the loaded 'cassette' file for later playback in replay mode.

- "replay": All received requests are terminated at the playback server, and responses are played back[1] from a cassette file. A existing cassette most be loaded.

- "auto": The server determines whether to run in "record" or "replay" mode on a per-cassette basis. If the cassette exists, operates in "replay" mode. If not, operates in "record" mode.

Currently, stripe playback only supports serving over HTTP.

// TODO: add documentation of control endpoints

[1]: requests are currently replayed sequentially in the same order they were recorded.
`,
		Example: `stripe playback
  stripe playback --replaymode
  stripe playback --cassette "my_cassette.yaml"`,
		RunE: pc.runPlaybackCmd,
	}

	pc.cmd.Flags().StringVar(&pc.mode, "mode", "auto", "Auto: record if cassette doesn't exist, replay if exists. Record: always record/re-record. Replay: always replay.")
	pc.cmd.Flags().StringVar(&pc.address, "address", fmt.Sprintf("localhost:%d", defaultPort), "Address to serve on")
	pc.cmd.Flags().StringVar(&pc.webhookAddress, "forward-to", fmt.Sprintf("localhost:%d", defaultWebhookPort), "Address to forward webhooks to")
	pc.cmd.Flags().StringVar(&pc.filepath, "cassette", "default_cassette.yaml", "The cassette file to use")
	// pc.cmd.Flags().BoolVar(&pc.serveHTTPS, "https", false, "Serve over HTTPS (default: HTTP")

	// // Hidden configuration flags, useful for dev/debugging
	pc.cmd.Flags().StringVar(&pc.apiBaseURL, "api-base", "https://api.stripe.com", "Sets the API base URL")

	return pc
}

func (pc *playbackCmd) runPlaybackCmd(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("Seting up playback server...")
	fmt.Println()

	filepath := pc.filepath
	addressString := pc.address
	remoteURL := pc.apiBaseURL

	// --- Validate command-line args
	if pc.mode != playback.Auto && pc.mode != playback.Record && pc.mode != playback.Replay {
		return errors.New(
			fmt.Sprintf(
				"\"%v\" is not a valid mode. It must be either \"%v\", \"%v\", or \"%v\"",
				pc.mode, playback.Auto, playback.Record, playback.Replay))
	}
	// TODO: disable HTTPS for now. it needs more work to be able to run in a released version of the stripe-cli
	// eg: how should we package cert.pem and key.pem? (see stripe-mock's implementation)
	pc.serveHTTPS = false

	var webhookAddress string
	if pc.serveHTTPS {
		webhookAddress = "https://" + pc.webhookAddress
	} else {
		webhookAddress = "http://" + pc.webhookAddress
	}

	// TODO: figure out interface for webhook / stripe listen configuration
	httpWrapper, err := playback.NewRecordReplayServer(remoteURL, webhookAddress)
	if err != nil {
		return nil
	}

	server := httpWrapper.InitializeServer(addressString)

	if pc.serveHTTPS {
		certFilepath := "pkg/playback/cert.pem"
		keyFilepath := "pkg/playback/key.pem"
		_, err := os.Stat(certFilepath)
		if err != nil {
			return fmt.Errorf("Error when loading cert.pem, are you sure it exists?: %w", err)
		}

		_, err = os.Stat(certFilepath)
		if err != nil {
			return fmt.Errorf("Error when loading key.pem, are you sure it exists?: %w", err)
		}

		go func() {
			server.ListenAndServeTLS(certFilepath, keyFilepath)
		}()

	} else {
		go func() {
			server.ListenAndServe()
		}()
	}

	var fullAddressString string
	if pc.serveHTTPS {
		fullAddressString = "https://" + addressString
	} else {
		fullAddressString = "http://" + addressString
	}

	// TODO: (is this okay?) create a custom client to skip verifying our self-signed HTTPS certificate
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(fullAddressString + "/pb/mode/" + pc.mode)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("Non 200 status code received during startup: " + string(resp.Status))
	}

	resp, err = client.Get(fullAddressString + "/pb/cassette/load?filepath=" + filepath)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("Non 200 status code received during startup: " + string(resp.Status))
	}

	fmt.Println()
	fmt.Println("------ Server Running ------")

	switch pc.mode {
	case playback.Record:
		fmt.Printf("In \"record\" mode.\n")
		fmt.Println("Will always record interactions, and write (or overwrite) to the given cassette filepath.")
		fmt.Println()
	case playback.Replay:
		fmt.Printf("In \"replay\" mode.\n")
		fmt.Println("Will always replay from the given cassette. Will error if loaded cassette path doesn't exist.")
		fmt.Println()
	case playback.Auto:
		fmt.Printf("In \"auto\" mode.\n")
		fmt.Println("Can both record or replay, depending on the file passed in. If exists, replays. If not, records.")
		fmt.Println()
	}

	fmt.Printf("Using cassette: \"%v\".\n", filepath)
	fmt.Println()

	if pc.serveHTTPS {
		fmt.Printf("Listening via HTTPS on %v\n", addressString)
	} else {
		fmt.Printf("Listening via HTTP on %v\n", addressString)
	}

	fmt.Printf("Forwarding webhooks to %v\n", webhookAddress)

	fmt.Println("-----------------------------")
	fmt.Println()

	select {}
}
