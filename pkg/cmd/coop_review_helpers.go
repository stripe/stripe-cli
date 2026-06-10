package cmd

import "github.com/stripe/stripe-cli/pkg/coop"

func nextPendingStepInChapter(session *coop.Session, chapterIndex, afterStep int) int {
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

func chapterReviewApplies(session *coop.Session, stepNum int) bool {
	chapter, _, _, err := session.ChapterByStepNumber(stepNum)
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
