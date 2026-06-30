package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/agentsetup"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type agentCmd struct {
	cmd *cobra.Command
}

type agentSetupCmd struct {
	cmd *cobra.Command

	statusOnly bool
	yes        bool
	force      bool
	jsonOutput bool
	client     string

	scanner    agentsetup.Scanner
	runInstall agentsetup.RunCommandFunc
}

type agentSetupJSON struct {
	agentsetup.ClaudeStatus
	Action  string   `json:"action"`
	Command []string `json:"command,omitempty"`
}

func newAgentCmd() *agentCmd {
	ac := &agentCmd{}
	ac.cmd = &cobra.Command{
		Use:   "agent",
		Short: "Set up AI agent tooling",
		Long:  "Set up AI agent tooling for Stripe development.",
		Args:  validators.NoArgs,
	}
	ac.cmd.AddCommand(newAgentSetupCmd().cmd)
	return ac
}

func newAgentSetupCmd() *agentSetupCmd {
	asc := &agentSetupCmd{
		client:     agentsetup.ClientClaudeCode,
		scanner:    agentsetup.DefaultScanner(),
		runInstall: agentsetup.RunCommand,
	}

	asc.cmd = &cobra.Command{
		Use:   "setup",
		Short: "Install or check Stripe agent tooling",
		Long:  "Install or check Stripe agent tooling. This POC supports Claude Code only.",
		Args:  validators.NoArgs,
		RunE:  asc.runSetup,
	}
	asc.cmd.Flags().BoolVar(&asc.statusOnly, "status", false, "Check installed agent tooling without making changes")
	asc.cmd.Flags().BoolVarP(&asc.yes, "yes", "y", false, "Skip confirmation prompts")
	asc.cmd.Flags().BoolVar(&asc.force, "force", false, "Reinstall even when agent tooling is already installed")
	asc.cmd.Flags().StringVar(&asc.client, "client", agentsetup.ClientClaudeCode, "Agent client to set up")
	asc.cmd.Flags().BoolVar(&asc.jsonOutput, "json", false, "Write machine-readable status output")

	return asc
}

func (asc *agentSetupCmd) runSetup(cmd *cobra.Command, args []string) error {
	if asc.client != agentsetup.ClientClaudeCode {
		return fmt.Errorf("unsupported agent client %q; this POC only supports %q", asc.client, agentsetup.ClientClaudeCode)
	}

	status := asc.scanner.ScanClaude()
	action, command := setupAction(status, asc.force)

	if asc.jsonOutput {
		return asc.writeJSON(cmd.OutOrStdout(), status, action, command)
	}

	printClaudeStatus(cmd.OutOrStdout(), status)

	if status.Status == agentsetup.StatusError {
		return errors.New(status.Error)
	}
	if asc.statusOnly {
		return nil
	}
	if !status.Detected {
		fmt.Fprintln(cmd.OutOrStdout(), "Nothing to do. Claude Code was not detected.")
		return nil
	}
	if action == "none" {
		fmt.Fprintln(cmd.OutOrStdout(), "Nothing to do. Stripe agent tooling is already installed.")
		return nil
	}

	name, installArgs := agentsetup.ClaudeInstallCommand()
	fmt.Fprintf(cmd.OutOrStdout(), "Planned command: %s %s\n", name, strings.Join(installArgs, " "))

	if !asc.yes {
		if !isInteractiveTerminal() {
			return fmt.Errorf("installation requires confirmation; rerun with --yes to install without prompting")
		}
		confirmed, err := confirmAgentSetup(cmd.InOrStdin(), cmd.OutOrStdout())
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Canceled. No changes made.")
			return nil
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Installing Stripe agent tooling for Claude Code...")
	if err := asc.runClaudeInstall(commandContextOrBackground(cmd), cmd.OutOrStdout(), name, installArgs, command); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Installed Stripe agent tooling for Claude Code.")
	return nil
}

func (asc *agentSetupCmd) runClaudeInstall(ctx context.Context, out io.Writer, name string, installArgs []string, command []string) error {
	if err := asc.runInstall(ctx, name, installArgs...); err == nil {
		return nil
	} else {
		updateName, updateArgs := agentsetup.ClaudeMarketplaceUpdateCommand()
		updateCommand := append([]string{updateName}, updateArgs...)
		fmt.Fprintf(out, "Install failed. Updating Claude plugin marketplace and retrying: %s\n", strings.Join(updateCommand, " "))
		if updateErr := asc.runInstall(ctx, updateName, updateArgs...); updateErr != nil {
			return fmt.Errorf("running %q after install failed: %w", strings.Join(updateCommand, " "), updateErr)
		}
		if retryErr := asc.runInstall(ctx, name, installArgs...); retryErr != nil {
			return fmt.Errorf("running %q after marketplace update: %w", strings.Join(command, " "), retryErr)
		}
		return nil
	}
}

func setupAction(status agentsetup.ClaudeStatus, force bool) (string, []string) {
	name, args := agentsetup.ClaudeInstallCommand()
	command := append([]string{name}, args...)

	switch {
	case status.Status == agentsetup.StatusError:
		return "error", nil
	case !status.Detected:
		return "none", nil
	case status.PluginInstalled && force:
		return "reinstall", command
	case status.PluginInstalled:
		return "none", nil
	default:
		return "install", command
	}
}

func (asc *agentSetupCmd) writeJSON(w io.Writer, status agentsetup.ClaudeStatus, action string, command []string) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(agentSetupJSON{
		ClaudeStatus: status,
		Action:       action,
		Command:      command,
	})
}

func printClaudeStatus(w io.Writer, status agentsetup.ClaudeStatus) {
	fmt.Fprintln(w, "Scanning for AI coding clients...")
	if !status.Detected {
		fmt.Fprintln(w, "  Claude Code: not detected")
		return
	}

	if status.ExecutablePath != "" {
		fmt.Fprintf(w, "  Claude Code: detected (%s)\n", status.ExecutablePath)
	} else {
		fmt.Fprintln(w, "  Claude Code: detected")
	}

	switch status.Status {
	case agentsetup.StatusInstalled:
		version := status.PluginVersion
		if version == "" {
			version = "unknown version"
		}
		fmt.Fprintf(w, "  Stripe plugin: installed as %s (%s", status.PluginID, version)
		if status.PluginScope != "" {
			fmt.Fprintf(w, ", %s", status.PluginScope)
		}
		if status.PluginScope == "local" && status.PluginProject != "" {
			fmt.Fprintf(w, ", project %s", status.PluginProject)
		}
		fmt.Fprintln(w, ")")
	case agentsetup.StatusError:
		fmt.Fprintf(w, "  Stripe plugin: error (%s)\n", status.Error)
	default:
		fmt.Fprintln(w, "  Stripe plugin: missing")
	}
}

func confirmAgentSetup(r io.Reader, w io.Writer) (bool, error) {
	fmt.Fprint(w, "Install now? [y/N] ")
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
