//go:build wasm
// +build wasm

package cmd

import (
	"github.com/spf13/afero"
	"github.com/stripe/stripe-cli/pkg/config"
)

func getLogin(fs *afero.Fs, cfg *config.Config) string {
	return ""
}

func GetCommandArgs() []string {
	return WasmArgs
}

func GetTopLevelCommand() string {
	return WasmArgs[0]
}
