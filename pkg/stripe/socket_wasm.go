//go:build wasm
// +build wasm

package stripe

import "os"

func getCLIUnixSocket() string {
	return os.Getenv("STRIPE_CLI_UNIX_SOCKET")
}
