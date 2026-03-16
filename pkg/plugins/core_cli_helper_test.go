package plugins

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEcho(t *testing.T) {
	coreCLIHelper := &coreCLIHelper{}
	output, err := coreCLIHelper.Echo("test")
	require.NoError(t, err)
	require.Equal(t, "test", output)
}
