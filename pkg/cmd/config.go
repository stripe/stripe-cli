package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

type configCmd struct {
	cmd    *cobra.Command
	config *config.Config

	list          bool
	edit          bool
	unset         string
	set           bool
	removeProfile string
	autoConfirm   bool
}

func newConfigCmd() *configCmd {
	cc := &configCmd{
		config: &Config,
	}
	cc.cmd = &cobra.Command{
		Use:   "config",
		Short: "Manually change the config values for the CLI",
		Long: `config lets you set and unset specific configuration values for your profile if
you need more granular control over the configuration.`,
		Example: `stripe config --list
  stripe config --set color off
  stripe config --unset color
  stripe config --remove-profile my-project`,
		RunE: cc.runConfigCmd,
	}

	cc.cmd.Flags().BoolVar(&cc.list, "list", false, "List configs")
	cc.cmd.Flags().BoolVarP(&cc.edit, "edit", "e", false, "Open an editor to the config file")
	cc.cmd.Flags().StringVar(&cc.unset, "unset", "", "Unset a specific config field")
	cc.cmd.Flags().BoolVar(&cc.set, "set", false, "Set a config field to some value")
	cc.cmd.Flags().StringVar(&cc.removeProfile, "remove-profile", "", "Remove a profile and its stored credentials from the config file")
	cc.cmd.Flags().BoolVarP(&cc.autoConfirm, "confirm", "c", false, "Skip the confirmation prompt for --remove-profile")

	cc.cmd.Flags().SetInterspersed(false) // allow args to happen after flags to enable 2 arguments to --set

	return cc
}

func (cc *configCmd) runConfigCmd(cmd *cobra.Command, args []string) error {
	switch ok := true; ok {
	case cc.set && len(args) == 2:
		return cc.config.Profile.WriteConfigField(args[0], args[1])
	case cc.unset != "":
		return cc.config.Profile.DeleteConfigField(cc.unset)
	case cc.removeProfile != "":
		return cc.runRemoveProfile(cmd)
	case cc.list:
		return cc.config.PrintConfig()
	case cc.edit:
		return cc.config.EditConfig()
	default:
		// no flags set or unrecognized flags/args
		return cc.cmd.Help()
	}
}

func (cc *configCmd) runRemoveProfile(cmd *cobra.Command) error {
	profileName := cc.removeProfile

	confirmed, err := cc.confirmRemoveProfile(cmd, profileName)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(cmd.OutOrStdout(), "Aborted. No changes were made.")
		return nil
	}

	if err := cc.config.RemoveProfile(profileName); err != nil {
		if errors.Is(err, config.ErrProfileNotFound) {
			return errorcategory.UserInputErrorf("no profile named %q was found in %s", profileName, cc.config.ProfilesFile)
		}
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Removed profile %q and cleared its stored credentials.\n", profileName)

	return nil
}

// confirmRemoveProfile asks before removing, because a profile can hold an API
// key the user pasted in by hand rather than one `stripe login` can reissue.
// Non-interactive runs must pass --confirm rather than silently proceeding.
func (cc *configCmd) confirmRemoveProfile(cmd *cobra.Command, profileName string) (bool, error) {
	if cc.autoConfirm {
		return true, nil
	}

	if !cc.isInteractive(cmd) {
		return false, errorcategory.UserInputErrorf("refusing to remove profile %q without confirmation; re-run with --confirm", profileName)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "This removes the profile %q and its stored credentials from %s.\nEnter 'yes' to confirm: ", profileName, cc.config.ProfilesFile)

	input, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil {
		return false, err
	}

	return strings.ToLower(strings.Trim(input, " \r\n")) == "yes", nil
}

// isInteractive reports whether there is a user on the other end to answer the
// prompt. The terminal check applies to os.Stdin only: when a caller has
// substituted the command's input, that input is the thing being read, so it is
// what decides, not the process's real stdin.
func (cc *configCmd) isInteractive(cmd *cobra.Command) bool {
	if cmd.InOrStdin() != os.Stdin {
		return true
	}

	return term.IsTerminal(int(os.Stdin.Fd()))
}
