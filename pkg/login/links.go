package login

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/version"
)

// Links provides the URLs for the CLI to continue the login flow
type Links struct {
	BrowserURL       string `json:"browser_url"`
	PollURL          string `json:"poll_url"`
	VerificationCode string `json:"verification_code"`
}

// GetLinks provides the URLs for the CLI to continue the login flow.
//
// The machineUUID is sent to the server so it can gate individual machines
// into the OAuth device-code flow via feature flag. When the server responds
// with a 3xx, the caller should switch to LoginWithDeviceCode instead of
// proceeding with the legacy RAK flow; in that case GetLinks returns
// (nil, true, nil).
func GetLinks(ctx context.Context, baseURL string, deviceName string, machineUUID string) (*Links, bool, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, false, err
	}

	client := &stripe.Client{
		BaseURL:           parsedBaseURL,
		NoFollowRedirects: true,
	}

	data := url.Values{}
	data.Set("client_version", version.Version)
	data.Set("device_name", deviceName)
	if machineUUID != "" {
		data.Set("machine_uuid", machineUUID)
	}

	res, err := client.PerformRequest(ctx, http.MethodPost, stripeCLIAuthPath, data.Encode(), nil)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusMultipleChoices && res.StatusCode < http.StatusBadRequest {
		return nil, true, nil
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, false, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, false, errorcategory.Errorf(errorcategory.Auth, "unexpected http status code: %d %s", res.StatusCode, string(bodyBytes))
	}

	var links Links
	if err := json.Unmarshal(bodyBytes, &links); err != nil {
		return nil, false, err
	}

	return &links, false, nil
}
