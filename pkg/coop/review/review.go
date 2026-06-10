// Package review contains shared helpers for co-op human review boundaries.
package review

import "github.com/stripe/stripe-cli/pkg/coop"

func NextPendingStepInChapter(session *coop.Session, chapterIndex, afterStep int) int {
	step := 0
	for i := range session.Chapters {
		for j := range session.Chapters[i].Nodes {
			step++
			if i == chapterIndex && step > afterStep && session.Chapters[i].Nodes[j].State == coop.StepPending {
				return step
			}
		}
	}
	return 0
}

func ChapterReviewApplies(session *coop.Session, step int) bool {
	chapter, _, _, err := session.ChapterByStepNumber(step)
	if err != nil {
		return false
	}
	switch chapter.ReviewGranularity {
	case coop.ReviewGranularityAuto, coop.ReviewGranularityStep:
		return false
	default:
		return true
	}
}
