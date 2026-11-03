package doctor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/judge"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// Too many arguments must surface as a UserInput error from topicAndDir
// (which RunE routes through fail(): JSON error body, exit 2), not be
// rejected by Cobra before RunE runs.
func TestTopicAndDirRejectsExtraArgs(t *testing.T) {
	for _, args := range [][]string{
		{"dpm", ".", "extra"},
		{"dpm", "dpm", "."}, // no unknown-topic overflow to catch it otherwise
	} {
		_, _, err := topicAndDir(args)
		require.Error(t, err, args)
		assert.Contains(t, err.Error(), "at most 2 arguments", args)
		cat, ok := errorcategory.Get(err)
		require.True(t, ok)
		assert.Equal(t, errorcategory.UserInput, cat)
	}
}

func TestDoctorCmdDefersArgCountToRunE(t *testing.T) {
	c := newDoctorCmd(nil, nil)
	assert.NoError(t, c.ValidateArgs([]string{"dpm", ".", "extra"}))
}

// A remediation's per-finding verdict (e.g. "safe to remove") must not
// replace an unverified finding's UNVERIFIED verdict.
func TestApplyFindingVerdictsKeepsUnverified(t *testing.T) {
	rep := &DoctorReport{Findings: []judge.Finding{
		{Finding: catalog.Finding{File: "a.rb", Line: 1, Confidence: catalog.ConfidencePossible}, Verdict: "UNVERIFIED: x", Class: "UNVERIFIED"},
		{Finding: catalog.Finding{File: "b.rb", Line: 2, Confidence: catalog.ConfidenceConfirmed}, Verdict: "REVIEW: y", Class: "REVIEW"},
	}}
	applyFindingVerdicts(rep, map[string]string{
		remediation.FindingKey("a.rb", 1): "CANDIDATE: safe",
		remediation.FindingKey("b.rb", 2): "CANDIDATE: safe",
	})
	assert.Equal(t, "UNVERIFIED", rep.Findings[0].Class)
	assert.Equal(t, "CANDIDATE", rep.Findings[1].Class)
	assert.Equal(t, map[string]int{"UNVERIFIED": 1, "CANDIDATE": 1}, rep.Summary)
}
