package recipes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type mockGit struct {
	fs afero.Fs
}

func (mg mockGit) Clone(appCachePath, _ string) error {
	makeRecipe(mg.fs, appCachePath, []string{"webhooks", "no-webhooks"}, []string{"node", "python", "ruby"})

	return nil
}

func (mg mockGit) Pull(appCachePath string) error {
	return nil
}

func makeRecipe(fs afero.Fs, path string, integrations []string, languages []string) {
	for _, integration := range integrations {
		for _, language := range languages {
			fs.MkdirAll(filepath.Join(path, integration, "server", language), os.ModePerm)
			fs.MkdirAll(filepath.Join(path, integration, "client", language), os.ModePerm)
		}
	}
}

func TestInitialize(t *testing.T) {
	fs := afero.NewMemMapFs()
	name := "adding-sales-tax"

	recipe := Recipes{
		Fs: fs,
		Git: mockGit{
			fs: fs,
		},
	}

	err := recipe.Initialize(name)
	assert.Nil(t, err)
	assert.ElementsMatch(t, recipe.integrations, []string{"webhooks", "no-webhooks"})
	assert.ElementsMatch(t, recipe.languages, []string{"node", "python", "ruby"})
}

func TestDestinationNameEmpty(t *testing.T) {
	recipe := Recipes{
		integration:  []string{"webhooks"},
		integrations: []string{},
	}

	assert.Equal(t, "", recipe.destinationName(0))
}

func TestDestinationNameAll(t *testing.T) {
	recipe := Recipes{
		integration:   []string{"all"},
		integrations:  []string{"webhooks", "non-webhooks"},
		isIntegration: true,
	}

	assert.Equal(t, "webhooks", recipe.destinationName(0))
	assert.Equal(t, "non-webhooks", recipe.destinationName(1))
}

func TestDestinationName(t *testing.T) {
	recipe := Recipes{
		integration:   []string{"webhooks"},
		integrations:  []string{"webhooks", "non-webhooks"},
		isIntegration: true,
	}

	assert.Equal(t, "webhooks", recipe.destinationName(0))
}

func TestDestinationPathWithIntegration(t *testing.T) {
	recipe := Recipes{
		integration: []string{"bender", "fry"},
	}

	assert.Equal(t, "planet-express/robots/bender", recipe.destinationPath("planet-express", "robots", "bender"))
}

func TestDestinationPath(t *testing.T) {
	recipe := Recipes{
		integration: []string{"bender"},
	}

	assert.Equal(t, "planet-express/bender", recipe.destinationPath("planet-express", "robots", "bender"))
}
