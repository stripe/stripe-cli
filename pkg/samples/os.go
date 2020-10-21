package samples

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// cacheFolder is the local directory where we place local copies of samples
func (s *Samples) cacheFolder() (string, error) {
	configPath := s.Config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	cachePath := filepath.Join(configPath, "samples-cache")

	if _, err := s.Fs.Stat(cachePath); os.IsNotExist(err) {
		err := s.Fs.MkdirAll(cachePath, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	return cachePath, nil
}

// appCacheFolder returns the full path of the local cache with the recipe name
func (s *Samples) appCacheFolder(app string) (string, error) {
	path, err := s.cacheFolder()
	if err != nil {
		return "", err
	}

	appPath := filepath.Join(path, app)

	return appPath, nil
}

// MakeFolder creates the folder that'll contain the Stripe app the user is creating
func (s *Samples) MakeFolder(name string) (string, error) {
	appFolder, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	if _, err := s.Fs.Stat(appFolder); os.IsNotExist(err) {
		err = s.Fs.MkdirAll(appFolder, os.ModePerm)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("Path already exists, aborting: %s", appFolder)
	}

	return appFolder, nil
}

// GetFolders returns a list of all folders for a given path
func (s *Samples) GetFolders(path string) ([]string, error) {
	var dir []string

	files, err := afero.ReadDir(s.Fs, path)
	if err != nil {
		return []string{}, err
	}

	for _, file := range files {
		// We only want directories that are not hidden
		if file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
			dir = append(dir, file.Name())
		}
	}

	return dir, nil
}

// GetFiles returns a list of files for a given path
func (s *Samples) GetFiles(path string) ([]string, error) {
	var file []string

	files, err := afero.ReadDir(s.Fs, path)
	if err != nil {
		return []string{}, err
	}

	for _, f := range files {
		// We only want files
		if !f.IsDir() {
			file = append(file, f.Name())
		}
	}

	return file, nil
}

func (s *Samples) delete(name string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	appFolder := filepath.Join(dir, name)
	if exists, _ := afero.Exists(s.Fs, appFolder); exists {
		return s.Fs.RemoveAll(appFolder)
	}

	return nil
}

func folderSearch(folders []string, name string) bool {
	for _, folder := range folders {
		if folder == name {
			return true
		}
	}

	return false
}
