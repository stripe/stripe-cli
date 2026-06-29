# palette

A command palette [Bubble Tea][bubbletea] component with pluggable modes —
fuzzy-filter a static list, dispatch an async search, or mix both behind
different prefixes.

```go
import "github.com/stripe/stripe-cli/pkg/docs/internal/palette"

var commands = []palette.Item{
    palette.Command{Name: "Open file"},
    palette.Command{Name: "Save"},
    palette.Command{Name: "Quit"},
}

mode := palette.Mode{
    Name: "commands",
    Items: func(_ palette.Model, q string) []palette.Item {
        return palette.FilterFuzzy(commands, q)
    },
}

p := palette.New(palette.WithModes(mode))
```

Each `palette.Mode` owns its own `Match`, `Items`, and optional async `Search`
or typeable `Facets`.

## Built with

- [Bubble Tea][bubbletea] — the TUI framework
- [Bubbles][bubbles] — the primitive components
- [Lip Gloss][lipgloss] — styling and layout

[bubbletea]: https://github.com/charmbracelet/bubbletea
[bubbles]: https://github.com/charmbracelet/bubbles
[lipgloss]: https://github.com/charmbracelet/lipgloss
