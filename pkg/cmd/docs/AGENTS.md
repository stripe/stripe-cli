# cmd

This package defines the root Cobra command and all subcommands for the `stripe docs` plugin.

## Structure

- `root.go` — `RootCommand` struct, `New()` constructor, `WithOptions()`, and the `Option` functional type
- Subcommands are added as methods on `RootCommand` and registered in `New()` via `r.cmd.AddCommand(...)`

## Conventions

- Each subcommand lives in its own file named after the command (e.g. `search.go` for `stripe docs search`)
- Subcommand constructors are methods on `RootCommand` so they can close over shared dependencies
- Use `WithOptions` + functional `Option` values to inject dependencies (e.g. HTTP client) rather than globals
