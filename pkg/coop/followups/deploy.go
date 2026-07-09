// Package followups defines built-in guided follow-up sessions for co-op.
package followups

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/coop"
)

const (
	Deploy       = "deploy"
	DeployUpdate = "deploy-update"
)

func GuidedActionByID(id, target string) (*coop.GuidedAction, error) {
	switch id {
	case Deploy:
		return DeployGuidedAction(), nil
	case DeployUpdate:
		return DeployUpdateGuidedAction(target), nil
	default:
		return nil, fmt.Errorf("guided action %q not found", id)
	}
}

func DeployGuidedAction() *coop.GuidedAction {
	return &coop.GuidedAction{
		ID:           Deploy,
		Title:        "Deploy with Stripe Projects",
		AgentContext: `Use the Stripe Projects CLI plugin's own detection and commands. Do not supplement or modify co-op blueprints as the deployment source of truth. Do not print secret key material; "stripe whoami --format json" is safe for account context because it does not print key material.`,
		Steps: []coop.SessionStep{
			{
				StepDefinition: coop.StepDefinition{Key: "detect-deploy-path", Title: "Detect deploy path"},
				Nodes: []coop.SessionNode{
					node(coop.NodeCLICommand, "check-projects-plugin", "Check Stripe Projects plugin", `Run "stripe projects --help". If the command is unavailable, install it with "stripe plugin install projects" and run the help command again. Use the plugin's detection/help output to identify the project type and available deployment commands.`, "Confirm the Stripe Projects plugin is available and the agent reported the detected project and deploy path."),
				},
			},
			{
				StepDefinition: coop.StepDefinition{Key: "configure-deploy", Title: "Configure deployment"},
				Nodes: []coop.SessionNode{
					node(coop.NodeCLICommand, "configure-projects-deploy", "Configure Stripe Projects deployment", "Use Stripe Projects to initialize or update deployment configuration for this repository. Configure required Stripe environment variable names, but do not print secret values. Report the files or settings changed.", "Confirm deployment configuration was created or updated and the agent identified the required Stripe environment variables without exposing secret values."),
				},
			},
			{
				StepDefinition: coop.StepDefinition{Key: "deploy", Title: "Deploy"},
				Nodes: []coop.SessionNode{
					node(coop.NodeCLICommand, "run-projects-deploy", "Run Stripe Projects deploy", "Run the Stripe Projects deployment command selected by the plugin for this project. Capture the deployment URL, build output, and any follow-up setup required by the provider.", "Confirm the deploy command completed and the agent reported the deployed URL or the exact blocker."),
				},
			},
			{
				StepDefinition: coop.StepDefinition{Key: "verify-deployment", Title: "Verify deployment"},
				Nodes: []coop.SessionNode{
					node(coop.NodeTestHelper, "verify-deployed-integration", "Verify deployed integration", "Exercise the deployed URL or provider preview. Confirm the Stripe integration can complete its happy path and that webhooks or callbacks point at the deployed environment when applicable.", "Confirm the deployed integration was exercised successfully or the agent documented the remaining deploy-time blocker."),
				},
			},
		},
	}
}

func DeployUpdateGuidedAction(target string) *coop.GuidedAction {
	if target == "" {
		target = "the detected deployment target"
	}
	return &coop.GuidedAction{
		ID:           DeployUpdate,
		Title:        "Deploy your changes",
		AgentContext: fmt.Sprintf(`Use the existing %s deployment configuration and push the new integration code to %s. Do not start a Stripe Projects co-op blueprint unless the developer explicitly asks to migrate deployment infrastructure. Do not print secret values.`, target, target),
		Steps: []coop.SessionStep{
			{
				StepDefinition: coop.StepDefinition{Key: "confirm-target", Title: "Confirm deployment target"},
				Nodes: []coop.SessionNode{
					node(coop.NodeTestHelper, "inspect-deploy-config", "Inspect existing deploy config", fmt.Sprintf("Inspect the repository's existing deployment files and provider metadata to confirm the project deploys through %s. Report the relevant config files and deployment command or CI path.", target), fmt.Sprintf("Confirm the agent identified %s as the deployment target and reported the deploy path it will use.", target)),
				},
			},
			{
				StepDefinition: coop.StepDefinition{Key: "push-deploy", Title: "Push deployment"},
				Nodes: []coop.SessionNode{
					node(coop.NodeCLICommand, "push-integration-code", "Push integration code", fmt.Sprintf("Use the established %s workflow to deploy the new Stripe integration code. Preserve the existing provider setup and configure any required Stripe environment variables without printing secret values.", target), fmt.Sprintf("Confirm the agent pushed the integration changes to %s or reported the exact blocker.", target)),
				},
			},
			{
				StepDefinition: coop.StepDefinition{Key: "verify-deployment", Title: "Verify deployment"},
				Nodes: []coop.SessionNode{
					node(coop.NodeTestHelper, "verify-deployed-update", "Verify deployed update", fmt.Sprintf("Exercise the deployed %s URL or preview. Confirm the new Stripe integration behavior works against the deployed environment and report any URL, logs, or provider checks used.", target), "Confirm the deployed integration update was exercised successfully or the agent documented the remaining deploy-time blocker."),
				},
			},
		},
	}
}

func node(nodeType coop.NodeType, key, title, description, reviewPrompt string) coop.SessionNode {
	return coop.SessionNode{
		NodeDefinition: coop.NodeDefinition{
			Type:         nodeType,
			Key:          key,
			Title:        title,
			Description:  description,
			ReviewPrompt: reviewPrompt,
		},
		State: coop.NodePending,
	}
}
