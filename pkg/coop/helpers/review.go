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
	_, err := session.NodeByNumber(nodeNumber)
	return err == nil
}
