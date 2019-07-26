package recipes

import (
	"path/filepath"

	"gopkg.in/src-d/go-git.v4"
)

var recipesList = map[string]string{
	"sales-tax":                   "https://github.com/adreyfus-stripe/sales-tax.git",
	"placing-a-hold":              "https://github.com/adreyfus-stripe/placing-a-hold.git",
	"elements-modal":              "https://github.com/ctrudeau-stripe/elements-modal-demo.git",
	"saving-card-without-payment": "https://github.com/ctrudeau-stripe/saving-card-without-payment.git",
	"billing-quickstart":          "https://github.com/ctrudeau-stripe/stripe-billing-quickstart.git",
}

func (r *Recipes) clone(app string) error {
	path, err := r.cacheFolder()
	if err != nil {
		return err
	}

	appPath := filepath.Join(path, app)

	git.PlainClone(appPath, false, &git.CloneOptions{
		URL: recipesList[app],
	})

	return nil
}
