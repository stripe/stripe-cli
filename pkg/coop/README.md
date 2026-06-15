# Co-op Mode

Co-op mode enables an AI agent and a human developer to build Stripe integrations together in real time. The agent writes code and reports progress; the developer watches in a terminal UI, reviews work, and confirms each step.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ tmux session                                                │
│                                                             │
│  ┌──────────────────────┐   ┌────────────────────────────┐ │
│  │  TUI (coop join)     │   │  Agent (Claude/Codex)      │ │
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

## Step State Machine

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
| `stripe coop agent start-work --step <n>` | Mark node as active |
| `stripe coop agent report-work --step <n>` | Mark node as complete (→ review or → done if auto_confirm) |
| `stripe coop agent report-check --step <n>` | Add a verification check |
| `stripe coop agent skip --step <n>` | Skip a node |
| `stripe coop agent await-review --step <n>` | Block until developer confirms or requests changes |
| `stripe coop agent next-action` | Show post-completion options (blocks until selection) |

All agent commands output JSON with an `ok` field and a `next` field suggesting the next command.

## TUI Keybindings

| Key | Action |
|-----|--------|
| `↑`/`k` | Move cursor up |
| `↓`/`j` | Move cursor down |
| `e` / `Enter` | Toggle detail panel for selected step or node |
| `c` | Confirm the selected review item |
| `r` | Request changes for the selected review item |
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
# 2. Agent (Claude/Codex) is launched in right pane
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

Post-completion choices are written into the session file for the agent. Deploy follow-ups use `stripe coop run deploy-stripe-projects --parent-session=<id> --parent-step=<selection>` so the child session can return to the completed parent and mark that next step done.

## Auto-Confirm

Nodes with `"auto_confirm": true` skip human review:
- `agent report-work` transitions directly to `done` (not `review`)
- `agent await-review` returns immediately if the step is auto-confirmed
- The prepended "Understand the project" step is always auto-confirmed
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
| Step is active | Rejoin the session and check the agent pane/TUI state |
| Step is in review | Rejoin the session and confirm or request changes |
| Agent appears idle | Rejoin the session; the TUI shows heartbeat/idle state |
| Need a specific older session | Run `stripe coop join --resume` |

## Blueprint Format

Blueprints are embedded JSON in `pkg/coop/blueprints/`. Each has:
- `id` — unique identifier (also the filename without .json)
- `title`, `description` — human-readable
- `steps` — ordered groups of nodes

Each node has:
- `type` — `apiRequest`, `asyncHandler`, `uiComponent`, `cliCommand`, `testHelper`
- `auto_confirm` — skip human review for this node
- `description` — what the agent should do (source of truth)
- `review_prompt` — what the human should check before confirming
- `request` — API request details (for `apiRequest` nodes with SDK snippet support)
- `events` — webhook events (for `asyncHandler` nodes)

### Adding a Blueprint

1. Create `pkg/coop/blueprints/your-blueprint.json`
2. Follow the schema above
3. Test: `go run ./cmd/stripe coop run your-blueprint`
4. Prefix matching works: short prefixes resolve to full IDs if unambiguous

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| TUI shows "Agent appears idle" | Agent crashed or stopped | Check the agent pane; restart with `stripe coop start` |
| Agent stuck on "await" | Developer hasn't confirmed | Press `c` in TUI to confirm, or `r` to request changes |
| "Version conflict" error | TUI and agent wrote simultaneously | Agent retries the command (safe to re-run) |
| TUI shows wrong session | Multiple sessions exist | Use `stripe coop join <session-id>` with the correct ID |
| Steps not updating in TUI | Agent created a duplicate session | Check `stripe coop status` for the correct session ID |
| Agent ignores "next" hint | LLM didn't follow instructions | Copy the `next` value and run it manually, or restart |
| Double footer / layout broken | Terminal resize not detected | Resize the terminal window (triggers recalculation) |
| "Blueprint not found" | Typo in blueprint ID | Run `stripe coop recommend` to see available IDs |

## Optimistic Locking

`Store.Write()` checks the file's current version before writing. If another writer changed the file since you read it, the write fails with a version conflict error. This prevents the TUI and agent from clobbering each other's changes.

## File Structure

```
pkg/coop/
  types.go          — Session, Node, Step types and constants
  session.go        — State machine, validation, queries
  store.go          — Atomic file I/O, heartbeat, optimistic locking
  blueprint.go      — Blueprint type, embed loader, prefix matching
  snippet.go        — SDK snippet fetcher (docs.stripe.com)
  blueprints/       — Embedded JSON blueprints

pkg/coop/tui/
  app.go            — tea.Program entry points
  model.go          — Bubbletea model, Update, key handling
  view.go           — All rendering (header, steps, detail, footer, completion)
  commands.go       — Async commands (polling, snippets, session discovery)
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
