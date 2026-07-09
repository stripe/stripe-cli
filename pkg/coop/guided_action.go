package coop

import "time"

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
