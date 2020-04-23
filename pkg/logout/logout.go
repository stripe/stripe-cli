package logout

import (
	"fmt"
	"io"

	"github.com/stripe/stripe-cli/pkg/config"
)

// Logout function is used to clear the credentials set for the current Profile
func Logout(config *config.Config, input io.Reader) error {
	fmt.Println("Logging out...")

	err := config.Profile.ClearKeys()
	if err != nil {
		return err
	}

	fmt.Println("You are now logged out!")

	return nil
}
