//go:build !wasm
// +build !wasm

package cmd

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

func getLogin(fs *afero.Fs, cfg *config.Config) string {
	// We're checking against the path because we don't initialize the config
	// at this point of execution.
	path := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	file := filepath.Join(path, "config.toml")

	exists, _ := afero.Exists(*fs, file)

	if !exists {
		return `
Before using the CLI, you'll need to login:

  $ stripe login

If you're working on multiple projects, you can run the login command with the
--project-name flag:

  $ stripe login --project-name rocket-rides`
	}

	return ""
}

func GetCommandArgs() []string {
	return os.Args
}

func GetTopLevelCommand() string {
	return os.Args[1]
}
