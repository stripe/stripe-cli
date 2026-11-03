package doctor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
