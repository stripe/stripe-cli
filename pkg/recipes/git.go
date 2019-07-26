package recipes

import (
	"gopkg.in/src-d/go-git.v4"
)

var recipesList = map[string]string{
	"sales-tax":                   "https://github.com/adreyfus-stripe/sales-tax.git",
	"placing-a-hold":              "https://github.com/adreyfus-stripe/placing-a-hold.git",
	"elements-modal":              "https://github.com/ctrudeau-stripe/elements-modal-demo.git",
	"saving-card-without-payment": "https://github.com/ctrudeau-stripe/saving-card-without-payment.git",
	"billing-quickstart":          "https://github.com/ctrudeau-stripe/stripe-billing-quickstart.git",
}

func (r *Recipes) clone(appCachePath, app string) error {
	_, err := git.PlainClone(appCachePath, false, &git.CloneOptions{
		URL: recipesList[app],
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *Recipes) pull(appCachePath, app string) error {
	repo, err := git.PlainOpen(appCachePath)
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Pull(&git.PullOptions{})
	if err != nil {
		return err
	}

	return nil
}
