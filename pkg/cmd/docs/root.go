package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RootCommand is the root command for the docs plugin.
type RootCommand struct {
	cmd     *cobra.Command
	version string
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
	r.cmd.AddCommand(&cobra.Command{
		Use:                "version",
		Short:              "Print the docs plugin version",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "stripe docs version %s\n", r.version)
		},
	})
	return r
}

// WithVersion sets the plugin version displayed by the version subcommand.
func WithVersion(v string) Option {
	return func(r *RootCommand) {
		r.version = v
	}
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
