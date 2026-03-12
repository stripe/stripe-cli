package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// printCommandMap prints a tree-view "sitemap" of all available subcommands
// rooted at cmd. The root line is printed in bold, and descriptions are faint.
func printCommandMap(w io.Writer, cmd *cobra.Command) {
	color := ansi.Color(w)
	fmt.Fprintln(w, color.Sprintf(color.Bold(cmd.CommandPath())))

	children := getVisibleCommands(cmd)
	for i, child := range children {
		isLast := i == len(children)-1
		printCommandTree(w, child, "", isLast)
	}
}

// printCommandTree recursively prints a command and its children using
// box-drawing characters for visual nesting.
func printCommandTree(w io.Writer, cmd *cobra.Command, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	color := ansi.Color(w)
	name := cmd.Name()
	desc := cmd.Short

	line := prefix + connector + name
	if desc != "" {
		line += "  " + color.Sprintf(color.Faint(desc))
	}
	fmt.Fprintln(w, line)

	children := getVisibleCommands(cmd)
	childPrefix := prefix + "│   "
	if isLast {
		childPrefix = prefix + "    "
	}

	for i, child := range children {
		childIsLast := i == len(children)-1
		printCommandTree(w, child, childPrefix, childIsLast)
	}
}

// isMapFlag returns true if the argument is any form of the --map flag
// (i.e. "--map", "--map=true", "--map=false").
func isMapFlag(arg string) bool {
	return arg == "--map" || strings.HasPrefix(arg, "--map=")
}

// isMapEnabled returns true if the argument enables --map. The forms
// "--map" and "--map=true" enable it; "--map=false" does not.
func isMapEnabled(arg string) bool {
	if arg == "--map" {
		return true
	}
	if strings.HasPrefix(arg, "--map=") {
		return arg[len("--map="):] != "false"
	}
	return false
}

// hasMapFlag returns true if a --map flag that enables the feature
// appears in the argument list.
func hasMapFlag(args []string) bool {
	for _, a := range args {
		if isMapEnabled(a) {
			return true
		}
		if a == "--" {
			return false
		}
	}
	return false
}

// stripMapFlag returns a copy of args with all --map flag forms removed,
// so that rootCmd.Find can resolve the target command without the flag.
func stripMapFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if !isMapFlag(a) {
			out = append(out, a)
		}
	}
	return out
}

// getVisibleCommands returns the subset of cmd's subcommands that are not
// hidden, not deprecated, and not the auto-generated "help" command.
func getVisibleCommands(cmd *cobra.Command) []*cobra.Command {
	var visible []*cobra.Command
	for _, c := range cmd.Commands() {
		if c.Hidden || len(c.Deprecated) > 0 || strings.EqualFold(c.Name(), "help") {
			continue
		}
		visible = append(visible, c)
	}
	return visible
}
