package signals

// Output types for this package's substring code-signal scanning — honest
// about their strength: these are signals, never verdicts.

// HandlerSignals reports whether the USER'S code mentions each webhook event
// type the pack declares as expected for the migrated integration.
type HandlerSignals struct {
	Events     []EventSignal `json:"events"`
	AllPresent bool          `json:"all_present"`
}

type EventSignal struct {
	Event   string   `json:"event"`
	Note    string   `json:"note,omitempty"`
	Files   []string `json:"files,omitempty"`
	Present bool     `json:"present"`
}
