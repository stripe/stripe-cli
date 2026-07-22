// Package postinstall prints plugin-specific guidance after installation.
package postinstall

import (
	"fmt"
	"io"
)

// PrintTips prints optional next steps after the plugin has been installed.
func PrintTips(out io.Writer, pluginName string) {
	switch pluginName {
	case "directory":
		fmt.Fprintln(out, "Search:  stripe directory search \"your query here\"")
		fmt.Fprintln(out, "Help:    stripe directory --help")
		fmt.Fprintln(out, "Edit your profile: https://dashboard.stripe.com/settings/profile")
		fmt.Fprintln(out, "If you have feedback about this early preview, email directory@stripe.com")
	case "projects":
		fmt.Fprintln(out, "More information: https://projects.dev")
	}
}
