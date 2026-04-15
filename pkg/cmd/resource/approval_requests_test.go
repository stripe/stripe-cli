package resource

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestAddApprovalRequestsSubCmds(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	v2Cmd := &cobra.Command{Use: "v2"}
	coreCmd := &cobra.Command{Use: "core", Annotations: make(map[string]string)}
	v2Cmd.AddCommand(coreCmd)
	root.AddCommand(v2Cmd)

	AddApprovalRequestsSubCmds(root, &config.Config{})

	var found *cobra.Command
	for _, cmd := range coreCmd.Commands() {
		if cmd.Use == "approval_requests" {
			found = cmd
			break
		}
	}
	require.NotNil(t, found, "expected approval_requests command under v2 core")
}
