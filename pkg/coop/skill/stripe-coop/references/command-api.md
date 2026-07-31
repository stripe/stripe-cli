# Co-op command API

All agent-facing commands print a single JSON object. Successful responses go
to stdout; failures go to stderr with a non-zero exit code but still contain
structured JSON.

## Response shape

```json
{
  "ok": true,
  "protocol_version": 1,
  "session_id": "coop_abc123",
  "node": 3,
  "state": "active",
  "message": "Started: Create a PaymentIntent",
  "next": "stripe coop agent report-check --session=coop_abc123 --step=3 ...",
  "next_template": "stripe coop agent report-work --session=coop_abc123 --step=3 --note=\"<what you did>\"",
  "required_inputs": [{"name": "note", "flag": "--note", "description": "..."}],
  "wait_timeout_seconds": 300,
  "required_outputs": [{"source": "", "field": "id"}],
  "agent_prompt": "Current node 3 of 9 ...",
  "api_request": {"method": "post", "path": "/v1/payment_intents", "params": {}},
  "test_requests": [],
  "events": [],
  "sdk_example": "..."
}
```

Field semantics:

- `next` — an exact command. Run it verbatim; it never contains placeholders.
- `next_template` + `required_inputs` — a command template. Substitute a real
  value for every `<...>` placeholder before running. Never execute a command
  that still contains `<` `>`.
- Exactly one of `next` / `next_template` is set when a continuation exists.
- `wait_timeout_seconds` — the command in `next` blocks up to this many
  seconds. Give your shell tool at least one minute more than this so the
  structured timeout response can arrive (e.g. 300s wait → 360s shell budget).
- `agent_prompt` — the task contract for the current node.
- `required_outputs` — values you must report with `--output` on
  `report-work` (see node-contracts.md).

Failure shape:

```json
{
  "ok": false,
  "protocol_version": 1,
  "error": "what went wrong",
  "recovery": {
    "hint": "how to proceed",
    "next": "stripe coop status"
  }
}
```

`recovery` always contains a `hint` and either `next` or `next_template` (with
`required_inputs`), following the same rules as above. Read the hint, fix the
issue, and continue with the recovery continuation.

## Session commands

| Command | Purpose |
|---|---|
| `stripe coop recommend --all` | List blueprint summaries for selection. Add `--include-testing` to include testing blueprints. |
| `stripe coop run <blueprint-id>` | Create a session from a blueprint. Flags: `--language`, repeatable `--setting key=value` and `--param key=value`, `--parent-session`, `--parent-step`. |
| `stripe coop status [--session=<id>] [--json]` | Inspect session progress. Read-only. |
| `stripe coop stop [--session=<id>]` | End a session (normally the human's decision). |
| `stripe coop join <id>` | Human-facing live TUI. Not for agents. |

## Agent lifecycle commands

All take `--session=<id>` and (except `resume`, `next-action`,
`start-followup`) a 1-based `--step=<n>` task number.

### `stripe coop agent start-work --session=<id> --step=<n> --note="..."`
Marks the node active and returns the node's `agent_prompt`, structured
context, and `required_outputs`. Safe to replay: re-running on an active node
just returns the current contract.

### `stripe coop agent report-check --session=<id> --step=<n> --check="..." [--detail="..."] [--passed]`
Records one verification. `--check` is a one-line label the reviewer sees;
put command output and reasoning in `--detail`. Pass `--passed` only if you
observed the expected result. Report at least one meaningful check before
`report-work` on every non-skipped reviewable node; failed or partial checks
are reported without `--passed` plus an explanation of the limitation.

### `stripe coop agent report-work --session=<id> --step=<n> --note="..." [--file=...] [--lines=a-b] [--snippet=...] [--output field=value]...`
Reports the implementation. `--note` is required. `--file` should be the main
implementation file in the user's app (not README/package files unless the
node is documentation-only). Every entry in `required_outputs` must be
supplied as `--output field=value` or `--output source:field=value`; values
may be raw strings or JSON. Moves the node to review (or done for
informational nodes) and returns what to do next.

### `stripe coop agent skip --session=<id> --step=<n> --note="<reason>"`
Skips a genuinely inapplicable node. Nodes that depend on its outputs are
skipped with it; skipping fails if a dependent node is already done.

### `stripe coop agent await-review --session=<id> --step=<n>`
Blocks until the developer confirms or requests changes, up to
`wait_timeout_seconds` (currently 300s). Only run it when a response tells
you to. See recovery.md for timeout, confirmation, and rejection handling.

### `stripe coop agent resume --session=<id>`
Read-only: reports the current lifecycle continuation after a wake-up or lost
context. Run it when the TUI injects a resume prompt, then run the returned
non-empty `next` exactly; an empty continuation means another handoff already
advanced the session — continue your current work.

### `stripe coop agent next-action --session=<id> [--completed=<action-id>]`
After session completion, waits (up to 10 minutes per call) for the
developer's next-action choice, or records a finished follow-up with
`--completed`. Re-run on timeout. The response lists `suggestions`, each with
an `id` — that `id` is the value for `--action` on `start-followup` and later
for `--completed`; the response's `next`/`agent_prompt` name the selected one.

### `stripe coop agent start-followup --session=<parent-id> --action=<action-id> [--target=...]`
Starts a guided follow-up session for an action offered by `next-action`.
This is the only way to begin follow-up work after completion.

## Stripe documentation lookup

When Stripe behavior, parameters, or events are ambiguous, consult current
docs through the CLI instead of guessing:

```
stripe docs search "<question>" --non-interactive --no-pager
stripe docs <result-path> --non-interactive --no-pager
stripe docs api <resource-or-event> --non-interactive --no-pager
```

Treat current CLI documentation as authoritative over model memory. Do not use
docs to choose or switch integrations — the blueprint owns that decision.
