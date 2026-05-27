package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
	"github.com/stripe/stripe-cli-docs-plugin/internal/pager"
	"github.com/stripe/stripe-cli-docs-plugin/markdown"
)

// RootCommand is the root command for the docs plugin.
type RootCommand struct {
	cmd      *cobra.Command
	client   *docs.Client
	renderer markdown.Renderer
	version  string
	noPager  bool
}

// Option is a functional option for configuring RootCommand.
type Option func(*RootCommand)

// WithClient sets the docs HTTP client used to fetch pages.
func WithClient(client *docs.Client) Option {
	return func(r *RootCommand) { r.client = client }
}

// WithRenderer sets the Markdown renderer used to display pages.
func WithRenderer(renderer markdown.Renderer) Option {
	return func(r *RootCommand) { r.renderer = renderer }
}

// New creates a new RootCommand with sensible defaults.
func New() *RootCommand {
	r := &RootCommand{
		client: docs.NewClient("unknown"),
	}
	if renderer, err := markdown.NewRenderer(); err == nil {
		r.renderer = renderer
	}

	r.cmd = &cobra.Command{
		Use:   "docs <path>",
		Short: "Search, browse, and read docs.stripe.com documentation from the terminal",
		Example: `  stripe docs /payments
  stripe docs /connect/accounts
  stripe docs /api/customers`,
		Args:         cobra.ArbitraryArgs,
		RunE:         r.run,
		SilenceUsage: true,
	}
	r.cmd.PersistentFlags().BoolVar(&r.noPager, "no-pager", false, "Do not pipe output through a pager")
	r.cmd.AddCommand(&cobra.Command{
		Use:                "version",
		Short:              "Print the docs plugin version",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "stripe docs version %s\n", r.version)
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

func (r *RootCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	path := args[0]
	if !strings.HasPrefix(path, "/") {
		path = "/" + strings.Join(args, "/")
	}

	w := pager.New(cmd.OutOrStdout(), !r.noPager)
	defer w.Close()
	return r.fetchPage(cmd.Context(), w, path)
}

func (r *RootCommand) fetchPage(ctx context.Context, w io.Writer, path string) error {
	if r.client == nil {
		return fmt.Errorf("docs client not initialized")
	}
	if r.renderer == nil {
		return fmt.Errorf("markdown renderer not initialized")
	}

	ref := &url.URL{Path: path}
	page, err := r.client.FetchPage(ctx, ref)
	if err != nil {
		return err
	}

	doc, err := markdown.Parse(page.Content)
	if err != nil {
		return err
	}

	out, err := r.renderer.Render(doc)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, out)
	return err
}
