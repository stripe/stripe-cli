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
