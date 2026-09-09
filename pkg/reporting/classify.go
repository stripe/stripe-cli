package reporting

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"

	"github.com/spf13/pflag"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
)

func classifyError(err error) errorcategory.Category {
	if category, ok := errorcategory.Get(err); ok {
		return category
	}

	var flagNotFound *pflag.NotExistError
	if errors.As(err, &flagNotFound) {
		return errorcategory.UserInput
	}

	var flagValueRequired *pflag.ValueRequiredError
	if errors.As(err, &flagValueRequired) {
		return errorcategory.UserInput
	}

	if statusCode, ok := requestErrorStatusCode(err); ok {
		switch statusCode {
		case 401, 403:
			return errorcategory.Auth
		case 429:
			return errorcategory.RateLimit
		}
		return errorcategory.API
	}

	var pathError *os.PathError
	if errors.As(err, &pathError) {
		return errorcategory.Filesystem
	}

	var urlError *url.Error
	if errors.As(err, &urlError) {
		return errorcategory.Network
	}

	var networkError net.Error
	if errors.As(err, &networkError) {
		return errorcategory.Network
	}

	if errors.Is(err, context.Canceled) {
		return errorcategory.UserInput
	}

	return errorcategory.Internal
}

func requestErrorStatusCode(err error) (int, bool) {
	var requestError requests.RequestError
	if errors.As(err, &requestError) {
		return requestError.StatusCode, true
	}

	var requestErrorPointer *requests.RequestError
	if errors.As(err, &requestErrorPointer) {
		return requestErrorPointer.StatusCode, true
	}

	return 0, false
}
