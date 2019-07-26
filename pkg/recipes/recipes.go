package recipes

import "github.com/stripe/stripe-cli/pkg/config"

// Recipes does stuff
// TODO
type Recipes struct {
	Config config.Config
}

func (r *Recipes) Download(app string) {
	r.clone(app)
	// TODO: check if exists, pull new data
}
