package cmd

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/agentsetup"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const (
	eventAgentSetupDetected       = "Agent Setup Detected"
	eventAgentSetupTUI            = "Agent Setup TUI"
	eventAgentSetupSelected       = "Agent Setup Selected"
	eventAgentSetupClientResult   = "Agent Setup Client Result"
	eventAgentSetupSkillInstalled = "Agent Setup Skill Installed"
	eventAgentSetupSkillsResult   = "Agent Setup Skills Result"

	agentSetupTUIConfirmed         = "confirmed"
	agentSetupTUISelectionCanceled = "selection_canceled"
	agentSetupTUIScopeCanceled     = "scope_canceled"
	agentSetupTUINoSelection       = "no_selection"

	agentSetupResultSuccess = "success"
	agentSetupResultFailed  = "failed"
	agentSetupResultSkipped = "skipped"
	agentSetupResultManual  = "manual"
)

func trackAgentSetupEvent(ctx context.Context, eventName string, eventValue string) {
	telemetryClient := stripe.GetTelemetryClient(ctx)
	if telemetryClient == nil {
		return
	}

	go telemetryClient.SendEvent(ctx, eventName, eventValue)
}

func emitAgentSetupDetectedEvents(ctx context.Context, statuses []agentsetup.Status) {
	detected := detectedStatuses(statuses)
	if len(detected) == 0 {
		trackAgentSetupEvent(ctx, eventAgentSetupDetected, "none")
		return
	}

	for _, status := range detected {
		trackAgentSetupEvent(ctx, eventAgentSetupDetected, status.Client)
	}
}

func emitAgentSetupTUIEvent(ctx context.Context, result string) {
	trackAgentSetupEvent(ctx, eventAgentSetupTUI, result)
}

func emitAgentSetupConfirmedSelection(ctx context.Context, sel Selection, scope string) {
	emitAgentSetupTUIEvent(ctx, agentSetupTUIConfirmed)

	for _, status := range sel.Agents {
		trackAgentSetupEvent(ctx, eventAgentSetupSelected, "client:"+status.Client)
	}

	if sel.InstallSkills {
		trackAgentSetupEvent(ctx, eventAgentSetupSelected, "skills:"+scope)
	}
}

func emitAgentSetupClientResult(ctx context.Context, client string, result string) {
	trackAgentSetupEvent(ctx, eventAgentSetupClientResult, client+":"+result)
}

func emitAgentSetupSkillsInstalled(ctx context.Context, scope string, skills []string) {
	for _, skill := range skills {
		trackAgentSetupEvent(ctx, eventAgentSetupSkillInstalled, scope+":"+skill)
	}
}

func emitAgentSetupSkillsFailure(ctx context.Context, scope string) {
	trackAgentSetupEvent(ctx, eventAgentSetupSkillsResult, scope+":"+agentSetupResultFailed)
}
