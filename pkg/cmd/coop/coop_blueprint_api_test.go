package coopcmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type recordingBlueprintRepository struct {
	delegate      commandTestBlueprintRepository
	listCalls     int
	retrieveCalls int
	retrievedKey  string
}

func (r *recordingBlueprintRepository) List(ctx context.Context) ([]coop.WorkbenchBlueprintSummary, error) {
	r.listCalls++
	return r.delegate.List(ctx)
}

func (r *recordingBlueprintRepository) Retrieve(ctx context.Context, key string) (*coop.WorkbenchBlueprint, error) {
	r.retrieveCalls++
	r.retrievedKey = key
	return r.delegate.Retrieve(ctx, key)
}

func useRecordingBlueprintRepository(t *testing.T) *recordingBlueprintRepository {
	t.Helper()
	previous := options
	repository := &recordingBlueprintRepository{}
	options.BlueprintRepository = repository
	t.Cleanup(func() { options = previous })
	return repository
}

func TestCoopRecommendUsesListAndFiltersLearning(t *testing.T) {
	repository := useRecordingBlueprintRepository(t)
	command := newCoopRecommendCmd().cmd
	command.SilenceErrors = true
	command.SilenceUsage = true

	output := captureStdout(t, func() {
		require.NoError(t, command.Execute())
	})

	var response struct {
		Blueprints []struct {
			ID        string `json:"id"`
			NodeCount *int   `json:"node_count"`
			StepCount int    `json:"step_count"`
		} `json:"blueprints"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &response))
	assert.Equal(t, 1, repository.listCalls)
	assert.Zero(t, repository.retrieveCalls)
	var ids []string
	for _, blueprint := range response.Blueprints {
		ids = append(ids, blueprint.ID)
	}
	assert.Contains(t, ids, "one-time-payment")
	assert.NotContains(t, ids, "testing-only")
	require.NotEmpty(t, response.Blueprints)
	assert.Nil(t, response.Blueprints[0].NodeCount)
	assert.Equal(t, 1, response.Blueprints[0].StepCount)
	assert.Contains(t, output, `"node_count": null`)
}

func TestCoopRecommendCanIncludeTestingBlueprints(t *testing.T) {
	useRecordingBlueprintRepository(t)
	command := newCoopRecommendCmd().cmd
	command.SilenceErrors = true
	command.SilenceUsage = true
	command.SetArgs([]string{"--include-testing"})

	output := captureStdout(t, func() {
		require.NoError(t, command.Execute())
	})
	assert.Contains(t, output, `"id": "testing-only"`)
}

func TestCoopRunRetrievesSelectedBlueprintAndPinsSession(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repository := useRecordingBlueprintRepository(t)
	command := newCoopAgentRunCmd().cmd
	command.SilenceErrors = true
	command.SilenceUsage = true
	command.SetArgs([]string{"one-time", "--setting", "simulation=success"})

	captureStdout(t, func() {
		require.NoError(t, command.Execute())
	})

	assert.Equal(t, 1, repository.listCalls)
	assert.Equal(t, 1, repository.retrieveCalls)
	assert.Equal(t, "one-time-payment", repository.retrievedKey)
	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	session, err := store.LatestSession()
	require.NoError(t, err)
	require.NotNil(t, session.BlueprintPin)
	assert.Equal(t, "one-time-payment", session.BlueprintPin.Key)
	assert.Equal(t, 6, session.BlueprintPin.BlueprintVersion)
	assert.Regexp(t, `^sha256:[0-9a-f]{64}$`, session.BlueprintPin.Digest)
	assert.Equal(t, "/v1/payment_intents", session.Steps[1].Nodes[0].Request.Path)
}

func TestStartSessionReusesCompiledBlueprint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repository := useRecordingBlueprintRepository(t)
	blueprint, err := coop.LoadBlueprint(t.Context(), repository, "one-time", nil)
	require.NoError(t, err)

	session, err := (&coopRunCmd{}).startSessionQuietly(blueprint)
	require.NoError(t, err)
	assert.Equal(t, "one-time-payment", session.Blueprint)
	assert.Equal(t, 1, repository.listCalls)
	assert.Equal(t, 1, repository.retrieveCalls)
}
