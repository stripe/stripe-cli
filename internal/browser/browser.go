// Package browser provides utilities for opening URLs in the user's default browser.
package browser

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

var allowedHosts = map[string]bool{
	"docs.stripe.com": true,
}

// StartCommand starts an exec.Cmd. Overridden in tests to avoid launching a real browser.
var StartCommand = func(cmd *exec.Cmd) error {
	return cmd.Start()
}

// Open opens the given URL in the user's default browser.
// The URL must have an https scheme and an allowed host.
func Open(ctx context.Context, u *url.URL) error {
	if u == nil {
		return fmt.Errorf("nil URL")
	}
	if u.Scheme != "https" || !allowedHosts[u.Host] {
		return fmt.Errorf("URL not allowed: %s", u)
	}

	cmd, err := openCommand()
	if err != nil {
		return err
	}

	//nolint:gosec // u is validated against allowedHosts above
	c := exec.CommandContext(ctx, cmd[0], append(cmd[1:], u.String())...)
	if err = StartCommand(c); err != nil {
		return fmt.Errorf("starting browser: %w", err)
	}
	return nil
}

func openCommand() ([]string, error) {
	switch runtime.GOOS {
	case "darwin":
		return []string{"open"}, nil
	case "windows":
		return []string{"rundll32", "url.dll,FileProtocolHandler"}, nil
	case "linux":
		return []string{"xdg-open"}, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
