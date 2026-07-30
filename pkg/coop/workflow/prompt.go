package workflow

import (
	"fmt"
	"strings"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func nodeAgentPrompt(session *coop.Session, node *coop.SessionNode, nodeNumber int, requiredOutputs []coop.RequiredOutput) string {
	stepTitle := ""
	if step, _, _, err := session.StepByNodeNumber(nodeNumber); err == nil {
		stepTitle = step.TitleText()
	}

	var prompt strings.Builder
	fmt.Fprintf(&prompt, "Current node %d of %d", nodeNumber, session.TotalNodes())
	if stepTitle != "" {
		fmt.Fprintf(&prompt, " in step %q", stepTitle)
	}
	fmt.Fprintf(&prompt, ": %s\n\n", node.TitleText())

	if node.NodeType != "" {
		fmt.Fprintf(&prompt, "Node type: %s\n\n", node.NodeType)
	}
	if node.DescriptionText() != "" {
		fmt.Fprintf(&prompt, "Task (source of truth): %s\n\n", node.DescriptionText())
	}
	if guidance := nodeTypeGuidance(node); guidance != "" {
		fmt.Fprintf(&prompt, "How to approach it: %s\n\n", guidance)
	}
	if node.ReviewPrompt != "" {
		fmt.Fprintf(&prompt, "Acceptance check: %s\n", node.ReviewPrompt)
	}
	if node.ReviewCommand != "" {
		fmt.Fprintf(&prompt, "Verification command: run %q exactly, or explain concretely why it does not apply.\n", node.ReviewCommand)
	}
	if node.IsInformationalNode {
		prompt.WriteString("This node is auto-confirmed, so continue immediately after reporting the work.\n")
	}
	if len(requiredOutputs) > 0 {
		prompt.WriteString("\nRecord these outputs with report-work because later nodes reference them:\n")
		for _, output := range requiredOutputs {
			fmt.Fprintf(&prompt, "- %s\n", output.Selector())
		}
	}

	prompt.WriteString("\nWork only on this node. Inspect the existing project before changing it, implement a working result, and verify it. Make your report-work note and report-check evidence directly address the task")
	if node.ReviewPrompt != "" {
		prompt.WriteString(" and acceptance check")
	}
	prompt.WriteString(". Give the developer concrete actions and expected results for any manual verification.")
	fmt.Fprintf(&prompt, "\n\nBefore running the next command, report each passed verification with:\nstripe coop agent report-check --session=%s --step=%d --check=\"<what you verified>\" --passed", session.ID, nodeNumber)
	fmt.Fprintf(&prompt, "\nIf this node does not apply, use:\nstripe coop agent skip --session=%s --step=%d --note=\"<reason>\"", session.ID, nodeNumber)
	return prompt.String()
}

func nodeTypeGuidance(node *coop.SessionNode) string {
	if node.Key == "scan-project" {
		return "Read the project files, identify the stack and existing Stripe integration points, and summarize what you find. Do not ask the developer questions you can answer from the code."
	}

	switch node.NodeType {
	case coop.NodeAPIRequest:
		return "Use api_request and, when present, sdk_example as the technical starting point. Based on the task and existing project, decide whether the call belongs in runtime application code, a setup or seed script, or one-time provisioning; do not add a runtime endpoint for one-time provisioning unless the task requires one. Run and verify the call, and reuse returned IDs where later work needs them."
	case coop.NodeAsyncHandler:
		return `Implement and run the webhook handler. Test it with "stripe listen --forward-to localhost:<port>/webhook" and verify Stripe signatures before acting on events.`
	case coop.NodeUIComponent:
		return "Build the user-facing flow and exercise it in the running application."
	case coop.NodeCLICommand:
		return "Run the requested CLI operation and report its concrete result."
	case coop.NodeTestHelper:
		return "Verify the integration end to end. Any test_requests advance Stripe test state and should be run as test setup, not implemented in the application."
	case coop.NodeDashboard:
		return "Complete the requested Stripe Dashboard configuration and verify the resulting state."
	case coop.NodeSetUpWebhooks:
		return "Configure the requested webhook destination and verify that the application receives the listed events."
	default:
		return ""
	}
}

func language(session *coop.Session) string {
	if session != nil && session.Settings != nil && session.Settings["language"] != "" {
		return session.Settings["language"]
	}
	return "node"
}
