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

	apiBaseURL     string
	filepath       string
	address        string
	webhookAddress string
	replayMode     bool
	serveHTTPS     bool
}

func newPlaybackCmd() *playbackCmd {
	pc := &playbackCmd{}

	pc.cmd = &cobra.Command{
		Use:   "playback",
		Args:  validators.NoArgs,
		Short: "Start a `playback` server",
		Long: `The playback command starts a local proxy server that intercepts outgoing requests to the Stripe API.

If run in record mode, this server acts as a transparent layer between the client and Stripe,
recording the request/response pairs in a cassette file.

If run in replay mode, requests are terminated at the playback server, and responses are played back from a cassette file.

Playback Server Control via HTTP Endpoints:
/pb/stop: stops recording and writes the current session to cassette`,
		Example: `stripe playback
  stripe playback --replaymode
  stripe playback --https --cassette "my_cassette.yaml"`,
		RunE: pc.runPlaybackCmd,
	}

	pc.cmd.Flags().BoolVar(&pc.replayMode, "replaymode", false, "Replay events (default: record)")
	pc.cmd.Flags().StringVar(&pc.address, "address", fmt.Sprintf("localhost:%d", defaultPort), "Address to serve on")
	pc.cmd.Flags().StringVar(&pc.webhookAddress, "forward-to", fmt.Sprintf("localhost:%d", defaultWebhookPort), "Address to forward webhooks to")
	pc.cmd.Flags().StringVar(&pc.filepath, "cassette", "default_cassette.yaml", "The cassette file to use")
	pc.cmd.Flags().BoolVar(&pc.serveHTTPS, "https", false, "Serve over HTTPS (default: HTTP")

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
	recordMode := !pc.replayMode
	remoteURL := pc.apiBaseURL

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

	if recordMode {
		resp, err := client.Get(fullAddressString + "/pb/mode/record")
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return errors.New("Non 200 status code received during startup: " + string(resp.Status))
		}
	} else {
		resp, err := client.Get(fullAddressString + "/pb/mode/replay")
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return errors.New("Non 200 status code received during startup: " + string(resp.Status))
		}
	}

	resp, err := client.Get(fullAddressString + "/pb/cassette/load?filepath=" + filepath)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("Non 200 status code received during startup: " + string(resp.Status))
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
