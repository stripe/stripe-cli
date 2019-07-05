package validators

import (
	"errors"
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
