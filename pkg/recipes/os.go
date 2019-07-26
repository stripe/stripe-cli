package recipes

import (
	"fmt"
	"os"
	"path/filepath"
)

func (r *Recipes) cacheFolder() (string, error) {
	configPath := r.Config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	cachePath := filepath.Join(configPath, "cache")

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		err := os.MkdirAll(cachePath, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	return cachePath, nil
}

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
