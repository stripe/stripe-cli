package websocket

import "encoding/json"

// V2EventPayload describes the payload from the server for a v2 event
type V2EventPayload struct {
	Created       string               `json:"created"`
	Data          json.RawMessage      `json:"data,omitempty"`
	ID            string               `json:"id"`
	Object        string               `json:"object"`
	Reason        eventReason          `json:"reason"`
	RelatedObject primaryRelatedObject `json:"related_object"`
	Type          string               `json:"type"`
}

type eventReason struct {
	Type string `json:"type"`
}

type primaryRelatedObject struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url"`
}
