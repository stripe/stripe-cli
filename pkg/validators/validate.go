package validators

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ArgValidator is an argument validator. It accepts a string and returns an
// error if the string is invalid, or nil otherwise.
type ArgValidator func(string) error

var (
	// ErrAPIKeyNotConfigured is the error returned when the loaded profile is missing the api key property
	ErrAPIKeyNotConfigured = errors.New("you have not configured API keys yet")
	// ErrDeviceNameNotConfigured is the error returned when the loaded profile is missing the device name property
	ErrDeviceNameNotConfigured = errors.New("you have not configured your device name yet")
	// ErrAccountIDNotConfigured is the error returned when the loaded profile is missing the account_id property
	ErrAccountIDNotConfigured = errors.New("you have not configured your accountID yet")
)

// CallNonEmptyArray calls an argument validator on all non-empty elements of
// a string array.
func CallNonEmptyArray(validator ArgValidator, values []string) error {
	if len(values) == 0 {
		return nil
	}

	for _, value := range values {
		err := CallNonEmpty(validator, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// CallNonEmpty calls an argument validator on a string if the string is not
// empty.
func CallNonEmpty(validator ArgValidator, value string) error {
	if value == "" {
		return nil
	}

	return validator(value)
}

// APIKey validates that a string looks like an API key.
func APIKey(input string) error {
	if len(input) == 0 {
		return ErrAPIKeyNotConfigured
	} else if len(input) < 12 {
		return errors.New("the API key provided is too short, it must be at least 12 characters long")
	}

	keyParts := strings.Split(input, "_")
	if len(keyParts) < 3 {
		return errors.New("you are using a legacy-style API key which is unsupported by the CLI. Please generate a new test mode API key")
	}

	if keyParts[0] != "sk" && keyParts[0] != "rk" {
		return errors.New("the CLI only supports using a secret or restricted key")
	}

	return nil
}

// APIKeyNotRestricted validates that a string looks like a secret API key and is not a restricted key.
func APIKeyNotRestricted(input string) error {
	if len(input) == 0 {
		return ErrAPIKeyNotConfigured
	} else if len(input) < 12 {
		return errors.New("the API key provided is too short, it must be at least 12 characters long")
	}

	keyParts := strings.Split(input, "_")
	if len(keyParts) < 3 {
		return errors.New("you are using a legacy-style API key which is unsupported by the CLI. Please generate a new test mode API key")
	}

	if keyParts[0] != "sk" || keyParts[0] == "rk" {
		return errors.New("this CLI command only supports using a secret key. Please re-run using the --api-key flag override with your secret API key")
	}

	return nil
}

// Account validates that a string is an acceptable account filter.
func Account(account string) error {
	accountUpper := strings.ToUpper(account)

	if accountUpper == "CONNECT_IN" || accountUpper == "CONNECT_OUT" || accountUpper == "SELF" {
		return nil
	}

	return fmt.Errorf("%s is not an acceptable account filter (CONNECT_IN, CONNECT_OUT, SELF)", account)
}

// HTTPMethod validates that a string is an acceptable HTTP method.
func HTTPMethod(method string) error {
	methodUpper := strings.ToUpper(method)

	if methodUpper == http.MethodGet || methodUpper == http.MethodPost || methodUpper == http.MethodDelete {
		return nil
	}

	return fmt.Errorf("%s is not an acceptable HTTP method (GET, POST, DELETE)", method)
}

// RequestSource validates that a string is an acceptable request source.
func RequestSource(source string) error {
	sourceUpper := strings.ToUpper(source)

	if sourceUpper == "API" || sourceUpper == "DASHBOARD" {
		return nil
	}

	return fmt.Errorf("%s is not an acceptable source (API, DASHBOARD)", source)
}

// RequestStatus validates that a string is an acceptable request status.
func RequestStatus(status string) error {
	statusUpper := strings.ToUpper(status)

	if statusUpper == "SUCCEEDED" || statusUpper == "FAILED" {
		return nil
	}

	return fmt.Errorf("%s is not an acceptable request status (SUCCEEDED, FAILED)", status)
}

// StatusCode validates that a provided status code is within the range of
// those used in the Stripe API.
func StatusCode(code string) error {
	num, err := strconv.Atoi(code)
	if err != nil {
		return err
	}

	if num >= 200 && num < 300 {
		return nil
	}

	if num >= 400 && num < 600 {
		return nil
	}

	return fmt.Errorf("Provided status code %s is not in the range of acceptable status codes (200's, 400's, 500's)", code)
}

// StatusCodeType validates that a provided status code type is one of those
// used in the Stripe API.
func StatusCodeType(code string) error {
	codeUpper := strings.ToUpper(code)

	if codeUpper != "2XX" && codeUpper != "4XX" && codeUpper != "5XX" {
		return fmt.Errorf("Provided status code type %s is not a valid type (2XX, 4XX, 5XX)", code)
	}

	return nil
}

// OneDollar validates that a provided number is at least 100 (ie. 1 dollar)
func OneDollar(number string) error {
	num, err := strconv.Atoi(number)
	if err != nil {
		return fmt.Errorf("Provided amount %v to charge should be an integer (eg. 100)", number)
	}

	if num >= 100 {
		return nil
	}

	return fmt.Errorf("Provided amount %v to charge is not at least 100", number)
}
