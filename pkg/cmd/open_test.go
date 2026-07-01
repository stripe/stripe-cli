package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLongestShortcut(t *testing.T) {
	shortcuts := []string{"bender", "fry", "leela"}

	require.Equal(t, 6, getLongestShortcut(shortcuts))
}

func TestNamePadding(t *testing.T) {
	require.Equal(t, "fry   ", padName("fry", 6))
	require.Equal(t, "leela ", padName("leela", 6))
	require.Equal(t, "bender", padName("bender", 6))
}
