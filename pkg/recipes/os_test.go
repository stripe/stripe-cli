package recipes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFolderSearch(t *testing.T) {
	folders := []string{"foo", "bar", "baz"}

	expectedFound := folderSearch(folders, "bar")
	expectedNotFound := folderSearch(folders, "box")

	assert.Equal(t, true, expectedFound)
	assert.Equal(t, false, expectedNotFound)
}
