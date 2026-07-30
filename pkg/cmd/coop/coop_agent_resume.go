package coopcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func newCoopAgentResumeCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "resume",
		Short: "Return the exact command for the current Co-op lifecycle state",
		Args:  agentNoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(c.session) == "" {
				return outputCoopError(
					"--session flag is required",
					"Provide the Co-op session to resume.",
					coop.ResumeTemplate(),
				)
			}
			service, err := newWorkflowService()
			if err != nil {
				return outputAgentError(err)
			}
			resp, err := service.Resume(c.session)
			return outputAgentResponse(resp, err)
		},
	}
	c.cmd.Flags().StringVar(&c.session, "session", "", "Session ID")
	configureAgentCommand(c.cmd)
	return c
}
