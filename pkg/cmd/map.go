package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// mapMode represents the output format for the --map flag.
type mapMode string

const (
	mapModeDefault    mapMode = ""            // flag not passed
	mapModeTree       mapMode = "tree"        // bare --map or --map=tree
	mapModeCompact    mapMode = "compact"     // tree without descriptions
	mapModePaths      mapMode = "paths"       // flat list of full command paths
	mapModeJSON       mapMode = "json"        // machine-readable JSON tree
	mapModeJSONManual mapMode = "json-manual" // JSON tree of hand-authored commands only, with Long/Example/Flags
)

// generatedAnnotationValues are the Annotations values that pkg/cmd/resource
// sets on a parent command's Annotations map to tag a child as an
// auto-generated (OpenAPI-derived) namespace, resource, or operation command.
var generatedAnnotationValues = map[string]bool{
	"namespace": true,
	"resource":  true,
	"operation": true,
}

// isGeneratedCommand reports whether child is an auto-generated
// (OpenAPI-derived) command, as tagged on parent's Annotations map.
func isGeneratedCommand(parent, child *cobra.Command) bool {
	kind, ok := parent.Annotations[child.Name()]
	return ok && generatedAnnotationValues[kind]
}

// printCommandMap prints a sitemap of all available subcommands rooted at cmd,
// using the specified output mode.
func printCommandMap(w io.Writer, cmd *cobra.Command, mode mapMode) {
	switch mode {
	case mapModePaths:
		printCommandPaths(w, cmd)
	case mapModeJSON:
		printCommandJSON(w, cmd, false)
	case mapModeJSONManual:
		printCommandJSON(w, cmd, true)
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
	Long     string        `json:"long,omitempty"`
	Examples []string      `json:"examples,omitempty"`
	Flags    []flagNode    `json:"flags,omitempty"`
	Commands []commandNode `json:"commands,omitempty"`
}

// splitExamples splits a cobra Command.Example string into individual
// example entries. Example strings in this repo follow three conventions,
// applied here in a single pass:
//   - blank lines separate one example from the next
//   - a line starting with "#" is a comment describing the example that
//     follows it, and is prefixed onto that example
//   - a line ending in "\" continues onto the next line (shell-style),
//     joined with a space, until a line without a trailing "\" is found
func splitExamples(example string) []string {
	var examples []string
	var comment []string
	var command string

	flush := func() {
		if command == "" {
			return
		}
		if len(comment) > 0 {
			examples = append(examples, strings.Join(comment, "\n")+"\n"+command)
		} else {
			examples = append(examples, command)
		}
		comment = nil
		command = ""
	}

	for _, line := range strings.Split(example, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") && command == "" {
			comment = append(comment, trimmed)
			continue
		}

		continued := strings.HasSuffix(trimmed, "\\")
		content := strings.TrimSpace(strings.TrimSuffix(trimmed, "\\"))
		if command == "" {
			command = content
		} else {
			command = command + " " + content
		}
		if continued {
			continue
		}

		flush()
	}
	flush()

	return examples
}

// flagNode is the JSON structure for a single flag defined on a command.
type flagNode struct {
	Name       string `json:"name"`
	Shorthand  string `json:"shorthand,omitempty"`
	Usage      string `json:"usage,omitempty"`
	Default    string `json:"default,omitempty"`
	Deprecated string `json:"deprecated,omitempty"`
}

// buildFlagNodes returns the flags defined directly on cmd (excluding those
// inherited from parent commands and those marked hidden).
func buildFlagNodes(cmd *cobra.Command) []flagNode {
	var flags []flagNode
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		flags = append(flags, flagNode{
			Name:       f.Name,
			Shorthand:  f.Shorthand,
			Usage:      f.Usage,
			Default:    f.DefValue,
			Deprecated: f.Deprecated,
		})
	})
	return flags
}

// buildCommandNode recursively builds a commandNode tree from a cobra command.
// When manual is true, Long, Example, and local Flags are also included, and
// auto-generated (OpenAPI-derived) namespace/resource/operation commands are
// skipped so only hand-authored commands remain.
func buildCommandNode(cmd *cobra.Command, manual bool) commandNode {
	node := commandNode{
		Name: cmd.Name(),
		Desc: cmd.Short,
	}
	if manual {
		node.Long = cmd.Long
		node.Examples = splitExamples(cmd.Example)
		node.Flags = buildFlagNodes(cmd)
	}
	for _, child := range getVisibleCommands(cmd) {
		if manual && isGeneratedCommand(cmd, child) {
			continue
		}
		node.Commands = append(node.Commands, buildCommandNode(child, manual))
	}
	return node
}

// printCommandJSON outputs the command tree as indented JSON. When manual is
// true, only hand-authored commands are included, each with its Long
// description, Example, and local Flags.
func printCommandJSON(w io.Writer, cmd *cobra.Command, manual bool) {
	node := buildCommandNode(cmd, manual)
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
	case "json-manual":
		return mapModeJSONManual, true
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
			if strings.HasPrefix(a, "--map=") {
				val := a[len("--map="):]
				fmt.Fprintf(mapStderr(), "Unknown --map mode %q. Valid modes: tree, compact, paths, json, json-manual\n", val)
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
