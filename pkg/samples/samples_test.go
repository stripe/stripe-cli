package samples

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-cli/pkg/config"
)

type mockGit struct {
	fs afero.Fs
}

func mockGetSamples() error {
	samplesJSON := []byte(`{
		"samples": [
			{
				"name": "adding-sales-tax",
				"description": "Learn how to use PaymentIntents to build a simple checkout flow",
				"URL": "https://github.com/stripe-samples/adding-sales-tax"
			}
		]
		}`)

	var allSamples SampleList

	err := json.Unmarshal(samplesJSON, &allSamples)
	if err != nil {
		return err
	}
	for i, sample := range allSamples.Samples {
		list[sample.Name] = &allSamples.Samples[i]
	}

	return nil
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
  ],
  "requiredResources": [
	  {
		  "name": "stripe_samples_price_recurring_basic_id",
		  "envVar": "BASIC_PRICE_ID"
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
	mockGetSamples()

	sample := Samples{
		Fs: fs,
		Git: mockGit{
			fs: fs,
		},
		Config: &config.Config{
			Profile: config.Profile{},
		},
	}

	err := sample.Initialize(name)
	assert.Nil(t, err)
	assert.ElementsMatch(t, sample.sampleConfig.integrationNames(), []string{"webhooks", "no-webhooks"})
	assert.ElementsMatch(t, sample.sampleConfig.integrationServers("webhooks"), []string{"node", "python", "ruby"})
	missing := sample.MissingRequiredResources()
	missingNames := []string{}
	for _, m := range missing {
		missingNames = append(missingNames, m.name)
	}
	assert.ElementsMatch(t, missingNames, []string{"stripe_samples_price_recurring_basic_id"})
}
