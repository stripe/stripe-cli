//go:build wasm
// +build wasm

package login

import (
	"context"
	"fmt"
	"io"

	"github.com/stripe/stripe-cli/pkg/config"
)

// Login function is used to obtain credentials via stripe dashboard.
func Login(ctx context.Context, baseURL string, config *config.Config, input io.Reader) error {
	return fmt.Errorf("unsupported platform")
}
