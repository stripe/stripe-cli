package samples

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type mockGit struct {
	fs afero.Fs
}

func (mg *mockGit) Clone(appCachePath, _ string) error {
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

func (mg *mockGit) Pull(appCachePath string) error {
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

type testSampleLister struct {
	data map[string]*SampleData
}

func (l testSampleLister) ListSamples(mode string) (map[string]*SampleData, error) {
	return l.data, nil
}
func TestInitialize(t *testing.T) {
	fs := afero.NewMemMapFs()
	name := "accept-a-payment"

	sampleManager := SampleManager{
		Fs: fs,
		Git: &mockGit{
			fs: fs,
		},
		SampleLister: testSampleLister{
			data: map[string]*SampleData{
				"accept-a-payment": {
					Name:        "accept-a-payment",
					Description: "Learn how to accept a payment",
					URL:         "https://github.com/stripe-samples/accept-a-payment",
				}},
		},
	}

	err := sampleManager.Initialize(name)
	assert.Nil(t, err)
	assert.ElementsMatch(t, sampleManager.SampleConfig.IntegrationNames(), []string{"webhooks", "no-webhooks"})
	assert.ElementsMatch(t, sampleManager.SampleConfig.integrationServers("webhooks"), []string{"node", "python", "ruby"})
}

func TestInitializeFailsWithEmptyName(t *testing.T) {
	fs := afero.NewMemMapFs()
	name := ""

	sampleManager := SampleManager{
		Fs: fs,
		Git: &mockGit{
			fs: fs,
		},
		SampleLister: testSampleLister{map[string]*SampleData{
			"accept-a-payment": {
				Name:        "accept-a-payment",
				Description: "Learn how to accept a payment",
				URL:         "https://github.com/stripe-samples/accept-a-payment",
			},
		}},
	}

	err := sampleManager.Initialize(name)
	assert.Equal(t, errors.New("Sample name is empty"), err)
}

func TestInitializeFailsWithNonexistentSample(t *testing.T) {
	fs := afero.NewMemMapFs()
	name := "foo"

	sampleManager := SampleManager{
		Fs: fs,
		Git: &mockGit{
			fs: fs,
		},
		SampleLister: testSampleLister{map[string]*SampleData{
			"accept-a-payment": {
				Name:        "accept-a-payment",
				Description: "Learn how to accept a payment",
				URL:         "https://github.com/stripe-samples/accept-a-payment",
			},
		}},
	}

	err := sampleManager.Initialize(name)
	assert.Equal(t, errors.New("Sample foo does not exist"), err)
}
