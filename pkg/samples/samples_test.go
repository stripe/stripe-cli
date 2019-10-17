package samples

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
	json := `{
  "name": "foo",
  "integrations": [
	  {
		  "name": "webhooks",
		  "clients": ["html"],
		  "servers": ["node", "python", "ruby"]
	  },
	  {
		  "name": "no-webhooks",
		  "clients": ["html"],
		  "servers": ["node", "python", "ruby"]
	  }
  ]
}`

	afero.WriteFile(mg.fs, filepath.Join(appCachePath, ".cli.json"), []byte(json), os.ModePerm)

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

	sample := Samples{
		Fs: fs,
		Git: mockGit{
			fs: fs,
		},
	}

	err := sample.Initialize(name)
	assert.Nil(t, err)
	assert.ElementsMatch(t, sample.sampleConfig.integrationNames(), []string{"webhooks", "no-webhooks"})
	assert.ElementsMatch(t, sample.sampleConfig.integrationServers("webhooks"), []string{"node", "python", "ruby"})
}

func TestDestinationNameEmpty(t *testing.T) {
	sample := Samples{
		integration: []string{"webhooks"},
		sampleConfig: sampleConfig{
			Integrations: []integration{},
		},
	}

	assert.Equal(t, "", sample.destinationName(0))
}

func TestDestinationNameAll(t *testing.T) {
	sample := Samples{
		integration: []string{"all"},
		sampleConfig: sampleConfig{
			Integrations: []integration{
				integration{Name: "webhooks"},
				integration{Name: "non-webhooks"},
			},
		},
	}

	assert.Equal(t, "webhooks", sample.destinationName(0))
	assert.Equal(t, "non-webhooks", sample.destinationName(1))
}

func TestDestinationName(t *testing.T) {
	sample := Samples{
		integration: []string{"webhooks"},
		sampleConfig: sampleConfig{
			Integrations: []integration{
				integration{Name: "webhooks"},
				integration{Name: "non-webhooks"},
			},
		},
	}

	assert.Equal(t, "webhooks", sample.destinationName(0))
}

func TestDestinationPathWithIntegration(t *testing.T) {
	sample := Samples{
		integration: []string{"bender", "fry"},
	}

	assert.Equal(t, "planet-express/robots/bender", sample.destinationPath("planet-express", "robots", "bender"))
}

func TestDestinationPath(t *testing.T) {
	sample := Samples{
		integration: []string{"bender"},
	}

	assert.Equal(t, "planet-express/bender", sample.destinationPath("planet-express", "robots", "bender"))
}
