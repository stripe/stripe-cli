package coopcmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestCoopStatusWithoutSessionRecommendsBlueprint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cmd := newCoopStatusCmd().cmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	stderr := captureStderr(t, func() {
		err := cmd.Execute()
		require.Error(t, err)
		assert.IsType(t, RenderedError{}, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	require.NotNil(t, resp.Recovery)
	assert.Equal(t, "stripe coop recommend", resp.Recovery.Next)
	require.NoError(t, resp.Validate())
}
