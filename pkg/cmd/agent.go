package cmd

import (
	"bufio"
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

	providers map[string]agentsetup.Provider
}

type agentSetupJSON struct {
	Status  string              `json:"status"`
	Clients []agentsetup.Status `json:"clients"`
	Actions []agentsetup.Plan   `json:"actions,omitempty"`
	Errors  []string            `json:"errors,omitempty"`
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
		client:    agentsetup.ClientClaudeCode,
		providers: agentsetup.DefaultProviders(),
	}

	asc.cmd = &cobra.Command{
		Use:           "setup",
		Short:         "Install or check Stripe agent tooling",
		Long:          "Install or check Stripe agent tooling. This POC supports Claude Code only.",
		Args:          validators.NoArgs,
		RunE:          asc.runSetup,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	asc.cmd.Flags().BoolVar(&asc.statusOnly, "status", false, "Check installed agent tooling without making changes")
	asc.cmd.Flags().BoolVarP(&asc.yes, "yes", "y", false, "Skip confirmation prompts")
	asc.cmd.Flags().BoolVar(&asc.force, "force", false, "Reinstall even when agent tooling is already installed")
	asc.cmd.Flags().StringVar(&asc.client, "client", agentsetup.ClientClaudeCode, "Agent client to set up")
	asc.cmd.Flags().BoolVar(&asc.jsonOutput, "json", false, "Write machine-readable status output")

	return asc
}

func (asc *agentSetupCmd) runSetup(cmd *cobra.Command, args []string) error {
	provider, ok := asc.providers[asc.client]
	if !ok {
		return fmt.Errorf("unsupported agent client %q; supported clients: %s", asc.client, agentsetup.SupportedProviderIDs(asc.providers))
	}

	status := provider.Detect()
	plan := provider.Plan(status, asc.force)

	if asc.jsonOutput {
		if err := asc.writeJSON(cmd.OutOrStdout(), status, plan); err != nil {
			return err
		}
		if status.Status == agentsetup.StatusError {
			return errors.New(status.Error)
		}
		return nil
	}

	printAgentStatus(cmd.OutOrStdout(), status)

	if status.Status == agentsetup.StatusError {
		return errors.New(status.Error)
	}
	if asc.statusOnly {
		return nil
	}
	if !status.Detected {
		fmt.Fprintf(cmd.OutOrStdout(), "Nothing to do. %s was not detected.\n", status.DisplayName)
		return nil
	}
	if plan.Action == agentsetup.ActionNone {
		fmt.Fprintln(cmd.OutOrStdout(), "Nothing to do. Stripe agent tooling is already installed.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Planned command: %s\n", strings.Join(plan.Command, " "))

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

	fmt.Fprintf(cmd.OutOrStdout(), "Installing Stripe agent tooling for %s...\n", status.DisplayName)
	if err := provider.Apply(commandContextOrBackground(cmd), cmd.OutOrStdout(), plan); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Installed Stripe agent tooling for %s.\n", status.DisplayName)
	return nil
}

func (asc *agentSetupCmd) writeJSON(w io.Writer, status agentsetup.Status, plan agentsetup.Plan) error {
	result := agentSetupJSON{
		Status:  status.Status,
		Clients: []agentsetup.Status{status},
	}
	if plan.Action != agentsetup.ActionNone {
		result.Actions = []agentsetup.Plan{plan}
	}
	if status.Status == agentsetup.StatusError {
		result.Errors = []string{status.Error}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func printAgentStatus(w io.Writer, status agentsetup.Status) {
	fmt.Fprintln(w, "Scanning for AI coding clients...")
	if !status.Detected {
		fmt.Fprintf(w, "  %s: not detected\n", status.DisplayName)
		return
	}

	if status.ExecutablePath != "" {
		fmt.Fprintf(w, "  %s: detected (%s)\n", status.DisplayName, status.ExecutablePath)
	} else {
		fmt.Fprintf(w, "  %s: detected\n", status.DisplayName)
	}

	switch status.Status {
	case agentsetup.StatusInstalled:
		version := status.Plugin.Version
		if version == "" {
			version = "unknown version"
		}
		fmt.Fprintf(w, "  Stripe plugin: installed as %s (%s", status.Plugin.ID, version)
		if status.Plugin.Scope != "" {
			fmt.Fprintf(w, ", %s", status.Plugin.Scope)
		}
		if status.Plugin.Scope == "local" && status.Plugin.Project != "" {
			fmt.Fprintf(w, ", project %s", status.Plugin.Project)
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
