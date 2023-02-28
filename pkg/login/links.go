package login

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

// Links provides the URLs for the CLI to continue the login flow
type Links struct {
	BrowserURL       string `json:"browser_url"`
	PollURL          string `json:"poll_url"`
	VerificationCode string `json:"verification_code"`
}

// GetLinks provides the URLs for the CLI to continue the login flow
func GetLinks(ctx context.Context, baseURL string, deviceName string) (*Links, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
	}

	data := url.Values{}
	data.Set("device_name", deviceName)

	res, err := client.PerformRequest(ctx, http.MethodPost, stripeCLIAuthPath, data.Encode(), nil)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http status code: %d %s", res.StatusCode, string(bodyBytes))
	}

	var links Links

	err = json.Unmarshal(bodyBytes, &links)
	if err != nil {
		return nil, err
	}

	return &links, nil
}
