package samples

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
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
	assert.Equal(t, errors.New("sample name is empty"), err)
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
	assert.Equal(t, errors.New("sample foo does not exist"), err)
}

func TestCopySkipsSymlinks(t *testing.T) {
	// Use the real filesystem because afero.MemMapFs doesn't support symlinks.
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a source layout: server/node/ with a regular file and a symlink.
	serverNodeDir := filepath.Join(srcDir, "server", "node")
	require.NoError(t, os.MkdirAll(serverNodeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(serverNodeDir, "index.js"), []byte("console.log('ok')"), 0o644))

	// Create an external victim file that the symlink targets.
	victimFile := filepath.Join(t.TempDir(), "victim.txt")
	require.NoError(t, os.WriteFile(victimFile, []byte("original"), 0o644))

	// Create a symlink server/node/.env -> victim file.
	require.NoError(t, os.Symlink(victimFile, filepath.Join(serverNodeDir, ".env")))

	sm := &SampleManager{
		Fs:       afero.NewOsFs(),
		repoPath: srcDir,
		SelectedConfig: SelectedConfig{
			Integration: &SampleConfigIntegration{
				Name:    "main",
				Servers: []string{"node"},
			},
			Server: "node",
		},
	}

	err := sm.Copy(dstDir)
	require.NoError(t, err)

	// Regular file should be copied.
	assert.FileExists(t, filepath.Join(dstDir, "server", "index.js"))

	// Symlink .env should NOT be copied.
	_, err = os.Lstat(filepath.Join(dstDir, "server", ".env"))
	assert.True(t, os.IsNotExist(err), "symlink .env should not exist at destination")

	// Victim file should be untouched.
	content, err := os.ReadFile(victimFile)
	require.NoError(t, err)
	assert.Equal(t, "original", string(content))
}

func TestWriteDotEnvRefusesSymlink(t *testing.T) {
	sampleDir := t.TempDir()

	// Create server directory with .env as a symlink.
	serverDir := filepath.Join(sampleDir, "server")
	require.NoError(t, os.MkdirAll(serverDir, 0o755))

	victimFile := filepath.Join(t.TempDir(), "victim.txt")
	require.NoError(t, os.WriteFile(victimFile, []byte("original"), 0o644))

	require.NoError(t, os.Symlink(victimFile, filepath.Join(serverDir, ".env")))

	// Create a .env.example for godotenv.Parse to read.
	require.NoError(t, os.WriteFile(filepath.Join(sampleDir, ".env.example"), []byte("FOO=bar\n"), 0o644))

	sm := &SampleManager{
		Fs: afero.NewOsFs(),
		SampleConfig: SampleConfig{
			ConfigureDotEnv: true,
		},
		SelectedConfig: SelectedConfig{
			Integration: &SampleConfigIntegration{
				Name:    "main",
				Servers: []string{"node"},
			},
			Server: "node",
		},
		ConfigureDotEnv: func(ctx context.Context, cfg *config.Config) (map[string]string, error) {
			return map[string]string{
				"STRIPE_SECRET_KEY":      "sk_test_123",
				"STRIPE_PUBLISHABLE_KEY": "pk_test_123",
				"STRIPE_WEBHOOK_SECRET":  "whsec_test_123",
				"STATIC_DIR":             "../client",
			}, nil
		},
	}

	err := sm.WriteDotEnv(context.Background(), sampleDir)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "symlink"), "error should mention symlink, got: %s", err.Error())

	// Victim file should be untouched.
	content, err := os.ReadFile(victimFile)
	require.NoError(t, err)
	assert.Equal(t, "original", string(content))
}

func TestWriteDotEnvWorksForRegularFile(t *testing.T) {
	sampleDir := t.TempDir()

	// Create server directory (no symlinks).
	serverDir := filepath.Join(sampleDir, "server")
	require.NoError(t, os.MkdirAll(serverDir, 0o755))

	// Create a .env.example for godotenv.Parse to read.
	require.NoError(t, os.WriteFile(filepath.Join(sampleDir, ".env.example"), []byte("FOO=bar\n"), 0o644))

	sm := &SampleManager{
		Fs: afero.NewOsFs(),
		SampleConfig: SampleConfig{
			ConfigureDotEnv: true,
		},
		SelectedConfig: SelectedConfig{
			Integration: &SampleConfigIntegration{
				Name:    "main",
				Servers: []string{"node"},
			},
			Server: "node",
		},
		ConfigureDotEnv: func(ctx context.Context, cfg *config.Config) (map[string]string, error) {
			return map[string]string{
				"STRIPE_SECRET_KEY":      "sk_test_123",
				"STRIPE_PUBLISHABLE_KEY": "pk_test_123",
				"STRIPE_WEBHOOK_SECRET":  "whsec_test_123",
				"STATIC_DIR":             "../client",
			}, nil
		},
	}

	err := sm.WriteDotEnv(context.Background(), sampleDir)
	require.NoError(t, err)

	// .env should be a regular file with the expected keys.
	envFile := filepath.Join(serverDir, ".env")
	assert.FileExists(t, envFile)

	content, err := os.ReadFile(envFile)
	require.NoError(t, err)
	envContent := string(content)
	assert.Contains(t, envContent, "STRIPE_SECRET_KEY")
	assert.Contains(t, envContent, "STRIPE_PUBLISHABLE_KEY")
	assert.Contains(t, envContent, "STRIPE_WEBHOOK_SECRET")
}
