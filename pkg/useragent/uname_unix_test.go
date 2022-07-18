//go:build !windows
// +build !windows

package useragent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnameNotEmpty(t *testing.T) {
	u := getUname()
	t.Logf("%x", u) // For NULL trim paranoia
	require.NotEmpty(t, u)
}

func TestTrimNulls(t *testing.T) {
	input := [256]byte{0xff}
	t.Log(input)
	output := trimNulls(input[:])
	t.Log(output)
	require.NotEqual(t, len(input), len(output))
}
