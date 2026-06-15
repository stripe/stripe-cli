package helpers

import "github.com/stripe/stripe-cli/pkg/coop"

func NextPendingNodeInStep(session *coop.Session, stepIndex, afterNode int) int {
	nodeNumber := 0
	for i := range session.Steps {
		for j := range session.Steps[i].Nodes {
			nodeNumber++
			if i == stepIndex && nodeNumber > afterNode && session.Steps[i].Nodes[j].State == coop.NodePending {
				return nodeNumber
			}
		}
	}
	return 0
}

func StepReviewApplies(session *coop.Session, nodeNumber int) bool {
	step, _, _, err := session.StepByNodeNumber(nodeNumber)
	if err != nil {
		return false
	}
	switch step.ReviewGranularity {
	case coop.ReviewGranularityAuto, coop.ReviewGranularityNode:
		return false
	default:
		return true
	}
}
