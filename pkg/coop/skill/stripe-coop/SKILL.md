---
name: stripe-coop
description: Operate an active Stripe CLI Co-op integration session. Use when a prompt or command mentions `stripe coop`, a `coop_*` session, or building a Stripe integration through Co-op's live human-review TUI. Follow the typed command protocol, implement one node at a time, and stop at review boundaries. Do not use for ordinary Stripe coding outside Co-op.
---

# Stripe Co-op

Co-op pairs you with a human developer to build a Stripe integration in their
application. The human watches and reviews your progress live in a terminal UI.
You drive the session through typed `stripe coop` CLI commands that return JSON.

**The deliverable is a working integration in the user's application** — not a
sequence of successful CLI commands. CLI commands are for setup, test data, and
verification; the implementation lives in the user's codebase unless a node is
explicitly a `cliCommand`.

## Core rules

1. **Work one node at a time.** Never work ahead of the current node.
2. **The current node's `agent_prompt` is the task source of truth.** Follow it
   over your own assumptions.
3. **Use only the typed control commands** (`start-work`, `report-check`,
   `report-work`, `skip`, `await-review`, `resume`, `next-action`,
   `start-followup`). Never invent state transitions, and never edit Co-op
   session JSON, heartbeat files, or any internal Co-op state directly.
4. **Follow the `next` field.** Every response tells you what to run next.
   `next` is an exact command — run it verbatim. `next_template` +
   `required_inputs` means you must substitute real values first; never run a
   command that still contains `<...>` placeholders.
5. **Stop at review boundaries.** When told to await review, run
   `await-review` and block. React to the decision the CLI returns; do not ask
   the human to re-type feedback that is already in the response.
6. **Report honestly.** Run meaningful checks; pass `--passed` only for results
   you actually observed. Report the main implementation file with useful
   line/snippet/note detail.
7. **Skip only genuinely inapplicable nodes**, with a concrete `--note` reason.

## Starting

- **No blueprint selected yet:** inspect the project first (language,
  framework, existing Stripe code). Ask the developer only for intent or
  constraints you cannot infer from the code. Then run
  `stripe coop recommend --all`, pick the best blueprint, confirm your choice
  with the developer in plain language, and run
  `stripe coop run <blueprint-id> --language=<lang>`.
- **Session already created:** run the exact bootstrap command you were given
  (usually `stripe coop agent start-work ...`) and follow each response.

In both cases, check API access before creating or working a session: run
`stripe whoami`. If not authenticated, or it shows "Test mode key: not
available", run `stripe sandbox create --from-git`. The claim URL appears in
the developer's TUI automatically.

## Working a node

1. `start-work` → read `agent_prompt`, plus the structured context:
   `api_request`, `test_requests`, `events`, and `sdk_example` describe the
   exact API call, test fixtures, webhook events, and SDK usage for this node.
   See `references/node-contracts.md` for node types and how to interpret each
   field.
2. Implement in the user's app, matching its existing conventions, framework,
   and dependency choices. Exercise the real application code when practical,
   not just direct API calls.
3. Verify, then `report-check` each meaningful check (short `--check` label,
   long output in `--detail`).
4. `report-work` with `--file`, `--lines`/`--snippet` and a concrete `--note`.
   Supply every `required_outputs` value via `--output` — later nodes resolve
   `${node...}` references from them (see `references/node-contracts.md`).
5. Run the response's `next` command. At review boundaries, prepare the
   developer first: for UI or server work keep the server running, and give the
   URL to open, the action to take, and the expected result.

Review, rejection, timeout, and wake-up handling: `references/recovery.md`.
Full command and response reference: `references/command-api.md`.

## After completion

When all nodes are done, run `next-action` and wait for the developer's
choice. Any follow-up work must go through `start-followup` — do not start
free-form work on a completed session.

## Safety

- Never hardcode secret keys, webhook secrets, or Stripe-shaped fake secrets
  in code, tests, or fallbacks. Read secrets from environment variables or the
  project's existing secret-management convention.
- Verify webhook signatures against the **raw request body** with the official
  SDK helpers.
- Never put full card numbers in application code, tests, or API calls. Use
  supported test PaymentMethod IDs (e.g. `pm_card_visa`) or official Stripe
  test helpers; collect real card details only through hosted/official
  client-side components.
- Keep CLI test-helper parameters (fixtures, triggers, test clocks) out of
  application code — they belong to test setup, not the integration.
