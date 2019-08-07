package recipes

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// cacheFolder is the local directory where we place local copies of recipes
func (r *Recipes) cacheFolder() (string, error) {
	configPath := r.Config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	cachePath := filepath.Join(configPath, "recipes-cache")

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		err := os.MkdirAll(cachePath, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	return cachePath, nil
}

// appCacheFolder returns the full path of the local cache with the recipe name
func (r *Recipes) appCacheFolder(app string) (string, error) {
	path, err := r.cacheFolder()
	if err != nil {
		return "", err
	}

	appPath := filepath.Join(path, app)

	return appPath, nil
}

// MakeFolder creates the folder that'll contain the Stripe app the user is creating
func (r *Recipes) MakeFolder(name string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	appFolder := filepath.Join(dir, name)
	if _, err := os.Stat(appFolder); os.IsNotExist(err) {
		err = os.Mkdir(appFolder, os.ModePerm)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("Path already exists, aborting: %s", appFolder)
	}

	return appFolder, nil
}

func (r *Recipes) GetFolders(path string) ([]string, error) {
	files, err := ioutil.ReadDir(path)
	var dir []string
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

func (r *Recipes) GetFiles(path string) ([]string, error) {
	files, err := ioutil.ReadDir(path)
	var file []string
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

func folderSearch(folders []string, name string) bool {
	for _, folder := range folders {
		if folder == name {
			return true
		}
	}

	return false
}
