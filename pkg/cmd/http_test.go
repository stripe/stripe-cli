package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGetCmd(t *testing.T) {
	cmd := newGetCmd(false)
	assert.Equal(t, http.MethodGet, cmd.Method)
	assert.False(t, cmd.IsPreviewCommand)
	assert.Equal(t, "get <id or path>", cmd.Cmd.Use)
	require.NotNil(t, cmd.Cmd.Flags().Lookup("limit"))
	require.NotNil(t, cmd.Cmd.Flags().Lookup("starting-after"))
	require.NotNil(t, cmd.Cmd.Flags().Lookup("ending-before"))
	require.Contains(t, cmd.Cmd.Flags().Lookup("format").Usage, "'json' - Output the response in JSON format (default)")
	require.Contains(t, cmd.Cmd.Flags().Lookup("format").Usage, "'toon' - Output the response in TOON format")
}

func TestNewGetCmd_Preview(t *testing.T) {
	cmd := newGetCmd(true)
	assert.True(t, cmd.IsPreviewCommand)
	assert.Equal(t, "get <id or path>", cmd.Cmd.Use)
	require.NotNil(t, cmd.Cmd.Flags().Lookup("limit"))
}

func TestNewPostCmd(t *testing.T) {
	cmd := newPostCmd(false)
	assert.Equal(t, http.MethodPost, cmd.Method)
	assert.False(t, cmd.IsPreviewCommand)
	assert.Equal(t, "post <path>", cmd.Cmd.Use)
	assert.Nil(t, cmd.Cmd.Flags().Lookup("limit"))
}

func TestNewPostCmd_Preview(t *testing.T) {
	cmd := newPostCmd(true)
	assert.True(t, cmd.IsPreviewCommand)
}

func TestNewDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd(false)
	assert.Equal(t, http.MethodDelete, cmd.Method)
	assert.False(t, cmd.IsPreviewCommand)
	assert.Equal(t, "delete <path>", cmd.Cmd.Use)
	assert.Nil(t, cmd.Cmd.Flags().Lookup("limit"))
}

func TestNewDeleteCmd_Preview(t *testing.T) {
	cmd := newDeleteCmd(true)
	assert.True(t, cmd.IsPreviewCommand)
}
