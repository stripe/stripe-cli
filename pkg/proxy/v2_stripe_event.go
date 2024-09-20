package proxy

import (
	"encoding/json"
	"fmt"
)

// V2EventPayload describes the payload from the server for a v2 event
type V2EventPayload struct {
	Created       string               `json:"created"`
	Data          json.RawMessage      `json:"data,omitempty"`
	ID            string               `json:"id"`
	Object        string               `json:"object"`
	RelatedObject primaryRelatedObject `json:"related_object"`
	Type          string               `json:"type"`
	Context       string               `json:"context,omitempty"`
}

type primaryRelatedObject struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// URLForEventID builds a full URL from a V2StripeEvent ID.
func (e *V2EventPayload) URLForEventID(cliEndpointID string) string {
	return fmt.Sprintf("https://dashboard.stripe.com/workbench/webhooks/%s?event=%s", cliEndpointID, e.ID)
}

// IsConnect returns true if this event is associated with a connected account
func (e *V2EventPayload) IsConnect() bool {
	return e.Context != ""
}
