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
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  stripe projects catalog    Browse available services")
		fmt.Fprintln(out, "  stripe projects init       Set up a new project")
		fmt.Fprintln(out, "  stripe projects --help     See all commands")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Docs:     https://docs.stripe.com/projects")
		fmt.Fprintln(out, "Site:     https://projects.dev")
		fmt.Fprintln(out, "Feedback: projects-feedback@stripe.com")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "For AI coding agents:")
		fmt.Fprintln(out, "  npx skills add https://github.com/stripe/ai --skill stripe-projects -y -g")
	}
}
