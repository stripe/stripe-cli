package helpers

import "github.com/stripe/stripe-cli/pkg/coop"

func NextPendingNodeInStep(session *coop.Session, stepNumber, afterNode int) int {
	nodeNumber := 0
	for i := range session.Steps {
		for j := range session.Steps[i].Nodes {
			nodeNumber++
			if i+1 == stepNumber && nodeNumber > afterNode && session.Steps[i].Nodes[j].State == coop.NodePending {
				return nodeNumber
			}
		}
	}
	return 0
}
