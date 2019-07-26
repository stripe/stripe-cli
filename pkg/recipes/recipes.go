package recipes

import (
	"os"

	"github.com/stripe/stripe-cli/pkg/config"
)

// Recipes does stuff
// TODO
type Recipes struct {
	Config config.Config
}

func (r *Recipes) Download(app string) (string, error) {
	appPath, err := r.appCacheFolder(app)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		err = r.clone(appPath, app)
		if err != nil {
			return "", err
		}
	} else {
		err := r.pull(appPath, app)
		if err != nil {
			return "", err
		}
	}

	return appPath, nil
}
