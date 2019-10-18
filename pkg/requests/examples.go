package requests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/stripe/stripe-cli/pkg/config"
)

const (
	validToken        = "tok_visa"
	declinedToken     = "tok_chargeDeclined"
	disputeToken      = "tok_createDisputeInquiry"
	chargeFailedToken = "tok_chargeCustomerFail"
)

func parseResponse(response []byte) (map[string]interface{}, error) {
	var result map[string]interface{}

	err := json.Unmarshal(response, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Examples stores possible webhook test events to trigger for the CLI
type Examples struct {
	Profile    config.Profile
	APIBaseURL string
	APIVersion string
	APIKey     string
}

func (ex *Examples) buildRequest(method string, data []string) (*Base, *RequestParameters) {
	params := &RequestParameters{
		data:    data,
		version: ex.APIVersion,
	}

	base := &Base{
		Profile:        &ex.Profile,
		Method:         method,
		SuppressOutput: true,
		APIBaseURL:     ex.APIBaseURL,
	}

	return base, params
}

func (ex *Examples) performStripeRequest(req *Base, endpoint string, params *RequestParameters) (map[string]interface{}, error) {
	resp, err := req.MakeRequest(ex.APIKey, endpoint, params, true)
	if err != nil {
		return nil, err
	}

	return parseResponse(resp)
}

// ResendEvent resends a webhook event using it's event-id "evt_<id>"
func (ex *Examples) ResendEvent(id string) error {
	pattern := `^evt_[A-Za-z0-9]{3,255}$`

	match, err := regexp.MatchString(pattern, id)
	if err != nil {
		return err
	}

	if !match {
		return fmt.Errorf("Invalid event-id provided, should be of the form '%s'", pattern)
	}

	req, params := ex.buildRequest(http.MethodPost, []string{})
	reqURL := fmt.Sprintf("/v1/events/%s/retry", id)
	_, err = ex.performStripeRequest(req, reqURL, params)

	return err
}
