package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestNextPendingStepInChapter(t *testing.T) {
	session := &coop.Session{
		Chapters: []coop.SessionChapter{
			{
				Nodes: []coop.SessionNode{
					{State: coop.StepDone},
					{State: coop.StepPending},
				},
			},
			{
				Nodes: []coop.SessionNode{
					{State: coop.StepPending},
				},
			},
		},
	}

	assert.Equal(t, 2, NextPendingStepInChapter(session, 0, 1))
	assert.Equal(t, 0, NextPendingStepInChapter(session, 0, 2))
	assert.Equal(t, 3, NextPendingStepInChapter(session, 1, 0))
}

func TestChapterReviewApplies(t *testing.T) {
	session := &coop.Session{
		Chapters: []coop.SessionChapter{
			{
				ReviewGranularity: coop.ReviewGranularityChapter,
				Nodes:             []coop.SessionNode{{State: coop.StepReview}},
			},
			{
				ReviewGranularity: coop.ReviewGranularityStep,
				Nodes:             []coop.SessionNode{{State: coop.StepReview}},
			},
		},
	}

	assert.True(t, ChapterReviewApplies(session, 1))
	assert.False(t, ChapterReviewApplies(session, 2))
	assert.False(t, ChapterReviewApplies(session, 99))
}
