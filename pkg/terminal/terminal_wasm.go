//go:build wasm
// +build wasm

package terminal

import (
	"bufio"
	"io"
)

func IsTerminal(w io.Writer) bool {
	return false
}

func GetTerminalWidth() int {
	return 80
}

func SecurePrompt(input io.Reader) (string, error) {
	reader := bufio.NewReader(input)

	return reader.ReadString('\n')
}
