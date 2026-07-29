package coop

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredOutputsAndResolveNodeDefinition(t *testing.T) {
	session := outputTestSession()

	required, err := session.RequiredOutputs(1)
	require.NoError(t, err)
	assert.Equal(t, []RequiredOutput{
		{Field: "id"},
		{Field: "latest_version"},
		{Field: "lookup_key"},
	}, required)

	required, err = session.RequiredOutputs(2)
	require.NoError(t, err)
	assert.Equal(t, []RequiredOutput{{Source: "create-clock-request", Field: "id"}}, required)

	required, err = session.RequiredOutputs(3)
	require.NoError(t, err)
	assert.Equal(t, []RequiredOutput{{Source: "0", Field: "id"}}, required)

	resolved, err := session.ResolvedNodeDefinition(4)
	require.NoError(t, err)
	require.NotNil(t, resolved.Request)
	assert.Equal(t, "/v1/products/prod_123/versions/7", resolved.Request.Path)

	params, ok := resolved.Request.Params.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "clock_123", params["clock"])
	assert.Equal(t, "in_123", params["invoice"])
	assert.Equal(t, float64(7), params["version"])
	require.Contains(t, params, "metered")
	assert.Equal(t, "usage", params["metered"])
}

func TestResolveNodeDefinitionRejectsMissingOutput(t *testing.T) {
	session := outputTestSession()
	delete(session.Steps[0].Nodes[0].Outputs[DefaultOutputSource], "id")

	_, err := session.ResolvedNodeDefinition(4)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `missing output "id"`)
	assert.Contains(t, err.Error(), "${node.setup.create-product:id}")
}

func TestMissingRequiredOutputs(t *testing.T) {
	session := outputTestSession()
	delete(session.Steps[0].Nodes[0].Outputs[DefaultOutputSource], "lookup_key")

	missing, err := session.MissingRequiredOutputs(1)
	require.NoError(t, err)
	assert.Equal(t, []RequiredOutput{{Field: "lookup_key"}}, missing)
}

func TestEmbeddedBlueprintReferencesAreExposedAsRequiredOutputs(t *testing.T) {
	ids, err := ListBlueprints()
	require.NoError(t, err)

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			bp, err := LoadBlueprint(id)
			require.NoError(t, err)

			expected := map[string]bool{}
			require.NoError(t, walkNodeReferences(bp, func(reference nodeReference) error {
				expected[reference.Ref+":"+reference.Field] = true
				return nil
			}))

			session := NewSessionFromBlueprint(bp, "required_outputs_"+id, nil, nil)
			actual := map[string]bool{}
			nodeNumber := 0
			for _, step := range session.Steps {
				for _, node := range step.Nodes {
					nodeNumber++
					required, err := session.RequiredOutputs(nodeNumber)
					require.NoError(t, err)
					for _, output := range required {
						ref := step.Key + "." + node.Key
						if output.Source != "" {
							ref += "." + output.Source
						}
						actual[ref+":"+output.Field] = true
					}
				}
			}
			assert.Equal(t, expected, actual)
		})
	}
}

func outputTestSession() *Session {
	raw := func(value string) json.RawMessage {
		data, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		return data
	}
	return &Session{
		ID:     "outputs",
		Status: SessionActive,
		Steps: []SessionStep{{
			StepDefinition: StepDefinition{Key: "setup", Title: "Setup"},
			Nodes: []SessionNode{
				{
					NodeDefinition: NodeDefinition{Key: "create-product", Title: "Create product", Type: NodeAPIRequest},
					State:          NodeDone,
					Outputs: NodeOutputs{
						DefaultOutputSource: {
							"id":             raw("prod_123"),
							"latest_version": json.RawMessage("7"),
							"lookup_key":     raw("metered"),
						},
					},
				},
				{
					NodeDefinition: NodeDefinition{Key: "create-clock", Title: "Create clock", Type: NodeTestHelper},
					State:          NodeDone,
					Outputs: NodeOutputs{
						"create-clock-request": {"id": raw("clock_123")},
					},
				},
				{
					NodeDefinition: NodeDefinition{Key: "wait-for-invoice", Title: "Wait for invoice", Type: NodeAsyncHandler},
					State:          NodeDone,
					Outputs: NodeOutputs{
						"0": {"id": raw("in_123")},
					},
				},
				{
					NodeDefinition: NodeDefinition{
						Key:   "use-results",
						Title: "Use results",
						Type:  NodeAPIRequest,
						Request: &APIRequest{
							Path:   "/v1/products/${node.setup.create-product:id}/versions/${node.setup.create-product:latest_version}",
							Method: "post",
							Params: map[string]any{
								"clock":   "${node.setup.create-clock.create-clock-request:id}",
								"invoice": "${node.setup.wait-for-invoice.0:id}",
								"version": "${node.setup.create-product:latest_version}",
								"${node.setup.create-product:lookup_key}": "usage",
							},
						},
					},
					State: NodePending,
				},
			},
		}},
	}
}
