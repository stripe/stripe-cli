package rendering

import "encoding/json"

// JSONEnvelope is the standard output format for --format=json.
// Only success output goes here (stdout). Errors go to stderr + non-zero exit.
type JSONEnvelope struct {
	Command string          `json:"command"`
	Data    []EnvelopeBlock `json:"data"`
}

// EnvelopeBlock represents a single output block in the JSON envelope.
type EnvelopeBlock struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
