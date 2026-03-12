# CLAUDE.md

## Build & Test Commands

```bash
make build              # Build binary (runs go generate first)
make test               # Run tests with race detector and coverage
make lint               # Run golangci-lint v2
make fmt                # Format with gofmt + goimports
make ci                 # Full CI: build-all-platforms + test + go-mod-tidy + protoc-ci
make protoc             # Regenerate all protobuf code
make setup              # Download Go modules
go build ./...          # Quick compilation check
go vet ./...            # Static analysis
```

## Project Structure

- `cmd/stripe/main.go` - Entry point
- `pkg/cmd/` - All CLI commands (Cobra). `root.go` registers everything
- `pkg/cmd/resources_cmds.go` - **Auto-generated** from OpenAPI spec (do not edit manually)
- `pkg/proxy/events_list.go` - **Auto-generated** event type list
- `pkg/gen/` - Code generation logic and templates (`gen_resources_cmds.go`, `gen_events_list.go`)
- `api/openapi-spec/` - OpenAPI specs consumed by code generation
- `rpc/` - Protobuf definitions and generated gRPC code
- `pkg/plugins/` - HashiCorp go-plugin based plugin system
- `pkg/proxy/` - Webhook forwarding proxy (`stripe listen`)
- `pkg/websocket/` - WebSocket client for live event streaming
- `pkg/rpcservice/` - gRPC server for external integrations
- `pkg/requests/` - HTTP request building and Stripe API interaction
- `pkg/stripe/` - Low-level Stripe HTTP client
- `pkg/fixtures/` - Declarative test data creation
- `pkg/config/` - CLI configuration management
- `pkg/login/` - Authentication flows
- `.goreleaser/` - Per-platform release configs (linux, mac, windows)

## Code Generation

`go generate ./...` is triggered by `//go:generate` directives in `pkg/cmd/root.go`:
- Generates `pkg/cmd/resources_cmds.go` from OpenAPI spec → CLI resource commands
- Generates `pkg/proxy/events_list.go` → list of Stripe event types
- Generates `pkg/requests/stripe_version_header.go` → API version constants

Templates live in `pkg/gen/*.go.tpl`. Build tags `gen_resources` and `events_list` gate the generators.

## Key Conventions

- **Import ordering**: stdlib, then external packages, then `github.com/stripe/stripe-cli/...` (enforced by goimports with local-prefixes)
- **Error handling**: Use typed errors (e.g., `RequestError`), check with `errors.As`. User-facing errors go to stderr
- **Testing**: Use `testify/assert` and `testify/require`. Mock HTTP with `httptest.NewServer`. Mock filesystem with `spf13/afero`
- **Naming**: snake_case files, PascalCase exported types, generated files have `// This file is generated; DO NOT EDIT.` header
- **Indentation**: Tabs for Go, 2-space for JSON/YAML/Markdown

## Architecture Notes

**Command types:**
1. Manual commands in `pkg/cmd/*.go` (login, listen, trigger, etc.)
2. Auto-generated resource commands from OpenAPI (`stripe customers create`, `stripe billing meters list`, etc.)
3. Plugin commands loaded dynamically at startup

**Request flow:** Cobra command → `pkg/requests/base.go` request builder → `pkg/stripe/client.go` HTTP client → response handler → output

**Two API styles:** v1 (form-encoded params) and v2 (JSON body). The `--stripe-version` and content-type headers differ accordingly.

**Build tags:** `dev`, `localdev` (plugin dev mode), `gen_resources`, `events_list`

## Linting

golangci-lint v2 config is in `.golangci.yml`. Enabled linters include staticcheck, govet, unused, ineffassign, misspell, dupl, and others. The `resources_cmds.go` file is excluded from dupl checks. Several staticcheck style checks (ST*, QF*) are suppressed.

## Version Injection

GoReleaser sets the version at build time via:
```
-ldflags "-s -w -X github.com/stripe/stripe-cli/pkg/version.Version={{.Version}}"
```
Default value is `"master"` for dev builds.
