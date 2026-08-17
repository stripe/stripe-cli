# Centralized plugin output

Plugins can hand their output to the core CLI instead of printing it themselves.
The core renders it, so formatting lives in one place across every plugin.

This is milestone 1: transport and compatibility only. Core-owned `--format=json`,
Charmbracelet rendering, interactive prompts, and standardized error blocks come
later.

## Protocol

Two RPCs on `CoreCLIHelper` (`pkg/plugins/proto/main.proto`):

| RPC | Purpose | Frequency |
| --- | --- | --- |
| `SendCommandOutput` | Every kind of output, as a list of `OutputBlock`s | Any number of times |
| `Prompt` | Ask the user a question and read the answer | Any number of times |

All output rides one RPC. `OutputBlock` is a `oneof` over the kinds:

| Variant | Carries |
| --- | --- |
| `MessageBlock` | One message with a level (info/success/warning/error) |
| `ProgressBlock` | Step or spinner lifecycle, keyed by a caller-generated id |
| `DataBlock` | A typed, JSON-payload result block (`data`, `warning`, `nextstep`, `error`) |

`Prompt` stays separate because it is the only call with a meaningful response.

Requests carry semantic fields (typed level and progress enums, typed block
kinds) even though milestone 1 renders them as plain text. Callers should not
assume how a block is drawn.

Blocks are rendered in the order given. A `DataBlock` payload is JSON; object
fields render in payload order, so output is byte-deterministic for the same
input.

### `final`

`SendCommandOutputRequest.final` marks the request that ends a command's output.
The renderer uses it to tear down spinners before printing the result, and to
close the JSON envelope. It is explicit rather than inferred from block kinds, so
a command that emits data mid-run is not mistaken for a finished one. `sdk` sets
it only on `Output`.

### Adding a block kind

Prefer a new `DataBlock.Type`: `type` is a string, so it needs no protocol
change and an older core forwards the payload verbatim. A new `oneof` variant is
the fallback; note that an older core decodes it to an *empty* variant and the
RPC still succeeds, so the plugin cannot detect the loss. The renderer therefore
counts unrecognized variants and prints an upgrade notice to stderr rather than
dropping them silently.

## Host side

`coreCLIHelper` (`pkg/plugins/core_cli_helper.go`) forwards each RPC to
`rendering.Engine`, which writes to the writers the host supplies — never to the
plugin's stdio. Use `NewCoreCLIHelperWithWriters` to render somewhere other than
the process's stdout/stderr (tests do this).

Rendering is serialized, so a plugin sending from several goroutines cannot
interleave partial lines. A request with `final` set stops any spinner still
running before drawing the result.

In JSON mode stdout is reserved for one envelope per command, so message and
progress blocks go to stderr and data blocks are buffered until `final` arrives.

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
- gRPC `Unimplemented` — the core CLI is older than `SendCommandOutput`, so
  nothing was rendered. Because all output rides one RPC, this is a single
  capability probe: no host can render some kinds of output but not others.

Every other error must be surfaced. A transport failure can arrive *after* the
core already rendered the output; falling back there would print the result
twice.
