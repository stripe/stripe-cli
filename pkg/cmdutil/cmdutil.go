// Package cmdutil provides generic cobra command utilities shared across
// pkg/cmd and its subpackages.
package cmdutil

import "github.com/spf13/cobra"

// ArgsAfter returns a copy of args strictly after the first occurrence of name,
// or an empty slice if name is absent. It is used to recover the arguments meant
// for a plugin binary from the raw process arguments, since Cobra consumes the
// tokens it recognizes before the command's RunE sees them.
func ArgsAfter(args []string, name string) []string {
	for i, arg := range args {
		if arg == name {
			rest := args[i+1:]
			out := make([]string, len(rest))
			copy(out, rest)
			return out
		}
	}
	return make([]string, 0)
}

// FindSubCmd walks cmd's subcommand tree following names in order, returning
// the matching command and true. Returns nil and false if any name in the
// path is not found.
func FindSubCmd(cmd *cobra.Command, names ...string) (*cobra.Command, bool) {
	if len(names) == 0 {
		return cmd, true
	}
	// cobra.Find never returns a non-nil error; not-found is signaled by a
	// non-empty remaining-args slice. It also returns the closest matching
	// ancestor rather than nil on a miss. Normalize both here.
	found, remaining, _ := cmd.Find(names)
	if len(remaining) > 0 {
		return nil, false
	}
	return found, true
}
