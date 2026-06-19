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

	assert.Equal(t, 2, NextPendingNodeInStep(session, 1, 1))
	assert.Equal(t, 0, NextPendingNodeInStep(session, 1, 2))
	assert.Equal(t, 3, NextPendingNodeInStep(session, 2, 0))
}
