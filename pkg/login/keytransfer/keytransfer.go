package keytransfer

import (
	"context"
	"time"

	"github.com/stripe/stripe-cli/pkg/login/acct"
)

type AsyncPollResult struct {
	TestModeAPIKey string
	Account        *acct.Account
	Err            error
}

type KeyTransfer interface {
	AsyncPollKey(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int, ch chan AsyncPollResult)
}

type RAKTransfer struct {
	configurer *Configurer
}

func NewRAKTransfer(configurer *Configurer) *RAKTransfer {
	return &RAKTransfer{
		configurer: configurer,
	}
}

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
