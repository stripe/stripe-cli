package checkout

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
)

type response struct {
	ID string `json:"id"`
}

func createCheckoutSession(cfg *config.Config, port string) (string, error) {
	secretKey, err := cfg.Profile.GetAPIKey(false)
	if err != nil {
		return "", err
	}

	data := requests.RequestParameters{}
	data.AppendData([]string{
		fmt.Sprintf(`success_url=http://localhost:%s/success`, port),
		fmt.Sprintf(`cancel_url=http://localhost:%s/cancel`, port),
		`payment_method_types[0]=card`,
		`line_items[0][name]=Increment`,
		`line_items[0][description]=Software Architecture`,
		`line_items[0][amount]=1500`,
		`line_items[0][currency]=usd`,
		`line_items[0][quantity]=1`,
	})

	req := requests.Base{
		Method:         "POST",
		Profile:        &cfg.Profile,
		SuppressOutput: true,
		APIBaseURL:     "https://api.stripe.com",
	}

	resp, err := req.MakeRequest(secretKey, "/v1/checkout/sessions", &data, false)
	if err != nil {
		return "", err
	}

	checkoutResp := response{}
	err = json.Unmarshal(resp, &checkoutResp)
	if err != nil {
		return "", err
	}

	return checkoutResp.ID, nil
}

func getOrCreateSession(cfg *config.Config, port string) (string, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	// If we're not reading from a pipe, create a session
	// else read from the pipe
	if info.Mode()&os.ModeCharDevice != 0 || info.Size() <= 0 {
		return createCheckoutSession(cfg, port)
	}

	reader := bufio.NewReader(os.Stdin)
	var pipedInput []rune

	for {
		input, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}
		pipedInput = append(pipedInput, input)
	}

	checkoutResp := response{}
	err = json.Unmarshal([]byte(string(pipedInput)), &checkoutResp)
	if err != nil {
		return "", err
	}

	return checkoutResp.ID, nil
}
