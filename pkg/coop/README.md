# Co-op Mode

Co-op mode enables an AI agent and a human developer to build Stripe integrations together in real time. The agent writes code and reports progress; the developer watches in a terminal UI, reviews work, and confirms each step.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ tmux session                                                │
│                                                             │
│  ┌──────────────────────┐   ┌────────────────────────────┐ │
│  │  TUI (coop join)     │   │  Agent (see Supported      │ │
│  │                      │   │  Agents below)             │ │
│  │                      │   │                            │ │
│  │  Reads session.json  │   │  Writes session.json via   │ │
│  │  every 500ms         │   │  stripe coop agent commands│ │
│  │                      │   │                            │ │
│  └──────────┬───────────┘   └──────────────┬─────────────┘ │
│             │                              │               │
│             └──────────┬───────────────────┘               │
│                        ▼                                   │
│              ~/.config/stripe/coop/                         │
│              ├── coop_abc123.json        (session file)     │
│              └── coop_abc123.json.heartbeat (agent alive)   │
└─────────────────────────────────────────────────────────────┘
```

No server, no HTTP, no WebSocket. Communication is through a shared JSON session file with atomic writes (write to .tmp, rename).

## Supported Agents

`stripe coop start` detects installed agents on PATH and, when more than one is
present, asks which to use. `--agent=<id-or-command>` forces a specific one.
The registry lives in `pkg/cmd/coop/harness.go`.

| Agent | `--agent` | Binary | Invocation | Auto-approve |
|-------|-----------|--------|------------|--------------|
| Claude Code | `claude` | `claude` | positional prompt | `--dangerously-skip-permissions` |
| Codex | `codex` | `codex` | positional prompt | `--dangerously-bypass-approvals-and-sandbox` |
| Cursor CLI | `cursor` | `cursor-agent` | positional prompt | `--force` |
| Gemini CLI | `gemini` | `gemini` | `-i <prompt>` | `--approval-mode=yolo` |
| Goose | `goose` | `goose` | `run -s -t <prompt>` | `GOOSE_MODE=auto` (env) |
| opencode | `opencode` | `opencode` | `--prompt <prompt>` | `--auto` |
| Pi | `pi` | `pi` | positional prompt | none — see below |

### Adding an agent

The launcher generates one script per session — `exec <path> [args] [auto-approve] [prompt-flag] "$prompt"` —
so a new agent must **accept an initial prompt that leaves an interactive session
running**. This is the binding constraint, and it disqualifies more agents than
it admits.

Watch for two traps when reading a candidate's docs:

- **The obvious prompt flag is usually the wrong one.** On Gemini, Qwen, Copilot
  and others, `-p`/`--prompt` means "run headless and exit"; a *different* flag
  (`-i`, `--prompt-interactive`) is what seeds an interactive session. opencode
  inverts this — its `--prompt` is the interactive one. Only the interactive
  spelling belongs in `promptFlag`.
- **A headless one-shot agent cannot drive co-op's review loop.** Co-op depends
  on the agent process surviving `stripe coop agent await-review`, which blocks
  until the developer confirms in the TUI, and on discovery mode the agent must
  be able to ask the developer questions directly.

Known-incompatible as of 2026-07: **Crush** (`crush run` is non-interactive;
seeding the TUI is [charmbracelet/crush#1791](https://github.com/charmbracelet/crush/issues/1791),
still open), **Hermes** (`-q`/`-z` are both non-interactive; no positional prompt),
**GitHub Copilot CLI** (`-p` forces one-shot), **Aider** (`--message` forces
one-shot), and **Amp** (seeds only via stdin, which would break the pane's TUI).

`Qwen Code` and `Factory Droid` are compatible but not yet registered — Qwen is a
gemini-cli fork (`-i`, `--yolo`) and Droid takes a positional prompt with
`--skip-permissions-unsafe`.

### A note on Pi

Pi ships **no permission system and no sandbox**: it runs with the privileges of
the process that launched it and never prompts before acting. Co-op therefore
skips the permission-mode question for Pi, because there is nothing to skip —
not because it is running in a safe mode. Every co-op session with Pi behaves as
if auto-approve were on.

## Node State Machine

```
pending ──→ active ──→ review ──→ done     (normal flow)
   │          │          │
   │          │          └──→ active        (request changes: developer entered feedback)
   │          │
   │          └──→ done                     (auto_confirm nodes skip review)
   │          └──→ skipped                  (agent decides node doesn't apply)
   │
   └──→ skipped                             (agent skips from pending)
```

**Terminal states:** `done`, `skipped` — no transitions out.

**Transitions are validated** — `session.TransitionNode()` returns an error for invalid transitions (e.g. pending→done, done→active).

## Session States

```
active ──→ completed    (all nodes done/skipped, or "stripe coop stop")
   │
   └──→ aborted         ("stripe coop stop --abort")
```

## Commands

### User-facing (human runs these)
| Command | Purpose |
|---------|---------|
| `stripe coop start [blueprint]` | Launch tmux split with agent + TUI |
| `stripe coop join [session-id]` | Open the TUI for an existing session |
| `stripe coop status` | Show session summary |
| `stripe coop stop` | End the session |
| `stripe coop recommend` | List available blueprints |

### Agent-facing (AI agent runs these)
| Command | Purpose |
|---------|---------|
| `stripe coop run <blueprint>` | Create a session (outputs JSON with instructions) |
| `stripe coop agent start-work --step <n>` | Mark a node as active |
| `stripe coop agent report-work --step <n>` | Mark a node complete (→ review or → done if auto_confirm) |
| `stripe coop agent report-check --step <n>` | Add a verification check |
| `stripe coop agent skip --step <n>` | Skip a node |
| `stripe coop agent await-review --step <n>` | Block until developer confirms or requests changes |
| `stripe coop agent next-action` | Show post-completion options (blocks until selection) |
| `stripe coop agent start-followup` | Start an internal guided follow-up session selected from next actions |

All agent commands output JSON with an `ok` field and a `next` field suggesting the next command. The `--step` flag name is retained for the CLI, but its value is the 1-based node number across the session.

## TUI Keybindings

| Key | Action |
|-----|--------|
| `↑`/`k` | Move cursor up |
| `↓`/`j` | Move cursor down |
| `PgUp`/`b` | Page up |
| `PgDn`/`Space` | Page down |
| `Home`/`g` | Jump to top |
| `End`/`G` | Jump to bottom |
| `←` | Collapse selected step |
| `→` | Expand selected step |
| `e` / `?` / `Enter` | Toggle detail panel for selected step or node |
| `Tab` | Move to the next detail tab |
| `Esc` | Close details or cancel a prompt |
| `c` | Confirm the selected review item |
| `r` | Request changes for the selected review item |
| `y` | Copy the selected review command when one is available |
| `f` | Resume following the active/review node after manual navigation |
| `o` | Open claim URL in browser (when sandbox is unclaimed) |
| `q` / `Ctrl+C` | Quit TUI |

When requesting changes, `r` opens a feedback prompt. Press `Enter` to submit a note and move the reviewed node or step back to `active`; press `Esc` to cancel.

In the completion view:
| Key | Action |
|-----|--------|
| `↑`/`↓` | Navigate suggestions |
| `Enter` | Select a suggestion |
| `q` | Quit |

## Example Flow

```bash
# Explicit blueprint: developer starts a pre-created session (launches tmux with agent + TUI)
$ stripe coop start one-time-payment --language=node

# What happens behind the scenes:
# 1. CLI creates the session and gives the agent the exact session protocol
# 2. The detected agent is launched in right pane
# 3. TUI appears in left pane showing step progress
# 4. Agent starts from the provided next command:
#      stripe coop agent start-work --session=coop_abc123 --step=1 --note="Beginning: Understand the project"
# 5. Agent works through steps, calling:
#      stripe coop agent start-work --session=coop_abc123 --step=1 --note="Scanning project"
#      stripe coop agent report-work --session=coop_abc123 --step=1 --note="Found Next.js app"
#      stripe coop agent start-work --session=coop_abc123 --step=2 --note="Creating product"
#      stripe coop agent report-work --session=coop_abc123 --step=2 --file=server.js --lines=5-20 --note="Created product"
#      stripe coop agent await-review --session=coop_abc123 --step=2   ← blocks until developer confirms
# 6. Developer sees progress live, presses 'c' to confirm
# 7. Agent continues to next step
# 8. After all steps: agent runs "stripe coop agent next-action --session=coop_abc123"
# 9. Developer picks what to do next from TUI suggestions
```

Discovery mode is different:

```bash
$ stripe coop start

# The agent explores the codebase, asks what the developer wants to build,
# runs `stripe coop recommend`, and only then runs:
#   stripe coop run <blueprint-id> --language=<lang>
```

Post-completion choices are written into the session file for the agent. Deploy follow-ups are internal guided sessions, not blueprints: the agent runs `stripe coop agent start-followup --session=<parent> --action=deploy` or `--action=deploy-update`. The child session returns to the completed parent by running `stripe coop agent next-action --session=<parent> --completed=<action>` when it finishes.

## Auto-Confirm

Nodes with `"auto_confirm": true` skip human review:
- `agent report-work` transitions directly to `done` (not `review`)
- `agent await-review` returns immediately if the node is auto-confirmed
- The prepended "Project context" step contains an auto-confirmed "Understand the project" node
- Blueprint nodes can set `"auto_confirm": true` for mechanical steps

## Heartbeat

When the agent runs `stripe coop agent await-review`, it writes a `.heartbeat` file every 500ms. The TUI checks this file:
- **Fresh heartbeat (< 5s old):** Agent is actively waiting for confirmation
- **No heartbeat + no session update in 2min:** Show idle warning

The heartbeat file is cleaned up when `await` exits.

## Resuming

`stripe coop join` is the recovery path. With no session ID, it opens the most
recent active session, falling back to the latest session if none are active.
Use `stripe coop join --resume` to pick from recent sessions.

| Issue | What to do |
|-------|------------|
| Node is active | Rejoin the session and check the agent pane/TUI state |
| Node or step is in review | Rejoin the session and confirm or request changes |
| Agent appears idle | Rejoin the session; the TUI shows heartbeat/idle state |
| Need a specific older session | Run `stripe coop join --resume` |

## Blueprint Format

Blueprints are embedded JSON in `pkg/coop/blueprints/`. Each has:
- `id` — unique identifier (also the filename without .json)
- `title`, `description` — human-readable
- `steps` — ordered groups of nodes

Each node has:
- `type` — `apiRequest`, `asyncHandler`, `uiComponent`, `cliCommand`, `dashboard`, `setUpWebhooks`, `testHelper`
- `auto_confirm` — skip human review for this node
- `description` — what the agent should do (source of truth)
- `review_prompt` — what the human should check before confirming
- `review_command` — optional command the TUI can show/copy for developer verification
- `request` — API request details (for `apiRequest` nodes with SDK snippet support)
- `request.hidden_params` — request fields that should not be shown directly in the TUI
- `requests` — API-backed test helper requests for `testHelper` nodes
- `events` — webhook events (for `asyncHandler` nodes)

`testHelper` request metadata tells the agent which Stripe-backed test helpers can advance test state. Agents should use those helpers while verifying work, but should not encode helper-only request parameters into the user's application.

### Syncing Blueprints

Workbench blueprint definitions are the source of truth. Do not supplement or modify `pkg/coop/blueprints/` by hand to add CLI-only product work. Update the upstream blueprint source, then sync the CLI-friendly JSON:

```bash
BLUEPRINT_SOURCE=/path/to/pay-server/frontend/workbench/shared/blueprints/src/blueprintDefinitions make sync-blueprints
```

If pay-server has already exported `dist/blueprints/*.json`, `BLUEPRINT_SOURCE`
can point at that directory instead.

After syncing, test with `go run ./cmd/stripe coop run <blueprint-id>`. Prefix matching works: short prefixes resolve to full IDs if unambiguous.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| TUI shows "Agent appears idle" | Agent crashed or stopped | Check the agent pane; restart with `stripe coop start` |
| Agent stuck on "await" | Developer hasn't confirmed | Press `c` in TUI to confirm, or `r` to request changes |
| "Version conflict" error | TUI and agent wrote simultaneously | Agent retries the command (safe to re-run) |
| "timed out waiting for session lock" | A previous writer left a `.lock` file behind | If no `stripe coop` command is running, remove the named lock file and retry |
| TUI shows wrong session | Multiple sessions exist | Use `stripe coop join <session-id>` with the correct ID |
| Steps not updating in TUI | Agent created a duplicate session | Check `stripe coop status` for the correct session ID |
| Agent ignores "next" hint | LLM didn't follow instructions | Copy the `next` value and run it manually, or restart |
| Double footer / layout broken | Terminal resize not detected | Resize the terminal window (triggers recalculation) |
| "Blueprint not found" | Typo in blueprint ID | Run `stripe coop recommend` to see available IDs |

## Locking

Writes are serialized with a per-session `.lock` file. `Store.Write()` also checks the file's current version before writing. If another writer changed the file since you read it, the write fails with a version conflict error. This prevents the TUI and agent from clobbering each other's changes.

## File Structure

```
pkg/coop/
  types.go          — Session, Node, Step types and constants
  session.go        — State machine, validation, queries
  store.go          — Atomic file I/O, heartbeat, lock files, optimistic locking
  blueprint.go      — Blueprint type, embed loader, prefix matching
  guided_action.go  — In-code guided follow-up session model
  snippet.go        — SDK snippet fetcher (docs.stripe.com)
  blueprints/       — Embedded JSON blueprints
  colors/           — Sail Design System palette helpers
  followups/        — Built-in guided follow-up definitions

pkg/coop/tui/
  app.go            — tea.Program entry points
  model.go          — Bubbletea model and Update loop
  view.go           — Top-level rendering
  commands.go       — Async commands (polling, snippets, session discovery)
  completion.go     — Post-completion suggestion view
  detail.go         — Detail panel rendering
  keymap.go         — Keyboard bindings
  layout.go         — Responsive layout calculations
  markdown.go       — Glamour rendering helpers
  mouse.go          — Mouse interactions
  outline.go        — Step/node outline rendering
  review.go         — Review card rendering
  selection.go      — Navigation and selection helpers
  helpers.go        — Word wrap, formatting, browser open
  messages.go       — Custom message types
  theme.go          — Sail Design System colors

pkg/coop/workflow/
  service.go        — Shared lifecycle operations for agent commands and TUI review actions

pkg/coop/helpers/
  nextaction.go     — Post-completion suggestions, environment detection, and next-action responses
  prompt.go         — Shared Huh prompt helpers using Sail-styled prompts
  review.go         — Shared step-review navigation rules

pkg/cmd/coop/
  coop.go           — Parent command, subcommand registration, command-package options
  coop_start.go     — User-facing orchestrator (tmux launcher)
  harness.go        — Supported agent registry and --agent resolution
  coop_launcher.go  — Agent detection and tmux/process management
  coop_run.go       — Agent-facing session creator
  coop_agent.go     — Typed agent lifecycle commands
  coop_join.go      — TUI launcher
  coop_status.go    — Session status display
  coop_stop.go      — End session
  coop_recommend.go — Blueprint discovery
```

## Local Harness Artifacts

The repository-level `bin/` directory remains ignored. Tmux harness isolation files under `bin/` are treated as local development artifacts unless a specific script or fixture is moved into a tracked source path with tests. Do not commit the ignored `bin/` directory wholesale.
