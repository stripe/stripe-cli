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
│  │  every 500ms         │   │  stripe coop step commands │ │
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
   │          └──→ skipped                  (agent decides step doesn't apply)
   │
   └──→ skipped                             (agent skips from pending)
```

**Terminal states:** `done`, `skipped` — no transitions out.

**Transitions are validated** — `session.TransitionStep()` returns an error for invalid transitions (e.g. pending→done, done→active).

## Session States

```
active ──→ completed    (all steps done/skipped, or "stripe coop stop")
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
| `stripe coop recover` | Diagnose and fix stuck sessions |
| `stripe coop recommend` | List available blueprints |

### Agent-facing (AI agent runs these)
| Command | Purpose |
|---------|---------|
| `stripe coop run <blueprint>` | Create a session (outputs JSON with instructions) |
| `stripe coop step <n> start` | Mark step as active |
| `stripe coop step <n> done` | Mark step as complete (→ review or → done if auto_confirm) |
| `stripe coop step <n> verify` | Add a verification check |
| `stripe coop step <n> skip` | Skip a step |
| `stripe coop step <n> await` | Block until developer confirms or requests changes |
| `stripe coop next-steps` | Show post-completion options (blocks until selection) |

All agent commands output JSON with an `ok` field and a `next` field suggesting the next command.

## TUI Keybindings

| Key | Action |
|-----|--------|
| `↑`/`k` | Move cursor up |
| `↓`/`j` | Move cursor down |
| `e` / `Enter` | Toggle detail panel for selected step |
| `c` | Confirm the selected review item |
| `r` | Request changes for the selected review item |
| `f` | Resume following the active/review step after manual navigation |
| `o` | Open claim URL in browser (when sandbox is unclaimed) |
| `q` / `Ctrl+C` | Quit TUI |

When requesting changes, `r` opens a feedback prompt. Press `Enter` to submit a note and move the reviewed step or section back to `active`; press `Esc` to cancel.

In the completion view:
| Key | Action |
|-----|--------|
| `↑`/`↓` | Navigate suggestions |
| `Enter` | Select a suggestion |
| `q` | Quit |

## Example Flow

```bash
# Developer starts a session (launches tmux with agent + TUI)
$ stripe coop start one-time-payment --language=node

# What happens behind the scenes:
# 1. Agent (Claude/Codex) is launched in right pane
# 2. TUI appears in left pane showing step progress
# 3. Agent runs: stripe coop run one-time-payment --language=node
# 4. Agent works through steps, calling:
#      stripe coop step 1 start --note="Scanning project"
#      stripe coop step 1 done --note="Found Next.js app"
#      stripe coop step 2 start --note="Creating product"
#      stripe coop step 2 done --file=server.js --lines=5-20 --note="Created product"
#      stripe coop step 2 await --session=coop_abc123   ← blocks until developer confirms
# 5. Developer sees progress live, presses 'c' to confirm
# 6. Agent continues to next step
# 7. After all steps: agent runs "stripe coop next-steps --session=coop_abc123"
# 8. Developer picks what to do next from TUI suggestions
```

Post-completion choices are written into the session file for the agent. Deploy follow-ups use `stripe coop run deploy-stripe-projects --parent-session=<id> --parent-step=<selection>` so the child session can return to the completed parent and mark that next step done.

## Auto-Confirm

Nodes with `"auto_confirm": true` skip human review:
- `step done` transitions directly to `done` (not `review`)
- `step await` returns immediately if the step is auto-confirmed
- The prepended "Understand the project" step is always auto-confirmed
- Blueprint nodes can set `"auto_confirm": true` for mechanical steps

## Heartbeat

When the agent runs `stripe coop step <n> await`, it writes a `.heartbeat` file every 500ms. The TUI checks this file:
- **Fresh heartbeat (< 5s old):** Agent is actively waiting for confirmation
- **No heartbeat + no session update in 2min:** Show idle warning

The heartbeat file is cleaned up when `await` exits.

## Recovery

`stripe coop recover` diagnoses common issues:

| Issue | Detection | Fix |
|-------|-----------|-----|
| Step stuck in `active` | No heartbeat, step is active | Recovery reports the active step; it does not move in-progress work automatically |
| Session done but not marked | All steps done, status=active | `--fix` marks completed |
| No active step | No active/review steps found | Shows next pending step number |
| Agent crashed | Heartbeat stale | TUI shows warning, user runs `recover --fix` |

## Blueprint Format

Blueprints are embedded JSON in `pkg/coop/blueprints/`. Each has:
- `id` — unique identifier (also the filename without .json)
- `title`, `description` — human-readable
- `prompt` — optional custom agent instructions (overrides generic preamble)
- `chapters` — ordered sections of nodes

Each node has:
- `type` — `apiRequest`, `asyncHandler`, `uiComponent`, `cliCommand`, `testHelper`
- `auto_confirm` — skip human review for this step
- `description` — what the agent should do (source of truth)
- `review_prompt` — what the human should check before confirming
- `request` — API request details (for `apiRequest` nodes with SDK snippet support)
- `events` — webhook events (for `asyncHandler` nodes)

Sections may set `review_granularity` to `chapter` to group multiple reviewable nodes into one human approval milestone.

### Custom Agent Prompt

The optional `prompt` field overrides the generic agent instructions. Use it when a blueprint isn't about writing application code:

```json
{
  "id": "deploy-stripe-projects",
  "prompt": "The developer wants to configure their project for deployment using the Stripe Projects CLI plugin. Use `stripe projects --help` to discover available options...",
  "chapters": [...]
}
```

Without `prompt`, the agent gets: "You are building a working Stripe integration: <title>"

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
  theme.go          — Sail Design System colors + HuhTheme

pkg/cmd/
  coop.go           — Parent command, subcommand registration
  coop_start.go     — User-facing orchestrator (tmux launcher)
  coop_launcher.go  — Agent detection, tmux management, prompts
  coop_run.go       — Agent-facing session creator
  coop_step.go      — Step lifecycle (start/done/verify/skip/await)
  coop_join.go      — TUI launcher
  coop_nextsteps.go — Post-completion flow
  coop_env.go       — Environment detection, suggestion building
  coop_status.go    — Session status display
  coop_stop.go      — End session
  coop_recover.go   — Diagnose and fix stuck sessions
  coop_recommend.go — Blueprint discovery
```

## Local Harness Artifacts

The repository-level `bin/` directory remains ignored. Tmux harness isolation files under `bin/` are treated as local development artifacts unless a specific script or fixture is moved into a tracked source path with tests. Do not commit the ignored `bin/` directory wholesale.
