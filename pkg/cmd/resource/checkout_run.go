package resource

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// CheckoutResponse parses the body for a checkout request to extract the id
type CheckoutResponse struct {
	ID string `json:"id"`
}

// CheckoutRunCmd sets the structure of the checkout command
type CheckoutRunCmd struct {
	Cmd *cobra.Command
	Cfg *config.Config

	publishableKey string
	sessionID      string
}

func (crc *CheckoutRunCmd) checkoutHandler(w http.ResponseWriter, req *http.Request) {
	template := `
<html>
<head>
<script src="https://js.stripe.com/v3/"></script>
<script>
var stripe = Stripe('%s');
stripe.redirectToCheckout({
	sessionId: '%s'
  }).then(function (result) {
	console.log(result);
  });
</script>
</head>
<body></body>
</html>
`

	fmt.Fprint(w, fmt.Sprintf(template, crc.publishableKey, crc.sessionID))
}

func (crc *CheckoutRunCmd) createCheckoutSession() error {
	secretKey, err := crc.Cfg.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	data := requests.RequestParameters{}
	data.AppendData([]string{
		`success_url=https://localhost:4242/success`,
		`cancel_url=https://example.com/cancel`,
		`payment_method_types[0]=card`,
		`line_items[0][name]=T-shirt`,
		`line_items[0][description]=Comfortable cotton t-shirt`,
		`line_items[0][amount]=1500`,
		`line_items[0][currency]=usd`,
		`line_items[0][quantity]=2`,
	})

	req := requests.Base{
		Method:         "POST",
		Profile:        &crc.Cfg.Profile,
		SuppressOutput: true,
		APIBaseURL:     "https://api.stripe.com",
	}

	resp, err := req.MakeRequest(secretKey, "/v1/checkout/sessions", &data, false)
	if err != nil {
		return err
	}

	checkoutResp := CheckoutResponse{}
	err = json.Unmarshal(resp, &checkoutResp)
	if err != nil {
		return err
	}

	crc.sessionID = checkoutResp.ID

	return nil
}

// NewCheckoutRunCmd returns a new CheckoutRunCmd.
func NewCheckoutRunCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	var port string

	cc := CheckoutRunCmd{
		Cfg: cfg,
	}

	cc.Cmd = &cobra.Command{
		Use:   "run",
		Args:  validators.NoArgs,
		Short: "Run checkout session",
		Long:  "Run checkout session",
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := os.Stdin.Stat()
			if err != nil {
				return err
			}

			// If we're not reading from a pipe, create a session
			// else read from the pipe
			if info.Mode()&os.ModeCharDevice != 0 || info.Size() <= 0 {
				cc.createCheckoutSession()
			} else {
				reader := bufio.NewReader(os.Stdin)
				var pipedInput []rune

				for {
					input, _, err := reader.ReadRune()
					if err != nil && err == io.EOF {
						break
					}
					pipedInput = append(pipedInput, input)
				}

				checkoutResp := CheckoutResponse{}
				err = json.Unmarshal([]byte(string(pipedInput)), &checkoutResp)
				if err != nil {
					return err
				}

				cc.sessionID = checkoutResp.ID
			}

			publishableKey, err := cfg.Profile.GetPublishableKey(false)
			if err != nil {
				return err
			}
			cc.publishableKey = publishableKey

			fmt.Println("Starting stripe server at address", fmt.Sprintf("http://0.0.0.0:%s", port))
			http.HandleFunc("/", cc.checkoutHandler)
			go func() {
				time.Sleep(1 * time.Second)
				open.Browser("http://0.0.0.0:4242")
			}()
			err = http.ListenAndServe(fmt.Sprintf("localhost:%s", port), handlers.LoggingHandler(os.Stdout, http.DefaultServeMux))

			return err
		},
	}

	cc.Cmd.Flags().StringVar(&port, "port", "4242", "Provide a custom port to serve content from.")

	parentCmd.AddCommand(cc.Cmd)

	return cc.Cmd
}
