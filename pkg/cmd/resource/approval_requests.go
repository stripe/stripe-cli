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
func AddApprovalRequestsSubCmds(rootCmd *cobra.Command, cfg *config.Config) {
	coreCmd, _, err := rootCmd.Find([]string{"v2", "core"})
	if err != nil || coreCmd.Name() != "core" {
		// Fail open if "stripe v2 core" command is not found
		// This is best effort while the approval_requests API is still in private preview
		return
	}

	rApprovalRequestsCmd := NewResourceCmd(coreCmd, "approval_requests")
	rApprovalRequestsCmd.Cmd.Hidden = true

	for i := range approvalRequestsSpecs {
		NewOperationCmd(rApprovalRequestsCmd.Cmd, &approvalRequestsSpecs[i], cfg)
	}
}
