package samples

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func home() string {
	home, _ := homedir.Dir()
	return home
}

func TestFolderSearch(t *testing.T) {
	folders := []string{"foo", "bar", "baz"}

	expectedFound := folderSearch(folders, "bar")
	expectedNotFound := folderSearch(folders, "box")

	assert.True(t, expectedFound)
	assert.False(t, expectedNotFound)
}

func TestCacheFolder(t *testing.T) {
	fs := afero.NewMemMapFs()
	viper.SetFs(fs)

	sample := Samples{
		Fs: fs,
	}

	expectedPath := filepath.Join(home(), ".config", "stripe", "samples-cache")

	path, _ := sample.cacheFolder()
	pathExists, err := afero.Exists(fs, path)

	assert.Equal(t, expectedPath, path)
	assert.True(t, pathExists)
	assert.Nil(t, err)
}

func TestAppCacheFolder(t *testing.T) {
	fs := afero.NewMemMapFs()
	viper.SetFs(fs)

	sample := Samples{
		Fs: fs,
	}

	expectedPath := filepath.Join(home(), ".config", "stripe", "samples-cache", "bender")

	path, err := sample.appCacheFolder("bender")

	assert.Equal(t, expectedPath, path)
	assert.Nil(t, err)
}

func TestMakeFolder(t *testing.T) {
	fs := afero.NewMemMapFs()
	viper.SetFs(fs)

	sample := Samples{
		Fs: fs,
	}

	wd, _ := os.Getwd()
	expectedPath := filepath.Join(wd, "bender")

	path, err := sample.MakeFolder("bender")
	exists, _ := afero.Exists(fs, path)

	assert.Equal(t, expectedPath, path)
	assert.True(t, exists)
	assert.Nil(t, err)

	absolutePath := filepath.Join(wd, "absolute/path/indeed")
	path, err = sample.MakeFolder(absolutePath)
	exists, _ = afero.Exists(fs, path)
	assert.Equal(t, absolutePath, path)
	assert.True(t, exists)
	assert.Nil(t, err)
}

func TestMakeFolderExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	viper.SetFs(fs)

	sample := Samples{
		Fs: fs,
	}

	wd, _ := os.Getwd()
	preExistingPath := filepath.Join(wd, "bender")
	fs.MkdirAll(preExistingPath, os.ModePerm)

	path, err := sample.MakeFolder("bender")

	assert.Equal(t, "", path)
	assert.EqualError(t, err, fmt.Sprintf("Path already exists, aborting: %s", preExistingPath))
}

func TestGetFolders(t *testing.T) {
	fs := afero.NewMemMapFs()

	fs.Mkdir("bender", os.ModePerm)
	fs.Mkdir("fry", os.ModePerm)
	fs.Mkdir("leela", os.ModePerm)
	fs.Create("zoidberg")

	sample := Samples{
		Fs: fs,
	}
	folders, err := sample.GetFolders("/")

	assert.Nil(t, err)
	assert.ElementsMatch(t, []string{"bender", "fry", "leela"}, folders)
}

func TestGetFiles(t *testing.T) {
	fs := afero.NewMemMapFs()

	fs.Create("bender")
	fs.Create("fry")
	fs.Create("leela")
	fs.Mkdir("zoidberg", os.ModePerm)

	sample := Samples{
		Fs: fs,
	}
	files, err := sample.GetFiles("/")

	assert.Nil(t, err)
	assert.ElementsMatch(t, []string{"bender", "fry", "leela"}, files)
}

func TestFoldersSearch(t *testing.T) {
	folders := []string{"bender", "fry", "leela"}
	assert.True(t, folderSearch(folders, "leela"))
	assert.False(t, folderSearch(folders, "zoidberg"))
}
