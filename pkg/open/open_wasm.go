//go:build wasm
// +build wasm

package open

import (
	"fmt"
)

// Browser takes a url and opens it using the default browser on the operating system
func Browser(url string) error {
	return fmt.Errorf("unsupported platform")
}

// CanOpenBrowser determines if no browser is set in linux
func CanOpenBrowser() bool {
	return false
}
