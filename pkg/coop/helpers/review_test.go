package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestNextPendingNodeInStep(t *testing.T) {
	session := &coop.Session{
		Steps: []coop.SessionStep{
			{
				Nodes: []coop.SessionNode{
					{State: coop.NodeDone},
					{State: coop.NodePending},
				},
			},
			{
				Nodes: []coop.SessionNode{
					{State: coop.NodePending},
				},
			},
		},
	}

	assert.Equal(t, 2, NextPendingNodeInStep(session, 0, 1))
	assert.Equal(t, 0, NextPendingNodeInStep(session, 0, 2))
	assert.Equal(t, 3, NextPendingNodeInStep(session, 1, 0))
}

func TestStepReviewApplies(t *testing.T) {
	session := &coop.Session{
		Steps: []coop.SessionStep{
			{
				ReviewGranularity: coop.ReviewGranularityStep,
				Nodes:             []coop.SessionNode{{State: coop.NodeReview}},
			},
			{
				ReviewGranularity: coop.ReviewGranularityNode,
				Nodes:             []coop.SessionNode{{State: coop.NodeReview}},
			},
		},
	}

	assert.True(t, StepReviewApplies(session, 1))
	assert.False(t, StepReviewApplies(session, 2))
	assert.False(t, StepReviewApplies(session, 99))
}
