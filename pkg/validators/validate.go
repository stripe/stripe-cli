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
	if len(input) < 12 {
		return errors.New("API key is too short, must be at least 12 characters long")
	}

	keyParts := strings.Split(input, "_")
	if len(keyParts) < 3 {
		return errors.New("you are using a legacy-style API key which is unsupported by the CLI. Please generate a new test mode API key")
	}

	if keyParts[0] != "sk" && keyParts[0] != "rk" {
		return errors.New("the CLI only supports using a secret or restricted key")
	}

	if keyParts[1] != "test" {
		return errors.New("the CLI only supports using a test mode key")
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
