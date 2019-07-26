package recipes

import (
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
