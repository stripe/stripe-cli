package validators

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// APIKey validates an API key.
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

func HTTPMethod(method string) error {
	methodUpper := strings.ToUpper(method)

	if methodUpper == "GET" || methodUpper == "POST" || methodUpper == "DELETE" {
		return nil
	}

	return fmt.Errorf("%s is not an acceptable HTTP method (GET, POST, DELETE)", method)
}

func RequestSource(source string) error {
	sourceUpper := strings.ToUpper(source)

	if sourceUpper == "API" || sourceUpper == "DASHBOARD" {
		return nil
	}

	return fmt.Errorf("%s is not an acceptable source (API, DASHBOARD)", source)
}

// StatusCode validates that a provided status code is within the range of those used in the Stripe API
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

// StatusCodeType validates that a provided status code type is one of those used in the Stripe API
func StatusCodeType(code string) error {
	num, err := strconv.Atoi(code)
	if err != nil {
		return err
	}

	if num != 200 && num != 400 && num != 500 {
		return fmt.Errorf("Provided status code type %s is not a valid type (200, 400, 500)", code)
	}

	return nil
}
