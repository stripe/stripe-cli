package keytransfer

import (
	"context"
	"time"

	"github.com/stripe/stripe-cli/pkg/login/acct"
	"github.com/stripe/stripe-cli/pkg/login/configurer"
	"github.com/stripe/stripe-cli/pkg/login/polling"
)

type AsyncPollResult struct {
	TestModeAPIKey string
	Account        *acct.Account
	Err            error
}

type IKeyTransfer interface {
	AsyncPollKey(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int, ch chan AsyncPollResult)
}

type KeyTransfer struct {
	configurer *configurer.Configurer
}

func NewKeyTransfer(configurer *configurer.Configurer) *KeyTransfer {
	return &KeyTransfer{
		configurer: configurer,
	}
}

func (kt *KeyTransfer) AsyncPollKey(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int, ch chan AsyncPollResult) {
	defer close(ch)

	response, account, err := polling.PollForKey(ctx, pollURL, interval, maxAttempts)
	if err != nil {
		ch <- AsyncPollResult{
			TestModeAPIKey: "",
			Account:        nil,
			Err:            err,
		}
		return
	}

	err = kt.configurer.SaveLoginDetails(response)
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
