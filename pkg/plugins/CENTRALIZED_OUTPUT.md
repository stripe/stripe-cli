# Centralized plugin output

Plugins can hand their output to the core CLI instead of printing it themselves.
The core renders it, so formatting lives in one place across every plugin.

This is milestone 1: transport and compatibility only. Core-owned `--format=json`,
Charmbracelet rendering, interactive prompts, and standardized error blocks come
later.

## Protocol

Three RPCs on `CoreCLIHelper` (`pkg/plugins/proto/main.proto`):

| RPC | Purpose | Frequency |
| --- | --- | --- |
| `SendMessage` | One incremental message with a level (info/success/warning/error) | Any number of times |
| `SendProgress` | Step or spinner lifecycle, keyed by a caller-generated id | Any number of times |
| `SendCommandOutput` | The command's final ordered result blocks | At most once per command |

Requests carry semantic fields (typed level and progress enums, typed block
kinds) even though milestone 1 renders them as plain text. Callers should not
assume how a block is drawn.

`SendCommandOutput` blocks are rendered in the order given. A `data` block's
payload is JSON; object fields render in payload order, so output is
byte-deterministic for the same input.

## Host side

`coreCLIHelper` (`pkg/plugins/core_cli_helper.go`) forwards each RPC to
`rendering.Engine`, which writes to the writers the host supplies — never to the
plugin's stdio. Use `NewCoreCLIHelperWithWriters` to render somewhere other than
the process's stdout/stderr (tests do this).

Rendering is serialized, so a plugin sending from several goroutines cannot
interleave partial lines. `SendCommandOutput` stops any spinner still running
before drawing the result.

Colors come from `pkg/ansi` and switch off automatically for non-terminal
writers, so piped output is plain text.

## Message sizes

Command output can exceed gRPC's 4MB default per-message limit (a large result,
or a plugin forwarding subprocess output). Both ends of the helper channel raise
the limit to 64MB — see `maxHelperMessageSize` in `interface_grpc_3.go`. Sending
output through this channel also avoids the `GRPCStdio` truncation seen when
large output is written to plugin stdio.

## Plugin side and fallback

Plugins use `pkg/plugins/sdk`, which builds the protobuf messages:

```go
cli := sdk.New(bootstrap.GetCoreCLIHelper()) // nil helper is fine
if err := cli.Output("apps upload",
    sdk.Data(result),
    sdk.Warning("Your app ID is permanent once uploaded"),
    sdk.NextStep("View status", dashboardURL),
); sdk.Unsupported(err) {
    // Render locally: no core helper, or a core CLI that predates this RPC.
}
```

`sdk.Unsupported` is the whole compatibility contract. Exactly two conditions
allow local rendering:

- `ErrNoHelper` — the plugin ran under the v1/v2 protocol or in standalone/dev
  mode, so nothing was sent.
- gRPC `Unimplemented` — the core CLI is older than this RPC, so nothing was
  rendered.

Every other error must be surfaced. A transport failure can arrive *after* the
core already rendered the output; falling back there would print the result
twice.
