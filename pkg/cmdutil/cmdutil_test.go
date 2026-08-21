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

func TestArgsAfter(t *testing.T) {
	tests := []struct {
		name string
		args []string
		find string
		want []string
	}{
		{
			name: "returns the args following the name",
			args: []string{"stripe", "directory", "search", "x"},
			find: "directory",
			want: []string{"search", "x"},
		},
		{
			name: "drops anything before the name",
			args: []string{"stripe", "--log-level", "debug", "directory", "search"},
			find: "directory",
			want: []string{"search"},
		},
		{
			name: "stops at the first occurrence so a repeated name is forwarded",
			args: []string{"stripe", "directory", "search", "directory"},
			find: "directory",
			want: []string{"search", "directory"},
		},
		{
			name: "returns empty when the name is last",
			args: []string{"stripe", "directory"},
			find: "directory",
			want: []string{},
		},
		{
			name: "returns empty when the name is absent",
			args: []string{"stripe", "listen"},
			find: "directory",
			want: []string{},
		},
		{
			name: "returns empty for no args",
			args: []string{},
			find: "directory",
			want: []string{},
		},
		{
			name: "matches an empty element for an empty name",
			args: []string{""},
			find: "",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ArgsAfter(tt.args, tt.find))
		})
	}
}

func TestArgsAfter_DoesNotAliasInput(t *testing.T) {
	args := []string{"stripe", "directory", "search"}

	got := ArgsAfter(args, "directory")
	got[0] = "mutated"

	assert.Equal(t, []string{"stripe", "directory", "search"}, args)
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
