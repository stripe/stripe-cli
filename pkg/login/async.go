package login

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/stripe/stripe-cli/pkg/login/acct"
	"github.com/stripe/stripe-cli/pkg/login/polling"
)

type pollResult struct {
	response *polling.PollAPIKeyResponse
	account  *acct.Account
	err      error
}

func asyncPollKey(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int, ch chan pollResult) {
	response, account, err := polling.PollForKey(ctx, pollURL, interval, maxAttempts)
	ch <- pollResult{
		response: response,
		account:  account,
		err:      err,
	}
	close(ch)
}

// AsyncInputReader is an interface that has an async version of scanln
type AsyncInputReader interface {
	scanln(ch chan int)
}

// AsyncStdinReader implements scanln(ch chan int), an async version of scanln
type AsyncStdinReader struct {
}

func (r AsyncStdinReader) scanln(ch chan int) {
	n, _ := fmt.Fscanln(os.Stdin)
	ch <- n
}
