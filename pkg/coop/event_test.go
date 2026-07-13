package coop

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventDefinitionLegacyStringRoundTrip(t *testing.T) {
	var event EventDefinition
	require.NoError(t, json.Unmarshal([]byte(`"checkout.session.completed"`), &event))
	assert.Equal(t, "checkout.session.completed", event.EventType)
	assert.Empty(t, event.EventPayloadType)

	encoded, err := json.Marshal(event)
	require.NoError(t, err)
	assert.Equal(t, `"checkout.session.completed"`, string(encoded))
}

func TestEventDefinitionStructuredRoundTrip(t *testing.T) {
	raw := `{"eventType":"invoice.paid","eventPayloadType":"snapshot"}`

	var event EventDefinition
	require.NoError(t, json.Unmarshal([]byte(raw), &event))
	assert.Equal(t, "invoice.paid", event.EventType)
	assert.Equal(t, "snapshot", event.EventPayloadType)

	encoded, err := json.Marshal(event)
	require.NoError(t, err)
	assert.JSONEq(t, raw, string(encoded))
}

func TestEventDefinitionRejectsInvalidJSONShapes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "null", raw: `null`},
		{name: "number", raw: `42`},
		{name: "boolean", raw: `true`},
		{name: "array", raw: `[]`},
		{name: "missing event type", raw: `{}`},
		{name: "empty object event type", raw: `{"eventType":""}`},
		{name: "blank object event type", raw: `{"eventType":" "}`},
		{name: "padded object event type", raw: `{"eventType":" invoice.paid "}`},
		{name: "null payload type", raw: `{"eventType":"invoice.paid","eventPayloadType":null}`},
		{name: "padded payload type", raw: `{"eventType":"invoice.paid","eventPayloadType":" snapshot "}`},
		{name: "unknown field", raw: `{"eventType":"invoice.paid","futureField":true}`},
		{name: "empty string event type", raw: `""`},
		{name: "blank string event type", raw: `" "`},
		{name: "padded string event type", raw: `" invoice.paid "`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event EventDefinition
			err := json.Unmarshal([]byte(tt.raw), &event)
			require.Error(t, err)
		})
	}
}

func TestEventDefinitionProgrammaticValueUsesObjectForm(t *testing.T) {
	encoded, err := json.Marshal(EventDefinition{EventType: "invoice.paid"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"eventType":"invoice.paid"}`, string(encoded))
}
