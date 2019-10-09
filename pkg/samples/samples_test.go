package samples

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	sample := Samples{
		Fs: fs,
		Git: mockGit{
			fs: fs,
		},
	}

	err := sample.Initialize(name)
	assert.Nil(t, err)
	assert.ElementsMatch(t, sample.integrations, []string{"webhooks", "no-webhooks"})
	assert.ElementsMatch(t, sample.languages, []string{"node", "python", "ruby"})
}

func TestDestinationNameEmpty(t *testing.T) {
	sample := Samples{
		integration:  []string{"webhooks"},
		integrations: []string{},
	}

	assert.Equal(t, "", sample.destinationName(0))
}

func TestDestinationNameAll(t *testing.T) {
	sample := Samples{
		integration:   []string{"all"},
		integrations:  []string{"webhooks", "non-webhooks"},
		isIntegration: true,
	}

	assert.Equal(t, "webhooks", sample.destinationName(0))
	assert.Equal(t, "non-webhooks", sample.destinationName(1))
}

func TestDestinationName(t *testing.T) {
	sample := Samples{
		integration:   []string{"webhooks"},
		integrations:  []string{"webhooks", "non-webhooks"},
		isIntegration: true,
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

func TestFindServerFiles(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a fake sample
	fs.MkdirAll("/user/bender/adding-sales-tax/server", os.ModePerm)
	fs.Create("/user/bender/adding-sales-tax/server/app.rb")
	fs.Create("/user/bender/adding-sales-tax/server/README.md")
	fs.MkdirAll("/user/bender/adding-sales-tax/client", os.ModePerm)
	fs.Create("/user/bender/adding-sales-tax/client/README.md")
	fs.Create("/user/bender/adding-sales-tax/client/frontend.js")

	// Let's give bender some user files to make sure we don't do anything we're not supposed to
	fs.MkdirAll("/user/bender/code/planex/server", os.ModePerm)
	fs.Create("/user/bender/code/planex/server/app.py")
	fs.MkdirAll("/user/bender/code/", os.ModePerm)

	expected := []string{"/user/bender/adding-sales-tax/server/README.md", "/user/bender/adding-sales-tax/server/app.rb"}

	sample := Samples{
		Fs: fs,
	}
	files, err := sample.findServerFiles("/user/bender/adding-sales-tax")
	require.Nil(t, err)
	require.Equal(t, expected, files)
}

func TestFileWhiteList(t *testing.T) {
	assert.True(t, fileWhitelist("/server/app.java"))
	assert.True(t, fileWhitelist("/server/app.js"))
	assert.True(t, fileWhitelist("/server/app.php"))
	assert.True(t, fileWhitelist("/server/app.rb"))
	assert.False(t, fileWhitelist("/server/app.py"))
	assert.False(t, fileWhitelist("/server/.env"))
	assert.False(t, fileWhitelist("/server/.htaccess"))
	assert.False(t, fileWhitelist("/server/img.png"))
	assert.False(t, fileWhitelist("/server/package.json"))
	assert.False(t, fileWhitelist("/server/Gemfile"))
	assert.False(t, fileWhitelist("/server/pom.xml"))
	assert.False(t, fileWhitelist("/server/config.lock"))
	assert.False(t, fileWhitelist("/server/requirements.txt"))
}

func TestPointToDotEnvWithOneIntegrationRb(t *testing.T) {
	fs := afero.NewMemMapFs()

	file := []byte(`ENV_PATH = '/../../../.env'.freeze`)

	// Create a fake sample
	fs.MkdirAll("/user/bender/adding-sales-tax/server", os.ModePerm)
	afero.WriteFile(fs, "/user/bender/adding-sales-tax/server/app.rb", file, os.ModePerm)

	sample := Samples{
		Fs:            fs,
		integration:   []string{"planex"},
		isIntegration: true,
	}
	err := sample.PointToDotEnv("/user/bender/adding-sales-tax")
	require.Nil(t, err)

	data, _ := afero.ReadFile(fs, "/user/bender/adding-sales-tax/server/app.rb")
	expected := []byte(`ENV_PATH = '../.env'.freeze`)
	assert.Equal(t, string(expected), string(data))
}

func TestPointToDotEnvWithNoIntegrationRb(t *testing.T) {
	fs := afero.NewMemMapFs()

	file := []byte(`ENV_PATH = '/../../../.env'.freeze`)

	// Create a fake sample
	fs.MkdirAll("/user/bender/adding-sales-tax/server", os.ModePerm)
	afero.WriteFile(fs, "/user/bender/adding-sales-tax/server/app.rb", file, os.ModePerm)

	sample := Samples{
		Fs:            fs,
		integration:   []string{},
		isIntegration: false,
	}
	err := sample.PointToDotEnv("/user/bender/adding-sales-tax")
	require.Nil(t, err)

	data, _ := afero.ReadFile(fs, "/user/bender/adding-sales-tax/server/app.rb")
	expected := []byte(`ENV_PATH = '../.env'.freeze`)

	assert.Equal(t, expected, data)
}

func TestPointToDotEnvWithMultipleIntegrationRb(t *testing.T) {
	fs := afero.NewMemMapFs()

	file := []byte(`ENV_PATH = '/../../../.env'.freeze`)

	// Create a fake sample
	fs.MkdirAll("/user/bender/adding-sales-tax/server", os.ModePerm)
	afero.WriteFile(fs, "/user/bender/adding-sales-tax/server/app.rb", file, os.ModePerm)

	sample := Samples{
		Fs:            fs,
		integration:   []string{"planex", "planex2"},
		isIntegration: true,
	}
	err := sample.PointToDotEnv("/user/bender/adding-sales-tax")

	data, _ := afero.ReadFile(fs, "/user/bender/adding-sales-tax/server/app.rb")
	expected := []byte(`ENV_PATH = '../../.env'.freeze`)

	require.Nil(t, err)
	assert.Equal(t, expected, data)
}

func TestPointToDotEnvWithMultipleIntegrationJava(t *testing.T) {
	fs := afero.NewMemMapFs()

	file := []byte(`String ENV_PATH = "../../..";`)

	// Create a fake sample
	fs.MkdirAll("/user/bender/adding-sales-tax/server", os.ModePerm)
	afero.WriteFile(fs, "/user/bender/adding-sales-tax/server/app.java", file, os.ModePerm)

	sample := Samples{
		Fs:            fs,
		integration:   []string{"planex", "planex2"},
		isIntegration: true,
	}
	err := sample.PointToDotEnv("/user/bender/adding-sales-tax")

	data, _ := afero.ReadFile(fs, "/user/bender/adding-sales-tax/server/app.java")
	expected := []byte(`String ENV_PATH = "../../";`)

	require.Nil(t, err)
	assert.Equal(t, expected, data)
}

func TestPointToDotEnvWithMultipleIntegrationPhp(t *testing.T) {
	fs := afero.NewMemMapFs()

	file := []byte(`$ENV_PATH = '../../..';`)

	// Create a fake sample
	fs.MkdirAll("/user/bender/adding-sales-tax/server", os.ModePerm)
	afero.WriteFile(fs, "/user/bender/adding-sales-tax/server/app.php", file, os.ModePerm)

	sample := Samples{
		Fs:            fs,
		integration:   []string{"planex", "planex2"},
		isIntegration: true,
	}
	err := sample.PointToDotEnv("/user/bender/adding-sales-tax")
	require.Nil(t, err)

	data, _ := afero.ReadFile(fs, "/user/bender/adding-sales-tax/server/app.php")
	expected := []byte(`$ENV_PATH = '../..';`)

	assert.Equal(t, expected, data)
}

func TestPointToDotEnvWithMultipleIntegrationJs(t *testing.T) {
	fs := afero.NewMemMapFs()

	file := []byte(`const envPath = resolve(__dirname, "../../../.env");`)

	// Create a fake sample
	fs.MkdirAll("/user/bender/adding-sales-tax/server", os.ModePerm)
	afero.WriteFile(fs, "/user/bender/adding-sales-tax/server/app.js", file, os.ModePerm)

	sample := Samples{
		Fs:            fs,
		integration:   []string{"planex", "planex2"},
		isIntegration: true,
	}
	err := sample.PointToDotEnv("/user/bender/adding-sales-tax")
	require.Nil(t, err)

	data, _ := afero.ReadFile(fs, "/user/bender/adding-sales-tax/server/app.js")
	expected := []byte(`const envPath = resolve(__dirname, "../../.env");`)

	assert.Equal(t, string(expected), string(data))
}
