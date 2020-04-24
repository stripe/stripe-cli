package logout

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/config"
)

// All function is used to clear the credentials on all profiles
func All(cfg *config.Config) error {
	fmt.Println("Logging out...")

	err := cfg.ClearAllCredentials()
	if err != nil {
		return err
	}

	fmt.Println("Credentials have been cleared for all projects.")

	return nil
}
