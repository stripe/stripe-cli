// Package coopcmd wires the co-op mode Cobra commands into the Stripe CLI.
package coopcmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
)

type Options struct {
	ConfigFolder             func() string
	ClaimURL                 func() string
	AIAgentHelpAnnotationKey string
}

const defaultAIAgentHelpAnnotationKey = "ai_agent_help"

var options Options

func New(opts Options) *cobra.Command {
	options = opts
	return newCoopCmd().cmd
}

type coopCmd struct {
	cmd *cobra.Command
}

func newCoopCmd() *coopCmd {
	cc := &coopCmd{}
	annotationKey := options.AIAgentHelpAnnotationKey
	if annotationKey == "" {
		annotationKey = defaultAIAgentHelpAnnotationKey
	}
	cc.cmd = &cobra.Command{
		Use:   "coop",
		Short: "Collaborative integration mode (AI agent + human developer)",
		Long: `Co-op mode enables AI agents and human developers to collaborate on Stripe
integrations in real time. The agent writes code and reports progress via CLI
commands; the developer watches live in a terminal UI.

Start a session with a blueprint, then let the agent work through it step by step.
The developer confirms each step before the agent moves on.`,
		Annotations: map[string]string{
			annotationKey: `  Workflow: start a session, then use typed agent commands to progress through it.
  1. stripe coop run <blueprint-id> — begin a session
  2. stripe coop agent start-work --session=<id> --step=<n> --note="..." — mark work active
  3. stripe coop agent report-check --session=<id> --step=<n> --check="..." --passed — add verification
  4. stripe coop agent report-work --session=<id> --step=<n> --file=... --note="..." — report work complete
  All commands output JSON with a "next" field suggesting the next command.
  Run "stripe coop recommend --query=..." to discover available blueprints.`,
		},
	}

	cc.cmd.AddCommand(newCoopRunCmd().cmd)
	cc.cmd.AddCommand(newCoopStartCmd().cmd)
	cc.cmd.AddCommand(newCoopJoinCmd().cmd)
	cc.cmd.AddCommand(newCoopAgentCmd().cmd)
	cc.cmd.AddCommand(newCoopStatusCmd().cmd)
	cc.cmd.AddCommand(newCoopStopCmd().cmd)
	cc.cmd.AddCommand(newCoopRecommendCmd().cmd)
	cc.cmd.AddCommand(newCoopDebugAgentCmd().cmd)

	return cc
}

func coopConfigFolder() string {
	if options.ConfigFolder == nil {
		var cfg config.Config
		return cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	}
	return options.ConfigFolder()
}

func coopClaimURL() string {
	if options.ClaimURL == nil {
		return ""
	}
	return options.ClaimURL()
}

func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic("marking required flag " + name + ": " + err.Error())
	}
}
