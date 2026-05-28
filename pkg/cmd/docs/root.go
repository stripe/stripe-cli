package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli-docs-plugin/internal/agent"
	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
	"github.com/stripe/stripe-cli-docs-plugin/internal/pager"
	"github.com/stripe/stripe-cli-docs-plugin/internal/tui"
	"github.com/stripe/stripe-cli-docs-plugin/markdown"
)

// RootCommand is the root command for the docs plugin.
type RootCommand struct {
	cmd       *cobra.Command
	client    *docs.Client
	renderer  markdown.Renderer
	version   string
	configDir string
	noPager   bool
	noTUI     bool
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

// WithConfigDirectory sets the Stripe config directory (e.g. ~/.config/stripe).
func WithConfigDirectory(dir string) Option {
	return func(r *RootCommand) { r.configDir = dir }
}

// New creates a new RootCommand with sensible defaults.
func New() *RootCommand {
	r := &RootCommand{}

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
	r.cmd.PersistentFlags().BoolVar(&r.noTUI, "no-tui", false, "Do not use the interactive TUI")
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
	if r.client == nil {
		r.client = docs.NewClient(r.version)
		if r.configDir != "" {
			if cache, err := docs.NewFSCache(filepath.Join(r.configDir, "docs", "cache")); err == nil {
				r.client = r.client.WithOptions(docs.WithCache(cache))
			}
		}
	}
	if r.renderer == nil {
		var opts []markdown.RendererOption
		if agent.Detect() {
			opts = append(opts, markdown.WithStyle("notty"))
			r.noPager = true
			r.noTUI = true
		}
		r.renderer, _ = markdown.NewRenderer(opts...)
	}
	return r
}

// Root returns the cobra command, used by tools like the doc generator.
func (r *RootCommand) Root() *cobra.Command { return r.cmd }

func (r *RootCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		if err := cmd.Help(); err != nil {
			return fmt.Errorf("showing help: %w", err)
		}
		return nil
	}

	path := args[0]
	if !strings.HasPrefix(path, "/") {
		path = "/" + strings.Join(args, "/")
	}

	if r.useTUI(cmd) {
		return r.runTUI(cmd.Context(), path)
	}

	w := pager.New(cmd.OutOrStdout(), !r.noPager)
	defer func() { _ = w.Close() }()
	return r.fetchPage(cmd.Context(), w, path)
}

func (r *RootCommand) useTUI(cmd *cobra.Command) bool {
	if r.noTUI {
		return false
	}
	f, ok := cmd.OutOrStdout().(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func (r *RootCommand) runTUI(ctx context.Context, path string) error {
	if r.client == nil {
		return fmt.Errorf("docs client not initialized")
	}
	if r.renderer == nil {
		return fmt.Errorf("markdown renderer not initialized")
	}

	ref := &url.URL{Path: path}
	page, err := r.client.FetchPage(ctx, ref)
	if err != nil {
		return fmt.Errorf("fetching page: %w", err)
	}

	m := tui.New(
		tui.WithClient(r.client),
		tui.WithRenderer(r.renderer),
		tui.WithPage(tui.Page{
			Content: page.Content,
			URL:     page.URL,
		}),
	)

	p := tea.NewProgram(m)
	if _, err = p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}
	return nil
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
		return fmt.Errorf("fetching page: %w", err)
	}

	doc, err := markdown.Parse(page.Content)
	if err != nil {
		return fmt.Errorf("parsing markdown: %w", err)
	}

	out, err := r.renderer.Render(doc)
	if err != nil {
		return fmt.Errorf("rendering page: %w", err)
	}

	if _, err = fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}
