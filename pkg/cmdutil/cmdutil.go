// Package cmdutil provides generic cobra command utilities shared across
// pkg/cmd and its subpackages.
package cmdutil

import "github.com/spf13/cobra"

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
