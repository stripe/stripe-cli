package proxy

import "strings"

// interopEventPrefix is prepended to a snapshot event type when that event is
// delivered as a v2 event, e.g. charge.succeeded is delivered as
// v1.charge.succeeded.
const interopEventPrefix = "v1."

// isInteropEvent reports whether eventType is the v2 rendering of a snapshot
// event. Those "interop" events duplicate what the webhooks channel already
// delivers, so a thin event wildcard subscription skips them and only forwards
// them when the user asks for them by name.
//
// Thin events that are v1-prefixed but have no snapshot counterpart, such as
// v1.billing.meter.no_meter_found, are not interop events.
func isInteropEvent(eventType string) bool {
	base, hasPrefix := strings.CutPrefix(eventType, interopEventPrefix)
	if !hasPrefix || base == "*" {
		return false
	}

	return validEvents[base]
}
