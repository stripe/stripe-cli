package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/stripe/stripe-cli/ansi"
	"github.com/stripe/stripe-cli/validators"

	log "github.com/sirupsen/logrus"
)

type configureCmd struct {
	cmd *cobra.Command
}

func newConfigureCmd() *configureCmd {
	cc := &configureCmd{}

	cc.cmd = &cobra.Command{
		Use:   "configure",
		Args:  validators.NoArgs,
		Short: "Configure the Stripe CLI",
		Long: `Add your Stripe test secret API Key to connect to Stripe.

By default, this will store the API key in the "default" namespace. You may
optionally provide a project name to store multiple API keys.

The configure command will also prompt for a device name to identify the
connected computer. This is used to show who is currently connected to the
webhooks tunnel through the Stripe Dashboard.

Run configuration:
$ stripe configure

Configure for a specific project:
$ stripe configure --project-name rocket_rides`,
		RunE: cc.runConfigureCmd,
	}

	return cc
}

func (cc *configureCmd) runConfigureCmd(cmd *cobra.Command, args []string) error {
	configPath := profile.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	dotfilePath := filepath.Join(configPath, "config.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err = os.MkdirAll(configPath, os.ModePerm)
		if err != nil {
			return err
		}
	}

	apiKey, err := cc.getConfigureAPIKey(os.Stdin)
	if err != nil {
		return err
	}

	deviceName := cc.getConfigureDeviceName(os.Stdin)

	log.WithFields(log.Fields{
		"prefix": "cmd.configureCmd.runConfigureCmd",
		"path":   dotfilePath,
	}).Debug("Writing config file")

	viper.SetConfigType("toml")
	viper.SetConfigFile(dotfilePath)

	viper.Set(profile.ProfileName+".secret_key", strings.TrimSpace(apiKey))
	viper.Set("default.device_name", strings.TrimSpace(deviceName))
	err = viper.WriteConfig()
	if err != nil {
		return err
	}

	fmt.Println("You're configured and all set to get started")

	return nil
}

func (cc *configureCmd) getConfigureAPIKey(input io.Reader) (string, error) {
	fmt.Print("Enter your test mode secret API key: ")
	apiKey, err := cc.securePrompt(input)
	if err != nil {
		return "", err
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", errors.New("API key is required, please provide your test mode secret API key")
	}
	err = validators.APIKey(apiKey)
	if err != nil {
		return "", err
	}

	fmt.Printf("Your API key is: %s\n", cc.redactAPIKey(apiKey))

	return apiKey, nil
}

func (cc *configureCmd) getConfigureDeviceName(input io.Reader) string {
	hostName, _ := os.Hostname()
	reader := bufio.NewReader(input)

	color := ansi.Color(os.Stdout)
	fmt.Printf("How would you like to identify this device in the Stripe Dashboard? [default: %s] ", color.Bold(color.Cyan(hostName)))

	deviceName, _ := reader.ReadString('\n')
	if strings.TrimSpace(deviceName) == "" {
		deviceName = hostName
	}

	return deviceName
}

// redactAPIKey returns a redacted version of API keys. The first 8 and last 4
// characters are not redacted, everything else is replaced by "*" characters.
//
// It panics if the provided string has less than 12 characters.
func (cc *configureCmd) redactAPIKey(apiKey string) string {
	var b strings.Builder

	b.WriteString(apiKey[0:8])                         // #nosec G104 (gosec bug: https://github.com/securego/gosec/issues/267)
	b.WriteString(strings.Repeat("*", len(apiKey)-12)) // #nosec G104 (gosec bug: https://github.com/securego/gosec/issues/267)
	b.WriteString(apiKey[len(apiKey)-4:])              // #nosec G104 (gosec bug: https://github.com/securego/gosec/issues/267)

	return b.String()
}

func (cc *configureCmd) securePrompt(input io.Reader) (string, error) {
	if input == os.Stdin {
		buf, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", err
		}
		fmt.Print("\n")
		return string(buf), nil
	}

	reader := bufio.NewReader(input)
	return reader.ReadString('\n')
}
