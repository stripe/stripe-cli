package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/spec"
)

//
// Private functions
//

func addAllResourceCmds(rootCmd *cobra.Command, stripeAPI *spec.Spec) {
	namespaceCmds := make(map[string]*resource.NamespaceCmd)

	for name, schema := range stripeAPI.Components.Schemas {
		if schema.XStripeOperations == nil {
			continue
		}

		namespaceName, resourceName := parseSchemaName(name)

		for _, op := range *schema.XStripeOperations {
			if op.MethodOn != "service" {
				continue
			}

			// Create the namespace command if it doesn't already exist
			if _, ok := namespaceCmds[namespaceName]; !ok {
				namespaceCmds[namespaceName] = resource.NewNamespaceCmd(rootCmd, namespaceName)
			}

			// Create the resource command if it doesn't already exist
			resourceCmdName := resource.GetResourceCmdName(resourceName)
			if _, ok := namespaceCmds[namespaceName].ResourceCmds[resourceCmdName]; !ok {
				var parentCmd *cobra.Command
				if namespaceName == "" {
					parentCmd = rootCmd
				} else {
					parentCmd = namespaceCmds[namespaceName].Cmd
				}
				namespaceCmds[namespaceName].ResourceCmds[resourceCmdName] = resource.NewResourceCmd(parentCmd, resourceCmdName)
			}

			// Create the operation command if it doesn't already exist
			if _, ok := namespaceCmds[namespaceName].ResourceCmds[resourceCmdName].OperationCmds[op.MethodName]; !ok {
				parentCmd := namespaceCmds[namespaceName].ResourceCmds[resourceCmdName].Cmd
				op2 := *stripeAPI.Paths[spec.Path(op.Path)][op.Operation]
				namespaceCmds[namespaceName].ResourceCmds[resourceCmdName].OperationCmds[op.MethodName] = resource.NewOperationCmd(parentCmd, op, op2)
			}
		}
	}
}

func parseSchemaName(name string) (string, string) {
	if strings.Contains(name, ".") {
		components := strings.SplitN(name, ".", 2)
		return components[0], components[1]
	}
	return "", name
}
