package rendering

import "encoding/json"

// JSONEnvelope is the standard output format for --format=json.
// The data field is an ordered list of typed blocks.
type JSONEnvelope struct {
	Ok      bool              `json:"ok"`
	Command string            `json:"command"`
	Data    []EnvelopeBlock   `json:"data"`
}

// EnvelopeBlock represents a single output block in the JSON envelope.
type EnvelopeBlock struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
