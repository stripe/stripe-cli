//go:build !wasm
// +build !wasm

package stripe

func getCLIUnixSocket() string {
	return "~/.stripeproxy"
}
