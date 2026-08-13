package helpers

import (
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/coop"
)

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

// WaitedInReview reports how long a step has been waiting on the developer,
// taken from the latest CompletedAt across its nodes in review. TransitionNode
// stamps CompletedAt on entry to review, so this survives the process boundary
// and lets a fresh await call report a wait spanning earlier ones.
func WaitedInReview(session *coop.Session, stepIndex int, now time.Time) time.Duration {
	since := stepReviewSince(session, stepIndex)
	if since.IsZero() {
		return 0
	}
	if waited := now.Sub(since); waited > 0 {
		return waited
	}
	return 0
}

func stepReviewSince(session *coop.Session, stepIndex int) time.Time {
	var since time.Time
	forEachReviewNode(session, stepIndex, func(node *coop.SessionNode) {
		if node.CompletedAt == nil {
			return
		}
		if since.IsZero() || node.CompletedAt.After(since) {
			since = *node.CompletedAt
		}
	})
	return since
}

// StepReviewPrompt joins the review prompts of the nodes currently in review so
// a waiting response can restate what the developer is checking.
func StepReviewPrompt(session *coop.Session, stepIndex int) string {
	var prompts []string
	seen := make(map[string]bool)
	forEachReviewNode(session, stepIndex, func(node *coop.SessionNode) {
		if node.ReviewPrompt == "" || seen[node.ReviewPrompt] {
			return
		}
		seen[node.ReviewPrompt] = true
		prompts = append(prompts, node.ReviewPrompt)
	})
	return strings.Join(prompts, " ")
}

func forEachReviewNode(session *coop.Session, stepIndex int, fn func(*coop.SessionNode)) {
	if session == nil || stepIndex < 0 || stepIndex >= len(session.Steps) {
		return
	}
	nodes := session.Steps[stepIndex].Nodes
	for i := range nodes {
		if nodes[i].State == coop.NodeReview {
			fn(&nodes[i])
		}
	}
}
