package coop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionStructuredEventsPreserveSchemaV2Compatibility(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	session := &Session{
		SchemaVersion: CurrentSessionSchemaVersion,
		ID:            "structured_events",
		Blueprint:     "test",
		Status:        SessionActive,
		Steps: []SessionStep{{
			StepDefinition: StepDefinition{Key: "step", Title: "Step"},
			Nodes: []SessionNode{
				{
					NodeDefinition: NodeDefinition{
						Type:                   NodeAsyncHandler,
						Key:                    "event",
						Title:                  "Event",
						Events:                 []EventDefinition{{EventType: "invoice.paid", EventPayloadType: "snapshot"}},
						ExpectedNumberOfEvents: 1,
					},
					State: NodePending,
				},
				{
					NodeDefinition: NodeDefinition{
						Type:  NodeAPIRequest,
						Key:   "request",
						Title: "Request",
						Request: &APIRequest{
							Path:           "/v1/test",
							Method:         "post",
							RequestOptions: map[string]interface{}{"idempotency_key": "test-key"},
						},
						Link: "https://example.com/checkout",
					},
					State: NodePending,
				},
			},
		}},
	}
	require.NoError(t, store.Write(session))

	raw, err := os.ReadFile(filepath.Join(dir, "structured_events.json"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"schema_version": 2`)
	assert.Contains(t, string(raw), `"event_definitions"`)

	var legacy struct {
		SchemaVersion int `json:"schema_version"`
		Steps         []struct {
			Nodes []struct {
				Events []string `json:"events"`
			} `json:"nodes"`
		} `json:"steps"`
	}
	require.NoError(t, json.Unmarshal(raw, &legacy))
	assert.Equal(t, CurrentSessionSchemaVersion, legacy.SchemaVersion)
	assert.Equal(t, []string{"invoice.paid"}, legacy.Steps[0].Nodes[0].Events)

	loaded, err := store.Read("structured_events")
	require.NoError(t, err)
	event := loaded.Steps[0].Nodes[0].Events[0]
	assert.Equal(t, "invoice.paid", event.EventType)
	assert.Equal(t, "snapshot", event.EventPayloadType)
	assert.Equal(t, 1, loaded.Steps[0].Nodes[0].ExpectedNumberOfEvents)
	assert.Equal(t, "test-key", loaded.Steps[0].Nodes[1].Request.RequestOptions["idempotency_key"])
	assert.Equal(t, "https://example.com/checkout", loaded.Steps[0].Nodes[1].Link)

	_, err = store.Update("structured_events", func(session *Session) error {
		session.Steps[0].Nodes[0].Activity = "still working"
		return nil
	})
	require.NoError(t, err)
	loaded, err = store.Read("structured_events")
	require.NoError(t, err)
	assert.Equal(t, "snapshot", loaded.Steps[0].Nodes[0].Events[0].EventPayloadType)
}

func TestSessionNodeReadsLegacyStringEvents(t *testing.T) {
	var node SessionNode
	require.NoError(t, json.Unmarshal([]byte(`{"events":["checkout.session.completed"],"state":"pending"}`), &node))
	assert.Equal(t, []string{"checkout.session.completed"}, node.EventTypes())
	assert.Empty(t, node.Events[0].EventPayloadType)

	encoded, err := json.Marshal(node)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "event_definitions")
}

func TestSessionNodeMigratesObjectValuedEvents(t *testing.T) {
	var node SessionNode
	require.NoError(t, json.Unmarshal([]byte(`{"events":[{"eventType":"invoice.paid","eventPayloadType":"snapshot"}],"state":"pending"}`), &node))
	assert.Equal(t, []string{"invoice.paid"}, node.EventTypes())
	assert.Equal(t, "snapshot", node.Events[0].EventPayloadType)

	encoded, err := json.Marshal(node)
	require.NoError(t, err)
	var legacy struct {
		Events []string `json:"events"`
	}
	require.NoError(t, json.Unmarshal(encoded, &legacy))
	assert.Equal(t, []string{"invoice.paid"}, legacy.Events)
	assert.Contains(t, string(encoded), "event_definitions")
}

func TestSessionNodeRejectsInconsistentEventDefinitions(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "length",
			raw:  `{"events":["invoice.paid"],"event_definitions":[]}`,
		},
		{
			name: "order",
			raw:  `{"events":["invoice.paid"],"event_definitions":[{"eventType":"checkout.session.completed","eventPayloadType":"snapshot"}]}`,
		},
		{
			name: "null",
			raw:  `{"events":[],"event_definitions":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node SessionNode
			err := json.Unmarshal([]byte(tt.raw), &node)
			require.Error(t, err)
		})
	}
}
