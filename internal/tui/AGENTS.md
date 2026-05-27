# internal/tui

Bubble Tea v2 TUI for browsing Stripe documentation in the terminal. Renders markdown in a scrollable viewport.

## Usage

```go
import "github.com/stripe/stripe-cli-docs-plugin/internal/tui"

// Create a TUI model with a document and renderer.
doc, _ := markdown.Parse(page.Content)
r, _ := markdown.NewRenderer()
m := tui.New(
    tui.WithClient(client),
    tui.WithRenderer(r),
    tui.WithDocument(doc),
    tui.WithTitle("Accept a Payment"),
)

// Run the TUI.
p := tea.NewProgram(m)
_, err := p.Run()
```

