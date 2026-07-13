package coop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type sessionEventDefinition struct {
	EventType        string `json:"eventType"`
	EventPayloadType string `json:"eventPayloadType,omitempty"`
}

func (e *sessionEventDefinition) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("session event definition must be an object")
	}

	var event EventDefinition
	if err := json.Unmarshal(trimmed, &event); err != nil {
		return err
	}
	e.EventType = event.EventType
	e.EventPayloadType = event.EventPayloadType
	return nil
}

// MarshalJSON keeps the schema-v2 events field readable by older CLI versions.
// Structured event metadata is stored separately in an additive field. Older
// clients preserve event names but may drop that metadata when rewriting a file.
func (n SessionNode) MarshalJSON() ([]byte, error) {
	type sessionNodeAlias SessionNode

	legacyNode := n
	legacyNode.Events = make([]EventDefinition, len(n.Events))
	var eventDefinitions []sessionEventDefinition
	for i, event := range n.Events {
		legacyNode.Events[i] = event
		legacyNode.Events[i].legacyString = true
		legacyNode.Events[i].EventPayloadType = ""
		if !event.legacyString || event.EventPayloadType != "" {
			eventDefinitions = make([]sessionEventDefinition, len(n.Events))
		}
	}
	if eventDefinitions != nil {
		for i, event := range n.Events {
			eventDefinitions[i] = sessionEventDefinition{
				EventType:        event.EventType,
				EventPayloadType: event.EventPayloadType,
			}
		}
	}

	return json.Marshal(struct {
		sessionNodeAlias
		EventDefinitions []sessionEventDefinition `json:"event_definitions,omitempty"`
	}{
		sessionNodeAlias: sessionNodeAlias(legacyNode),
		EventDefinitions: eventDefinitions,
	})
}

// UnmarshalJSON restores structured metadata while accepting legacy sessions.
func (n *SessionNode) UnmarshalJSON(data []byte) error {
	type sessionNodeAlias SessionNode
	var stored struct {
		sessionNodeAlias
		EventDefinitions json.RawMessage `json:"event_definitions"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}

	if len(stored.EventDefinitions) > 0 {
		if bytes.Equal(bytes.TrimSpace(stored.EventDefinitions), []byte("null")) {
			return fmt.Errorf("event_definitions must be an array")
		}
		var definitions []sessionEventDefinition
		if err := json.Unmarshal(stored.EventDefinitions, &definitions); err != nil {
			return fmt.Errorf("parsing event_definitions: %w", err)
		}
		if len(definitions) != len(stored.Events) {
			return fmt.Errorf("event_definitions length %d does not match events length %d", len(definitions), len(stored.Events))
		}
		for i, definition := range definitions {
			if strings.TrimSpace(definition.EventType) == "" {
				return fmt.Errorf("event_definitions[%d].eventType is required", i)
			}
			if definition.EventType != stored.Events[i].EventType {
				return fmt.Errorf("event_definitions[%d].eventType %q does not match events[%d] %q", i, definition.EventType, i, stored.Events[i].EventType)
			}
			stored.Events[i] = EventDefinition{
				EventType:        definition.EventType,
				EventPayloadType: definition.EventPayloadType,
			}
		}
	}

	*n = SessionNode(stored.sessionNodeAlias)
	return nil
}
