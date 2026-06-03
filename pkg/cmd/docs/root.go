package cmd

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	charmlog "charm.land/log/v2"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	cliconfig "github.com/stripe/stripe-cli/pkg/config"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli-docs-plugin/internal/agent"
	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
	"github.com/stripe/stripe-cli-docs-plugin/internal/markdown"
	"github.com/stripe/stripe-cli-docs-plugin/internal/pager"
	"github.com/stripe/stripe-cli-docs-plugin/internal/tui"
	"github.com/stripe/stripe-cli-docs-plugin/internal/ui"
)

// RootCommand is the root command for the docs plugin.
type RootCommand struct {
	cmd *cobra.Command

	cfg *cliconfig.Config

	client       *docs.Client
	renderer     markdown.Renderer
	rendererOpts []markdown.RendererOption
	logger       *slog.Logger

	version string
	noPager bool
	noTUI   bool
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

// WithLogger sets the structured logger for the plugin.
func WithLogger(logger *slog.Logger) Option {
	return func(r *RootCommand) { r.logger = logger }
}

// WithConfig sets the shared Stripe CLI configuration whose fields are
// populated by cobra flag parsing at runtime (e.g. --color, --log-level).
func WithConfig(cfg *cliconfig.Config) Option {
	return func(r *RootCommand) { r.cfg = cfg }
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
		Args:              cobra.ArbitraryArgs,
		PersistentPreRunE: r.preRun,
		RunE:              r.run,
		SilenceUsage:      true,
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
	r.cmd.AddCommand(r.newSearchCommand())
	r.cmd.AddCommand(r.newAPICmd())
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
	r.initClient()
	if r.cfg == nil {
		r.initRenderer()
	}
	return r
}

func (r *RootCommand) initClient() {
	if r.client != nil {
		if r.logger != nil {
			r.client.WithOptions(docs.WithLogger(r.logger))
		}
		return
	}
	r.client = docs.NewClient(r.version)
	var clientOpts []docs.ClientOption
	if r.logger != nil {
		clientOpts = append(clientOpts, docs.WithLogger(r.logger))
	}
	if r.cfg != nil {
		configDir := r.cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
		if cache, err := docs.NewFSCache(filepath.Join(configDir, "docs", "cache")); err == nil {
			clientOpts = append(clientOpts, docs.WithCache(cache))
		}
	}
	if len(clientOpts) > 0 {
		r.client.WithOptions(clientOpts...)
	}
}

func (r *RootCommand) initRenderer() {
	if r.renderer != nil {
		return
	}
	var opts []markdown.RendererOption
	switch r.color() {
	case "off":
		opts = append(opts, markdown.WithStyle("notty"))
	case "on":
		// Use default styled rendering (auto-detect dark/light).
	default:
		if agent.Detect() {
			opts = append(opts, markdown.WithStyle("notty"))
			r.noPager = true
		}
	}
	r.rendererOpts = opts
	r.renderer, _ = markdown.NewRenderer(opts...)
}

func (r *RootCommand) color() string {
	if r.cfg != nil && r.cfg.Color != "" {
		return r.cfg.Color
	}
	return "auto"
}

func (r *RootCommand) initLogger() {
	if r.logger == nil && r.cfg != nil {
		level, err := charmlog.ParseLevel(r.cfg.LogLevel)
		if err != nil {
			level = charmlog.InfoLevel
		}
		handler := charmlog.NewWithOptions(os.Stderr, charmlog.Options{
			Level:  level,
			Prefix: "docs",
		})
		r.logger = slog.New(handler).With("version", r.version)
	}
	if r.logger != nil && r.client != nil {
		r.client.WithOptions(docs.WithLogger(r.logger))
	}
}

func (r *RootCommand) preRun(_ *cobra.Command, _ []string) error {
	r.initLogger()
	r.initRenderer()
	return nil
}

// Root returns the cobra command, used by tools like the doc generator.
func (r *RootCommand) Root() *cobra.Command { return r.cmd }

func (r *RootCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		if r.useTUI(cmd) {
			return r.show(cmd, nil)
		}
		if err := cmd.Help(); err != nil {
			return fmt.Errorf("showing help: %w", err)
		}
		return nil
	}

	path := args[0]
	if !strings.HasPrefix(path, "/") {
		path = "/" + strings.Join(args, "/")
	}

	if r.client == nil {
		return fmt.Errorf("docs client not initialized")
	}
	if r.renderer == nil {
		return fmt.Errorf("markdown renderer not initialized")
	}

	ref := &url.URL{Path: path}
	page, err := r.client.FetchPage(cmd.Context(), ref)
	if err != nil {
		return fmt.Errorf("fetching page: %w", err)
	}

	return r.show(cmd, &page)
}

func (r *RootCommand) useTUI(cmd *cobra.Command) bool {
	if r.noTUI {
		return false
	}
	f, ok := cmd.OutOrStdout().(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// show displays a page to the user. When a TTY is detected it launches the
// interactive TUI; otherwise it renders markdown and pipes it through a pager.
// A nil page starts the TUI at the home screen.
func (r *RootCommand) show(cmd *cobra.Command, page *docs.Page) error {
	if r.client == nil {
		return fmt.Errorf("docs client not initialized")
	}
	if r.renderer == nil {
		return fmt.Errorf("markdown renderer not initialized")
	}

	if r.useTUI(cmd) {
		opts := []tui.Option{
			tui.WithClient(r.client),
			tui.WithRendererOptions(r.rendererOpts...),
			tui.WithStyles(ui.DefaultStyles()),
		}
		if page != nil {
			opts = append(opts, tui.WithPage(tui.Page{
				Content: page.Content,
				URL:     page.URL,
			}))
		}
		p := tea.NewProgram(tui.New(opts...), tea.WithFilter(tui.NewMouseEventFilter()))
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running TUI: %w", err)
		}
		return nil
	}

	w := pager.New(cmd.OutOrStdout(), !r.noPager)
	defer func() { _ = w.Close() }()

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
