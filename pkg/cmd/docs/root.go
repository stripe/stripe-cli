package docs

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
	"golang.org/x/term"

	cliconfig "github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/useragent"
	"github.com/stripe/stripe-cli/pkg/version"

	"github.com/stripe/stripe-cli/internal/docs"
	"github.com/stripe/stripe-cli/internal/markdown"
	"github.com/stripe/stripe-cli/internal/pager"
	"github.com/stripe/stripe-cli/internal/tui"
	"github.com/stripe/stripe-cli/internal/ui"
)

const colorValueOff = "off"

// RootCommand is the root command for the docs plugin.
type RootCommand struct {
	cmd *cobra.Command

	cfg *cliconfig.Config

	client       *docs.Client
	renderer     markdown.Renderer
	rendererOpts []markdown.RendererOption
	logger       *slog.Logger

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
		Short: "Browse docs.stripe.com from the terminal",
		Long: `Browse docs.stripe.com from the terminal.

Read a page by path:

  docs /payments
  docs /api/customers

Search documentation by keywords:

  docs search "payment intents"

Read API Reference pages by their identifier:

  docs api product
  docs api GET /v1/products
  docs api product.created`,
		Args:              cobra.ArbitraryArgs,
		PersistentPreRunE: r.preRun,
		RunE:              r.run,
		SilenceUsage:      true,
	}

	agentDetected := useragent.DetectAIAgent(os.Getenv) != ""
	r.cmd.PersistentFlags().BoolVar(&r.noPager, "no-pager", agentDetected, "Write output directly to stdout")
	r.cmd.PersistentFlags().BoolVar(&r.noTUI, "no-tui", agentDetected, "Write output directly without the interactive browser")

	docs := &cobra.Group{ID: "docs", Title: "Docs Commands:"}

	r.cmd.AddGroup(docs)

	searchCmd := r.newSearchCommand()
	searchCmd.GroupID = docs.ID
	r.cmd.AddCommand(searchCmd)

	apiCmd := r.newAPICmd()
	apiCmd.GroupID = docs.ID
	r.cmd.AddCommand(apiCmd)

	return r
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
	r.client = docs.NewClient(version.Version)
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
	case colorValueOff:
		opts = append(opts, markdown.WithStyle("notty"))
	case "on":
		// Use default styled rendering (auto-detect dark/light).
	default:
		if useragent.DetectAIAgent(os.Getenv) != "" {
			opts = append(opts, markdown.WithStyle("notty"))
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
		r.logger = slog.New(handler).With("version", version.Version)
	}
	if r.logger != nil && r.client != nil {
		r.client.WithOptions(docs.WithLogger(r.logger))
	}
}

func (r *RootCommand) preRun(_ *cobra.Command, _ []string) error {
	r.initLogger()
	r.initRenderer()
	if r.logger != nil {
		if a := useragent.DetectAIAgent(os.Getenv); a != "" {
			r.logger.Debug("agent detected", "name", a)
		}
	}
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
	if r.noTUI || useragent.DetectAIAgent(os.Getenv) != "" {
		return false
	}
	f, ok := cmd.OutOrStdout().(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// terminalSize returns the current terminal dimensions, if available.
func terminalSize(cmd *cobra.Command) (w, h int, ok bool) {
	f, isFile := cmd.OutOrStdout().(*os.File)
	if !isFile {
		return 0, 0, false
	}
	w, h, err := term.GetSize(int(f.Fd()))
	return w, h, err == nil
}

// show displays a page to the user. When a TTY is detected it launches the
// interactive TUI; otherwise it renders markdown and pipes it through a pager.
// A nil page starts the TUI at the home screen. Extra TUI options (e.g.
// tui.WithPaletteInput) are forwarded to the model constructor.
func (r *RootCommand) show(cmd *cobra.Command, page *docs.Page, extraOpts ...tui.Option) error {
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
		if w, h, ok := terminalSize(cmd); ok {
			opts = append(opts, tui.WithWindowSize(w, h))
		}
		opts = append(opts, extraOpts...)
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
