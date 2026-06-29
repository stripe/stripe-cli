// Package open provides utilities for opening URLs in a browser.
package open

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	execabs "golang.org/x/sys/execabs"
)

var execCommand = execabs.Command

// StartCommand starts an exec.Cmd. Overridden in tests to avoid launching a real browser.
var StartCommand = func(cmd *exec.Cmd) error {
	return cmd.Start()
}

// Browser takes a url and opens it using the default browser on the operating system
func Browser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = execCommand("xdg-open", url).Start()
	case "windows":
		err = execCommand("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = execCommand("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		return err
	}

	return nil
}

// CanOpenBrowser determines if no browser is set in linux
func CanOpenBrowser() bool {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return true
	}

	output, err := execCommand("xdg-settings", "get", "default-web-browser").Output()

	if err != nil {
		return false
	}

	if string(output) == "" {
		return false
	}

	return true
}

// OpenURL opens the given URL in the user's default browser.
// allowedHosts restricts which hostnames may be opened; pass nil to allow any https URL.
// The URL must use the https scheme.
func OpenURL(ctx context.Context, u *url.URL, allowedHosts map[string]bool) error {
	if u == nil {
		return fmt.Errorf("nil URL")
	}
	if u.Scheme != "https" || (allowedHosts != nil && !allowedHosts[u.Host]) {
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
