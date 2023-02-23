package keys

import (
	"context"
	"time"

	"github.com/stripe/stripe-cli/pkg/login/acct"
)

// AsyncPollResult is the data returned from polling for keys
type AsyncPollResult struct {
	TestModeAPIKey string
	Account        *acct.Account
	Err            error
}

// KeyTransfer handles polling for API keys
type KeyTransfer interface {
	AsyncPollKey(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int, ch chan AsyncPollResult)
}

// RAKTransfer implements KeyTransfer to poll for RAKs
type RAKTransfer struct {
	configurer Configurer
}

// NewRAKTransfer creates a new RAKTransfer object
func NewRAKTransfer(configurer Configurer) *RAKTransfer {
	return &RAKTransfer{
		configurer: configurer,
	}
}

// AsyncPollKey polls for RAKs
func (rt *RAKTransfer) AsyncPollKey(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int, ch chan AsyncPollResult) {
	defer close(ch)

	response, account, err := PollForKey(ctx, pollURL, interval, maxAttempts)
	if err != nil {
		ch <- AsyncPollResult{
			TestModeAPIKey: "",
			Account:        nil,
			Err:            err,
		}
		return
	}

	err = rt.configurer.SaveLoginDetails(response)
	if err != nil {
		ch <- AsyncPollResult{
			TestModeAPIKey: "",
			Account:        nil,
			Err:            err,
		}
		return
	}

	ch <- AsyncPollResult{
		TestModeAPIKey: response.TestModeAPIKey,
		Account:        account,
		Err:            nil,
	}
}
