package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLongestShortcut(t *testing.T) {
	shortcuts := []string{"bender", "fry", "leela"}

	require.Equal(t, getLongestShortcut(shortcuts), 6)
}

func TestNamePadding(t *testing.T) {
	require.Equal(t, padName("fry", 6), "fry   ")
	require.Equal(t, padName("leela", 6), "leela ")
	require.Equal(t, padName("bender", 6), "bender")
}
