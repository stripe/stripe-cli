package coop

import (
	"fmt"
	"time"
)

const (
	GuidedActionDeploy       = "deploy"
	GuidedActionDeployUpdate = "deploy-update"
)

type GuidedAction struct {
	ID           string
	Title        string
	AgentContext string
	Steps        []SessionStep
}

type GuidedActionSessionOptions struct {
	ParentSessionID string
	ParentStepID    string
	Settings        map[string]string
	UsedSandbox     bool
}

func GuidedActionByID(id, target string) (*GuidedAction, error) {
	switch id {
	case GuidedActionDeploy:
		return deployGuidedAction(), nil
	case GuidedActionDeployUpdate:
		return deployUpdateGuidedAction(target), nil
	default:
		return nil, fmt.Errorf("guided action %q not found", id)
	}
}

func NewSessionFromGuidedAction(action *GuidedAction, sessionID string, opts GuidedActionSessionOptions) *Session {
	now := time.Now().UTC()
	settings := copyStringMap(opts.Settings)
	settings["guided_action"] = action.ID

	return &Session{
		SchemaVersion:   CurrentSessionSchemaVersion,
		ID:              sessionID,
		Blueprint:       action.Title,
		Status:          SessionActive,
		Settings:        settings,
		Steps:           cloneGuidedActionSteps(action.Steps),
		UsedSandbox:     opts.UsedSandbox,
		ParentSessionID: opts.ParentSessionID,
		ParentStepID:    opts.ParentStepID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func deployGuidedAction() *GuidedAction {
	return &GuidedAction{
		ID:           GuidedActionDeploy,
		Title:        "Deploy with Stripe Projects",
		AgentContext: `Use the Stripe Projects CLI plugin's own detection and commands. Do not supplement or modify co-op blueprints as the deployment source of truth. Do not print secret key material; "stripe whoami --format json" is safe for account context because it does not print key material.`,
		Steps: []SessionStep{
			{
				StepDefinition: StepDefinition{Key: "detect-deploy-path", Title: "Detect deploy path"},
				Nodes: []SessionNode{
					guidedActionNode(NodeCLICommand, "check-projects-plugin", "Check Stripe Projects plugin", `Run "stripe projects --help". If the command is unavailable, install it with "stripe plugin install projects" and run the help command again. Use the plugin's detection/help output to identify the project type and available deployment commands.`, "Confirm the Stripe Projects plugin is available and the agent reported the detected project and deploy path."),
				},
			},
			{
				StepDefinition: StepDefinition{Key: "configure-deploy", Title: "Configure deployment"},
				Nodes: []SessionNode{
					guidedActionNode(NodeCLICommand, "configure-projects-deploy", "Configure Stripe Projects deployment", "Use Stripe Projects to initialize or update deployment configuration for this repository. Configure required Stripe environment variable names, but do not print secret values. Report the files or settings changed.", "Confirm deployment configuration was created or updated and the agent identified the required Stripe environment variables without exposing secret values."),
				},
			},
			{
				StepDefinition: StepDefinition{Key: "deploy", Title: "Deploy"},
				Nodes: []SessionNode{
					guidedActionNode(NodeCLICommand, "run-projects-deploy", "Run Stripe Projects deploy", "Run the Stripe Projects deployment command selected by the plugin for this project. Capture the deployment URL, build output, and any follow-up setup required by the provider.", "Confirm the deploy command completed and the agent reported the deployed URL or the exact blocker."),
				},
			},
			{
				StepDefinition: StepDefinition{Key: "verify-deployment", Title: "Verify deployment"},
				Nodes: []SessionNode{
					guidedActionNode(NodeTestHelper, "verify-deployed-integration", "Verify deployed integration", "Exercise the deployed URL or provider preview. Confirm the Stripe integration can complete its happy path and that webhooks or callbacks point at the deployed environment when applicable.", "Confirm the deployed integration was exercised successfully or the agent documented the remaining deploy-time blocker."),
				},
			},
		},
	}
}

func deployUpdateGuidedAction(target string) *GuidedAction {
	if target == "" {
		target = "the detected deployment target"
	}
	return &GuidedAction{
		ID:           GuidedActionDeployUpdate,
		Title:        "Deploy your changes",
		AgentContext: fmt.Sprintf(`Use the existing %s deployment configuration and push the new integration code to %s. Do not start a Stripe Projects co-op blueprint unless the developer explicitly asks to migrate deployment infrastructure. Do not print secret values.`, target, target),
		Steps: []SessionStep{
			{
				StepDefinition: StepDefinition{Key: "confirm-target", Title: "Confirm deployment target"},
				Nodes: []SessionNode{
					guidedActionNode(NodeTestHelper, "inspect-deploy-config", "Inspect existing deploy config", fmt.Sprintf("Inspect the repository's existing deployment files and provider metadata to confirm the project deploys through %s. Report the relevant config files and deployment command or CI path.", target), fmt.Sprintf("Confirm the agent identified %s as the deployment target and reported the deploy path it will use.", target)),
				},
			},
			{
				StepDefinition: StepDefinition{Key: "push-deploy", Title: "Push deployment"},
				Nodes: []SessionNode{
					guidedActionNode(NodeCLICommand, "push-integration-code", "Push integration code", fmt.Sprintf("Use the established %s workflow to deploy the new Stripe integration code. Preserve the existing provider setup and configure any required Stripe environment variables without printing secret values.", target), fmt.Sprintf("Confirm the agent pushed the integration changes to %s or reported the exact blocker.", target)),
				},
			},
			{
				StepDefinition: StepDefinition{Key: "verify-deployment", Title: "Verify deployment"},
				Nodes: []SessionNode{
					guidedActionNode(NodeTestHelper, "verify-deployed-update", "Verify deployed update", fmt.Sprintf("Exercise the deployed %s URL or preview. Confirm the new Stripe integration behavior works against the deployed environment and report any URL, logs, or provider checks used.", target), "Confirm the deployed integration update was exercised successfully or the agent documented the remaining deploy-time blocker."),
				},
			},
		},
	}
}

func guidedActionNode(nodeType NodeType, key, title, description, reviewPrompt string) SessionNode {
	return SessionNode{
		NodeDefinition: NodeDefinition{
			Type:         nodeType,
			Key:          key,
			Title:        title,
			Description:  description,
			ReviewPrompt: reviewPrompt,
		},
		State: NodePending,
	}
}

func cloneGuidedActionSteps(steps []SessionStep) []SessionStep {
	cloned := make([]SessionStep, len(steps))
	for i, step := range steps {
		cloned[i].StepDefinition = step.StepDefinition
		cloned[i].Nodes = make([]SessionNode, len(step.Nodes))
		copy(cloned[i].Nodes, step.Nodes)
	}
	return cloned
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
