package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/stretchr/testify/assert"
)

func TestKeyMap_ShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	bindings := km.ShortHelp()

	assert.Len(t, bindings, 1)
	assert.Equal(t, km.Help, bindings[0])
}

func TestKeyMap_FullHelp(t *testing.T) {
	km := DefaultKeyMap()
	groups := km.FullHelp()

	assert.Len(t, groups, 3)
	assert.Equal(t, []key.Binding{km.Up, km.Down}, groups[0])
	assert.Equal(t, []key.Binding{km.PageUp, km.PageDown}, groups[1])
	assert.Equal(t, []key.Binding{km.OpenInBrowser, km.Palette, km.Quit}, groups[2])
}

func TestKeyMap_SatisfiesHelpInterface(t *testing.T) {
	km := DefaultKeyMap()
	// Verify the interface is satisfied at compile time via assignment.
	var _ interface {
		ShortHelp() []key.Binding
		FullHelp() [][]key.Binding
	} = km
}
