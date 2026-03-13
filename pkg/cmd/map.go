package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// mapMode represents the output format for the --map flag.
type mapMode string

const (
	mapModeDefault mapMode = ""        // flag not passed
	mapModeTree    mapMode = "tree"    // bare --map or --map=tree
	mapModeCompact mapMode = "compact" // tree without descriptions
	mapModePaths   mapMode = "paths"   // flat list of full command paths
	mapModeJSON    mapMode = "json"    // machine-readable JSON tree
)

// printCommandMap prints a sitemap of all available subcommands rooted at cmd,
// using the specified output mode.
func printCommandMap(w io.Writer, cmd *cobra.Command, mode mapMode) {
	switch mode {
	case mapModePaths:
		printCommandPaths(w, cmd)
	case mapModeJSON:
		printCommandJSON(w, cmd)
	default: // mapModeTree, mapModeCompact
		color := ansi.Color(w)
		fmt.Fprintln(w, color.Sprintf(color.Bold(cmd.CommandPath())))

		children := getVisibleCommands(cmd)
		for i, child := range children {
			isLast := i == len(children)-1
			printCommandTree(w, child, "", isLast, mode)
		}
	}
}

// printCommandTree recursively prints a command and its children using
// box-drawing characters for visual nesting.
func printCommandTree(w io.Writer, cmd *cobra.Command, prefix string, isLast bool, mode mapMode) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	color := ansi.Color(w)
	name := cmd.Name()

	line := prefix + connector + name
	if mode != mapModeCompact {
		desc := cmd.Short
		if desc != "" {
			line += "  " + color.Sprintf(color.Faint(desc))
		}
	}
	fmt.Fprintln(w, line)

	children := getVisibleCommands(cmd)
	childPrefix := prefix + "│   "
	if isLast {
		childPrefix = prefix + "    "
	}

	for i, child := range children {
		childIsLast := i == len(children)-1
		printCommandTree(w, child, childPrefix, childIsLast, mode)
	}
}

// printCommandPaths prints a flat list of full command paths, one per line,
// for every visible leaf command.
func printCommandPaths(w io.Writer, cmd *cobra.Command) {
	children := getVisibleCommands(cmd)
	if len(children) == 0 {
		fmt.Fprintln(w, cmd.CommandPath())
		return
	}
	for _, child := range children {
		printCommandPaths(w, child)
	}
}

// commandNode is the JSON structure for a command in the tree.
type commandNode struct {
	Name     string        `json:"name"`
	Desc     string        `json:"desc,omitempty"`
	Commands []commandNode `json:"commands,omitempty"`
}

// buildCommandNode recursively builds a commandNode tree from a cobra command.
func buildCommandNode(cmd *cobra.Command) commandNode {
	node := commandNode{
		Name: cmd.Name(),
		Desc: cmd.Short,
	}
	children := getVisibleCommands(cmd)
	if len(children) > 0 {
		node.Commands = make([]commandNode, 0, len(children))
		for _, child := range children {
			node.Commands = append(node.Commands, buildCommandNode(child))
		}
	}
	return node
}

// printCommandJSON outputs the command tree as indented JSON.
func printCommandJSON(w io.Writer, cmd *cobra.Command) {
	node := buildCommandNode(cmd)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(node) //nolint:errcheck
}

// parseMapMode parses a single --map argument and returns the mode and
// whether the map feature is enabled. An unrecognized value returns
// (mapModeDefault, false).
func parseMapMode(arg string) (mapMode, bool) {
	if arg == "--map" {
		return mapModeTree, true
	}
	if !strings.HasPrefix(arg, "--map=") {
		return mapModeDefault, false
	}
	val := arg[len("--map="):]
	switch val {
	case "tree":
		return mapModeTree, true
	case "compact":
		return mapModeCompact, true
	case "paths":
		return mapModePaths, true
	case "json":
		return mapModeJSON, true
	case "true":
		return mapModeTree, true
	case "false":
		return mapModeDefault, false
	default:
		return mapModeDefault, false
	}
}

// stderrOverride can be set in tests to capture error output.
var stderrOverride io.Writer

func mapStderr() io.Writer {
	if stderrOverride != nil {
		return stderrOverride
	}
	return os.Stderr
}

// getMapMode scans args for a --map flag and returns the requested mode.
// Returns mapModeDefault if no --map flag is present or if it is disabled.
// Prints an error to stderr for unrecognized mode values.
func getMapMode(args []string) mapMode {
	for _, a := range args {
		if a == "--" {
			return mapModeDefault
		}
		if a == "--map" || strings.HasPrefix(a, "--map=") {
			mode, ok := parseMapMode(a)
			if ok {
				return mode
			}
			// parseMapMode returns false for --map=false (backward compat)
			// and for unknown values. Distinguish the two:
			if strings.HasPrefix(a, "--map=") {
				val := a[len("--map="):]
				if val != "false" {
					fmt.Fprintf(mapStderr(), "Unknown --map mode %q. Valid modes: tree, compact, paths, json\n", val)
				}
			}
			return mapModeDefault
		}
	}
	return mapModeDefault
}

// isMapFlag returns true if the argument is any form of the --map flag
// (i.e. "--map", "--map=tree", "--map=compact", etc.).
func isMapFlag(arg string) bool {
	return arg == "--map" || strings.HasPrefix(arg, "--map=")
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
