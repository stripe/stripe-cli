package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	victimFile := filepath.Join(tmpDir, "victim.txt")
	linkPath := filepath.Join(tmpDir, ".env")

	require.NoError(t, os.WriteFile(victimFile, []byte("original"), 0o644))
	require.NoError(t, os.Symlink(victimFile, linkPath))

	assert.True(t, IsSymlink(afero.NewOsFs(), linkPath))
}

func TestRefuseWriteThroughSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	victimFile := filepath.Join(tmpDir, "victim.txt")
	linkPath := filepath.Join(tmpDir, ".env")

	require.NoError(t, os.WriteFile(victimFile, []byte("original"), 0o644))
	require.NoError(t, os.Symlink(victimFile, linkPath))

	err := RefuseWriteThroughSymlink(afero.NewOsFs(), linkPath, ".env")
	require.Error(t, err)
	assert.Equal(t, "refusing to write .env: "+linkPath+" is a symlink", err.Error())
}
