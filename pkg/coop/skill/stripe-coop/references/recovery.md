# Reviews, timeouts, and recovery

## Await-review

Run `stripe coop agent await-review` only when a response's `next` says to.
It blocks until the developer decides, up to `wait_timeout_seconds`
(currently 300 seconds). Always give your shell tool at least one minute more
than `wait_timeout_seconds` (e.g. 360s) so the structured timeout response
can arrive instead of the harness killing the command.

Possible outcomes (`state` field):

- `confirmed` — the step was approved. Run the returned `next` command.
- `timeout` — nobody decided yet. This is normal: re-run the same
  `await-review` command and keep waiting. Do not treat a timeout as
  rejection or move on.
- `rejected` (or a message saying changes were requested) — the response
  `message` contains the developer's feedback and the `next` command points
  at the node to redo. Use that feedback directly; never ask the developer to
  repeat feedback that is already in the response. Redo the affected work
  from the feedback, re-verify, and report again.
- "already …" — the node moved on while you were away (e.g. a wake-up
  handled it). Follow the returned `next`.

## Wake-ups and `resume`

While you are blocked (or after your await timed out), the TUI may inject a
message telling you human input updated the session. Respond by running the
exact `stripe coop agent resume --session=<id>` command it gives. `resume` is
read-only and safe to repeat. If its JSON has a non-empty `next`, run that
command exactly; if the continuation is empty, another handoff already
advanced the session — continue your current work.

Use `resume` too whenever you lose track of session state (restarts, context
loss): it returns the single correct continuation for the current state.

## Completion and follow-ups

When every node is done, run
`stripe coop agent next-action --session=<id>`. It waits up to 10 minutes per
call for the developer to choose; on timeout, re-run it. The response for the
selected action is the instruction: either an `agent_prompt` to carry out
directly (e.g. writing a summary), or — for guided actions like deploys — a
`next` command telling you to run
`stripe coop agent start-followup --session=<parent> --action=<id>`, which
creates a new guided session worked with the same node lifecycle. Either way,
record the finished action with `next-action --completed=<action-id>` on the
parent session.

## Errors

Every failure response contains `error` plus `recovery.hint` and a recovery
continuation. Read the hint, fix the cause, and continue from the recovery
command — usually `stripe coop status` to re-inspect state. Do not retry a
failed command unchanged, and never work around a failure by editing Co-op's
session files, heartbeat files, or any internal state on disk. If the session
is aborted or unrecoverable, tell the developer and stop.

## Authentication problems

If commands fail with authentication errors, run `stripe whoami`. If not
authenticated or test-mode keys are unavailable, run
`stripe sandbox create --from-git` — the claim URL surfaces in the
developer's TUI automatically. Never paste secret keys into files or shell
history to work around auth.
