package useragent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnameWindowsNotEmpty(t *testing.T) {
	u := getUname()
	t.Logf("Raw uname: %q", u)
	require.NotEmpty(t, u, "getUname() should not return an empty string")

	parts := strings.Fields(u)
	require.GreaterOrEqual(t, len(parts), 5, "uname output should have at least 5 components")
	require.Equal(t, "Windows", parts[0], "uname should start with 'Windows'")
}

func TestTrimNullsForWindows(t *testing.T) {
	input := make([]byte, 256)
	input[0] = 'W'
	input[1] = 'i'
	input[2] = 'n'
	input[3] = '!'
	// The rest are zeros by default

	t.Logf("Input bytes: %x", input)
	output := trimNulls(input)
	t.Logf("Output: %q", output)

	require.Equal(t, string(output), "Win!")
	require.Less(t, len(output), len(input), "trimNulls should shrink the input if nulls exist")
}
