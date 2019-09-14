package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestGetLogin(t *testing.T) {
	fs := afero.NewMemMapFs()
	cfg := config.Config{}
	expected := `
Before using the CLI, you'll need to login:

  $ stripe login

If you're working on multiple projects, you can run the login command with the
--project-name flag:

  $ stripe login --project-name rocket-rides`
	output := getLogin(&fs, &cfg)

	assert.Equal(t, expected, output)
}

func TestGetLoginEmpty(t *testing.T) {
	fs := afero.NewMemMapFs()
	cfg := config.Config{}

	file := filepath.Join(cfg.GetConfigFolder(os.Getenv("XDG_CONFIG")), "config.toml")

	afero.WriteFile(fs, file, []byte{}, os.ModePerm)

	output := getLogin(&fs, &cfg)

	assert.Equal(t, "", output)
}
