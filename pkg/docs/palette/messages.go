package palette

// SearchResultMsg is the message a Mode's Search closure eventually
// emits with the items matching a given query. Mode is the Name of
// the mode that produced this result — the palette uses it to route
// the items into the right per-mode cache and to reject results from
// a mode that's no longer active. Err is non-nil if the search
// failed; Results should be ignored in that case.
type SearchResultMsg struct {
	Mode    string
	Query   string
	Results []Item
	Err     error
}

// SelectedMsg is dispatched when the user presses Execute (Enter by
// default) on a highlighted item. The host program type-switches on
// Item to decide how to react — close the palette, log, navigate,
// etc. When the item is a Command with a non-nil Run, the palette
// also fires Run()'s tea.Cmd in the same batch.
type SelectedMsg struct {
	Item Item
}
