package coop

// NewStepInfo returns the agent-facing blueprint contract for a session node.
func NewStepInfo(number int, node SessionNode) StepInfo {
	return StepInfo{
		Number:        number,
		Key:           node.Key,
		Title:         node.Title,
		Type:          node.Type,
		Description:   node.Description,
		ReviewPrompt:  node.ReviewPrompt,
		ReviewCommand: node.ReviewCommand,
		AutoConfirm:   node.AutoConfirm,
		APIRequest:    node.Request,
		TestRequests:  append([]TestHelperRequest(nil), node.TestRequests...),
		Events:        append([]string(nil), node.Events...),
		Semantics:     node.Semantics,
	}
}

// StepInfoByNumber returns the agent-facing blueprint contract for a 1-based node.
func (s *Session) StepInfoByNumber(n int) (*StepInfo, error) {
	step, _, nodeIndex, err := s.StepByNodeNumber(n)
	if err != nil {
		return nil, err
	}
	info := NewStepInfo(n, step.Nodes[nodeIndex])
	return &info, nil
}
