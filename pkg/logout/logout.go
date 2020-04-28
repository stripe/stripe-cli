package logout

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/config"
)

// Logout function is used to clear the credentials set for the current Profile
func Logout(config *config.Config) error {
	liveKey, _ := config.Profile.GetAPIKey(true)
	testKey, _ := config.Profile.GetAPIKey(false)

	if liveKey == "" && testKey == "" {
		fmt.Println("You are already logged out.")
		return nil
	}

	fmt.Println("Logging out...")

	profileName := config.Profile.ProfileName

	err := config.RemoveProfile(profileName)
	if err != nil {
		return err
	}

	if profileName == "default" {
		fmt.Println("Credentials have been cleared for the default project.")
	} else {
		fmt.Printf("Credentials have been cleared for %s.\n", profileName)
	}

	return nil
}

// All function is used to clear the credentials on all profiles
func All(cfg *config.Config) error {
	fmt.Println("Logging out...")

	err := cfg.RemoveAllProfiles()
	if err != nil {
		return err
	}

	fmt.Println("Credentials have been cleared for all projects.")

	return nil
}
