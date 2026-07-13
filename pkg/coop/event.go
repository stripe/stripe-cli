package coop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// EventDefinition describes a webhook or async event required by a blueprint.
// Blueprint JSON may use either the legacy event-type string or this object form.
type EventDefinition struct {
	EventType        string `json:"eventType"`
	EventPayloadType string `json:"eventPayloadType,omitempty"`
	legacyString     bool
}

// UnmarshalJSON accepts legacy string events and structured event definitions.
func (e *EventDefinition) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("event definition must be a string or object")
	}

	var parsed EventDefinition
	switch trimmed[0] {
	case '"':
		if err := json.Unmarshal(trimmed, &parsed.EventType); err != nil {
			return fmt.Errorf("parsing event type: %w", err)
		}
		parsed.legacyString = true
	case '{':
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &fields); err != nil {
			return fmt.Errorf("parsing event definition: %w", err)
		}
		for field := range fields {
			if field != "eventType" && field != "eventPayloadType" {
				return fmt.Errorf("event definition contains unsupported field %q", field)
			}
		}
		eventType, ok := fields["eventType"]
		if !ok || bytes.Equal(bytes.TrimSpace(eventType), []byte("null")) {
			return fmt.Errorf("event definition eventType is required")
		}
		if err := json.Unmarshal(eventType, &parsed.EventType); err != nil {
			return fmt.Errorf("parsing event definition eventType: %w", err)
		}
		if payloadType, ok := fields["eventPayloadType"]; ok {
			if bytes.Equal(bytes.TrimSpace(payloadType), []byte("null")) {
				return fmt.Errorf("event definition eventPayloadType must be a string")
			}
			if err := json.Unmarshal(payloadType, &parsed.EventPayloadType); err != nil {
				return fmt.Errorf("parsing event definition eventPayloadType: %w", err)
			}
		}
	default:
		return fmt.Errorf("event definition must be a string or object")
	}

	if strings.TrimSpace(parsed.EventType) == "" {
		return fmt.Errorf("event definition eventType is required")
	}
	if parsed.EventType != strings.TrimSpace(parsed.EventType) {
		return fmt.Errorf("event definition eventType must not have surrounding whitespace")
	}
	if parsed.EventPayloadType != strings.TrimSpace(parsed.EventPayloadType) {
		return fmt.Errorf("event definition eventPayloadType must not have surrounding whitespace")
	}
	*e = parsed
	return nil
}

// MarshalJSON preserves the legacy string form when that was the input form.
func (e EventDefinition) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(e.EventType) == "" {
		return nil, fmt.Errorf("event definition eventType is required")
	}
	if e.EventType != strings.TrimSpace(e.EventType) {
		return nil, fmt.Errorf("event definition eventType must not have surrounding whitespace")
	}
	if e.EventPayloadType != strings.TrimSpace(e.EventPayloadType) {
		return nil, fmt.Errorf("event definition eventPayloadType must not have surrounding whitespace")
	}
	if e.legacyString && e.EventPayloadType == "" {
		return json.Marshal(e.EventType)
	}

	type eventDefinition EventDefinition
	return json.Marshal(eventDefinition(e))
}
