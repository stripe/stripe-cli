package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/autoupdate"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type autoUpdateCmd struct {
	cmd *cobra.Command

	enable  bool
	disable bool
	now     bool
}

func newAutoUpdateCmd() *autoUpdateCmd {
	au := &autoUpdateCmd{}

	au.cmd = &cobra.Command{
		Use:   "auto-update",
		Short: "Manage the Stripe CLI's background self-update",
		Long: fmt.Sprintf(`Manage the background self-update of the Stripe CLI.

A stripe binary installed by the install script keeps itself current: at most once
a day, a detached background process checks for a newer release and, if there is
one, downloads it, verifies its checksum, and replaces the binary in place. The
command you ran is never blocked on that work.

This does not apply to a stripe installed by Homebrew, apt, Scoop, WinGet, or npm.
Those installs belong to the package manager, and the CLI leaves them alone.

Auto-update is on by default. Turn it off with --disable, or per-invocation by
setting %s=1.`, autoupdate.EnvDisable),
		Example: `stripe auto-update
  stripe auto-update --disable
  stripe auto-update --now`,
		Args:   validators.NoArgs,
		RunE:   au.run,
		Hidden: true,
	}

	au.cmd.Flags().BoolVar(&au.enable, "enable", false, "Allow the CLI to update itself in the background")
	au.cmd.Flags().BoolVar(&au.disable, "disable", false, "Stop the CLI from updating itself")
	au.cmd.Flags().BoolVar(&au.now, "now", false, "Update to the latest release right now, in the foreground")
	au.cmd.MarkFlagsMutuallyExclusive("enable", "disable")

	return au
}

func (au *autoUpdateCmd) run(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	if au.enable || au.disable {
		if err := Config.WriteConfigField(config.AutoUpdateSettingKey(), au.enable); err != nil {
			return err
		}

		if au.enable {
			fmt.Fprintln(out, "Auto-update enabled.")
		} else {
			fmt.Fprintln(out, "Auto-update disabled.")
		}
	}

	if au.now {
		return autoupdate.Run(cmd.Context(), &Config, out)
	}

	if !au.enable && !au.disable {
		printAutoUpdateStatus(out)
	}

	return nil
}

func printAutoUpdateStatus(out io.Writer) {
	exe, managed := autoupdate.SelfManaged()

	switch {
	case !managed:
		fmt.Fprintf(out, "Auto-update is unavailable: %s is not an install-script install.\n", describeBinary(exe))
		fmt.Fprintf(out, "Upgrade it with whatever installed it, or reinstall into %s.\n", autoupdate.InstallDir())
	case autoupdate.IsDevBuild():
		fmt.Fprintln(out, "Auto-update is unavailable: this is a development build, which has no matching release.")
	case autoupdate.Enabled():
		fmt.Fprintln(out, "Auto-update is enabled.")
		fmt.Fprintf(out, "Turn it off with `stripe auto-update --disable`, or set %s=1.\n", autoupdate.EnvDisable)
	default:
		fmt.Fprintln(out, "Auto-update is disabled.")
		fmt.Fprintln(out, "Turn it on with `stripe auto-update --enable`.")
	}
}

func describeBinary(exe string) string {
	if exe == "" {
		return "this binary"
	}

	return exe
}
