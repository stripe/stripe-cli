package cmd

import "github.com/spf13/cobra"

// RootCommand is the root command for the docs plugin.
type RootCommand struct {
	cmd *cobra.Command
}

// Option is a functional option for configuring RootCommand.
type Option func(*RootCommand)

// New creates a new RootCommand.
func New() *RootCommand {
	r := &RootCommand{}
	r.cmd = &cobra.Command{
		Use:   "docs",
		Short: "Search, browse, and read docs.stripe.com documentation from the terminal",
		Long:  "A Stripe CLI plugin for searching, browsing, and reading docs.stripe.com from the terminal.",
	}
	return r
}

// WithOptions applies the given options to the RootCommand.
func (r *RootCommand) WithOptions(opts ...Option) *RootCommand {
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Root returns the cobra command, used by tools like the doc generator.
func (r *RootCommand) Root() *cobra.Command { return r.cmd }
