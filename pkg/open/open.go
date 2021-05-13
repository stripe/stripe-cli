package open

import (
	"fmt"
	"os/exec"
	"runtime"
)

var execCommand = exec.Command

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
