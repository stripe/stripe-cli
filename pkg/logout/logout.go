package logout

import (
	"fmt"
	"io"

	"github.com/stripe/stripe-cli/pkg/config"
)

// Logout function is used to clear the credentials set for the current Profile
func Logout(config *config.Config, input io.Reader) error {
	liveKey, _ := config.Profile.GetAPIKey(true)
	testKey, _ := config.Profile.GetAPIKey(false)

	if liveKey == "" && testKey == "" {
		fmt.Println("You are already logged out.")
		return nil
	}

	fmt.Println("Logging out...")

	err := config.Profile.ClearKeys()
	if err != nil {
		return err
	}

	profileName := config.Profile.ProfileName

	if profileName == "default" {
		fmt.Println("Credentials have been cleared for the default project.")
	} else {
		fmt.Printf("Credentials have been cleared for %s.\n", profileName)
	}

	return nil
}
