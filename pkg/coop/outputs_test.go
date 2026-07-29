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
	require.NotNil(t, resolved.APIRequestDetails)
	request := resolved.APIRequestDetails.Fixture
	assert.Equal(t, "/v1/products/prod_123/versions/7", request.Path)

	params := request.Params
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
	assert.Contains(t, err.Error(), "${node.create-product:id}")
}

func TestMissingRequiredOutputs(t *testing.T) {
	session := outputTestSession()
	delete(session.Steps[0].Nodes[0].Outputs[DefaultOutputSource], "lookup_key")

	missing, err := session.MissingRequiredOutputs(1)
	require.NoError(t, err)
	assert.Equal(t, []RequiredOutput{{Field: "lookup_key"}}, missing)
}

func TestAPIBlueprintReferencesAreExposedAsRequiredOutputs(t *testing.T) {
	session, err := NewSessionFromBlueprint(loadTestBlueprint(t), "required_outputs", nil, nil)
	require.NoError(t, err)

	required, err := session.RequiredOutputs(2)
	require.NoError(t, err)
	assert.Contains(t, required, RequiredOutput{Field: "id"})
	assert.Contains(t, required, RequiredOutput{Field: "client_secret"})
}

func TestResolveNodeDefinitionPreservesCurrentNodeRequestReferences(t *testing.T) {
	session, err := NewSessionFromBlueprint(loadTestBlueprint(t), "current_node_reference", nil, nil)
	require.NoError(t, err)
	session.Steps[1].Nodes[0].Outputs = NodeOutputs{
		DefaultOutputSource: {
			"id":            json.RawMessage(`"pi_123"`),
			"client_secret": json.RawMessage(`"pi_123_secret_456"`),
		},
	}

	resolved, err := session.ResolvedNodeDefinition(5)
	require.NoError(t, err)
	require.NotNil(t, resolved.UIComponentDetails)
	require.Len(t, resolved.UIComponentDetails.Options, 2)
	require.Len(t, resolved.UIComponentDetails.Options[1].Requests, 2)
	assert.Equal(t, "/v1/payment_intents/pi_123", resolved.UIComponentDetails.Options[1].Requests[0].Path)
	assert.Equal(t, "/v1/payment_intents/${node.show-payment.0:id}/confirm", resolved.UIComponentDetails.Options[1].Requests[1].Path)
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
			BlueprintStepDefinition: BlueprintStepDefinition{Key: "setup", Title: MessageDescriptor{DefaultMessage: "Setup"}},
			Nodes: []SessionNode{
				{
					BlueprintNode: BlueprintNode{Key: "create-product", Title: MessageDescriptor{DefaultMessage: "Create product"}, NodeType: NodeAPIRequest},
					State:         NodeDone,
					Outputs: NodeOutputs{
						DefaultOutputSource: {
							"id":             raw("prod_123"),
							"latest_version": json.RawMessage("7"),
							"lookup_key":     raw("metered"),
						},
					},
				},
				{
					BlueprintNode: BlueprintNode{Key: "create-clock", Title: MessageDescriptor{DefaultMessage: "Create clock"}, NodeType: NodeTestHelper},
					State:         NodeDone,
					Outputs: NodeOutputs{
						"create-clock-request": {"id": raw("clock_123")},
					},
				},
				{
					BlueprintNode: BlueprintNode{Key: "wait-for-invoice", Title: MessageDescriptor{DefaultMessage: "Wait for invoice"}, NodeType: NodeAsyncHandler},
					State:         NodeDone,
					Outputs: NodeOutputs{
						"0": {"id": raw("in_123")},
					},
				},
				{
					BlueprintNode: BlueprintNode{
						Key:      "use-results",
						Title:    MessageDescriptor{DefaultMessage: "Use results"},
						NodeType: NodeAPIRequest,
						APIRequestDetails: &BlueprintAPIRequestDetails{Fixture: BlueprintRequestFixture{
							Path:   "/v1/products/${node.create-product:id}/versions/${node.setup.create-product:latest_version}",
							Method: "post",
							Params: map[string]any{
								"clock":   "${node.setup.create-clock.create-clock-request:id}",
								"invoice": "${node.setup.wait-for-invoice.0:id}",
								"version": "${node.setup.create-product:latest_version}",
								"${node.setup.create-product:lookup_key}": "usage",
							},
						}},
					},
					State: NodePending,
				},
			},
		}},
	}
}
