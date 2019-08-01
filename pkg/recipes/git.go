package recipes

import (
	"gopkg.in/src-d/go-git.v4"
)

var recipesList = map[string]string{
	"adding-sales-tax":            "https://github.com/stripe-samples/adding-sales-tax.git",
	"placing-a-hold":              "https://github.com/stripe-samples/placing-a-hold.git",
	"payment-form-model":          "https://github.com/stripe-samples/payment-form-modal.git",
	"saving-card-without-payment": "https://github.com/stripe-samples/saving-card-without-payment.git",
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

	repo.Fetch(&git.FetchOptions{
		Force: true,
	})

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Reset(&git.ResetOptions{
		Mode: git.HardReset,
	})
	if err != nil {
		return nil
	}

	return nil
}
