package resource

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
)

var approvalRequestsSpecs = []OperationSpec{
	{
		Name:   "list",
		Path:   "/v2/core/approval_requests",
		Method: "GET",
		Params: map[string]*ParamSpec{
			"limit": {
				Type:             "integer",
				ShortDescription: "Maximum number of results to return",
			},
			"page": {
				Type:             "string",
				ShortDescription: "Cursor for the requested page",
			},
		},
	},
	{
		Name:   "retrieve",
		Path:   "/v2/core/approval_requests/{id}",
		Method: "GET",
	},
	{
		Name:   "submit",
		Path:   "/v2/core/approval_requests/{id}/submit",
		Method: "POST",
		Params: map[string]*ParamSpec{
			"reason": {
				Type:             "string",
				ShortDescription: "The reason for submitting the approval request",
			},
		},
	},
	{
		Name:   "cancel",
		Path:   "/v2/core/approval_requests/{id}/cancel",
		Method: "POST",
	},
	{
		Name:   "execute",
		Path:   "/v2/core/approval_requests/{id}/execute",
		Method: "POST",
	},
}

// AddApprovalRequestsSubCmds patches approval_requests commands into the
// auto-generated `core` namespace command tree.
func AddApprovalRequestsSubCmds(rootCmd *cobra.Command, cfg *config.Config) error {
	var v2Cmd *cobra.Command
	var coreCmd *cobra.Command

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "v2" {
			v2Cmd = cmd
			break
		}
	}

	if v2Cmd == nil {
		return nil
	}

	for _, cmd := range v2Cmd.Commands() {
		if cmd.Use == "core" {
			coreCmd = cmd
			break
		}
	}

	if coreCmd == nil {
		return nil
	}

	rApprovalRequestsCmd := NewResourceCmd(coreCmd, "approval_requests")
	rApprovalRequestsCmd.Cmd.Hidden = true

	for i := range approvalRequestsSpecs {
		NewOperationCmd(rApprovalRequestsCmd.Cmd, &approvalRequestsSpecs[i], cfg)
	}

	return nil
}
