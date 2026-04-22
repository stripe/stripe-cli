package cmdutil

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeCmd(name string, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{Use: name}
	for _, c := range children {
		cmd.AddCommand(c)
	}
	return cmd
}

func TestFindSubCmd_EmptyNames(t *testing.T) {
	root := makeCmd("root")
	found, ok := FindSubCmd(root)
	require.True(t, ok)
	assert.Equal(t, root, found)
}

func TestFindSubCmd_SingleLevel_Found(t *testing.T) {
	child := makeCmd("child")
	root := makeCmd("root", child)
	found, ok := FindSubCmd(root, "child")
	require.True(t, ok)
	assert.Equal(t, child, found)
}

func TestFindSubCmd_SingleLevel_NotFound(t *testing.T) {
	root := makeCmd("root", makeCmd("other"))
	found, ok := FindSubCmd(root, "child")
	assert.False(t, ok)
	assert.Nil(t, found)
}

func TestFindSubCmd_MultiLevel_Found(t *testing.T) {
	grandchild := makeCmd("grandchild")
	child := makeCmd("child", grandchild)
	root := makeCmd("root", child)
	found, ok := FindSubCmd(root, "child", "grandchild")
	require.True(t, ok)
	assert.Equal(t, grandchild, found)
}

func TestFindSubCmd_MultiLevel_IntermediateMissing(t *testing.T) {
	// cobra.Find would return the closest ancestor ("root") with remaining=["missing","grandchild"]
	// FindSubCmd must normalize this to nil, false.
	root := makeCmd("root", makeCmd("child"))
	found, ok := FindSubCmd(root, "missing", "grandchild")
	assert.False(t, ok)
	assert.Nil(t, found)
}

func TestFindSubCmd_MultiLevel_LeafMissing(t *testing.T) {
	// cobra.Find returns "child" with remaining=["missing"] — must normalize to nil, false.
	child := makeCmd("child")
	root := makeCmd("root", child)
	found, ok := FindSubCmd(root, "child", "missing")
	assert.False(t, ok)
	assert.Nil(t, found)
}
