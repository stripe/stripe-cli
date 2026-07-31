# Node contracts

A blueprint is a sequence of steps; each step contains nodes. Nodes are
addressed by a 1-based `--step` task number across the whole session. The
`start-work` response for a node carries everything below.

## Node types

- `apiRequest` — implement app code that makes this Stripe API call using the
  official SDK or the project's existing Stripe client pattern. Decide from the
  task whether the call belongs in runtime code, a setup/seed script, or
  one-time provisioning. The CLI is for inspection, temporary test data, and
  verifying the app code — not the implementation.
- `asyncHandler` — implement the app's webhook/async handler for every event
  listed in `events`. Read the raw request body, verify the Stripe signature
  with the official SDK helpers and the webhook secret from the environment,
  branch on each listed event type, and act on the data the app needs. Test
  with `stripe listen --forward-to localhost:<actual-app-port>/webhook` and
  `stripe trigger <event>` when the event has a supported trigger; otherwise
  exercise the app flow that emits it. Do not assume port 4242.
- `uiComponent` — build or update the user-facing surface that starts,
  redirects to, or displays this part of the flow. Verify it through the
  running app, and give the developer the URL, the action to take, and the
  expected result.
- `testHelper` — verify the integration end to end. `test_requests` advance
  Stripe test state (test clocks, test helpers, fixtures) and are run as test
  setup — their parameters belong to the test harness, never to application
  code.
- `cliCommand` — run the requested CLI command and report its concrete result.
  The only node type where no app code may be required.
- `dashboard` — complete the requested Stripe Dashboard configuration and
  verify the resulting state (via API/CLI where possible).
- `setUpWebhooks` — configure the requested webhook destination and verify the
  application receives the listed events.

For `apiRequest`, `asyncHandler`, and `uiComponent`, a node is complete only
when the app has working code for the behavior, wired into the app's existing
routes/services/conventions, and verification exercised the app code — not
only a direct CLI/API call.

## Structured context fields

- `api_request` — `{method, path, params}` for the exact Stripe call. Params
  may contain `${node...}` references already resolved to real values.
- `sdk_example` — a code snippet for the call in the session's language. A
  starting point: adapt it to the project's conventions.
- `test_requests` — API fixtures (`{method, path, params}`) to run as test
  setup for `testHelper` nodes. Execute them as test tooling — the Stripe CLI
  (e.g. `stripe post /v1/test_helpers/... -d key=value`) or a short test
  script — never as application code.
- `events` — the webhook event types an `asyncHandler` must handle, with any
  payload details.

## Outputs and `${node...}` references

Later nodes reference earlier results with
`${node.<step-key>.<node-key>[:...]:<field>}` placeholders. Co-op resolves
them from outputs you report, so:

- `start-work` lists `required_outputs` for the current node — the values
  later nodes need. Each has an optional `source` (a named/numbered request
  result; empty means the node's primary result) and a `field` path.
- Report each one on `report-work` with `--output field=value` or
  `--output source:field=value`. Values must be the **real observed values**
  (IDs, URLs, keys of created objects) — never placeholders or invented data.
  `report-work` fails until all required outputs are supplied.
- If a resolved value appears in a later node's context, use it as-is. Never
  ship an unresolved `${node...}` or `${env...}` placeholder into application
  code, config, or a command. If a `${node...}` placeholder reaches you
  unresolved, look the value up in the recorded outputs of the referenced node
  via `stripe coop status --session=<id> --json` (each node's `outputs` map),
  or report the missing `--output` on that node first. `${env...}` values come
  from the environment (`${env:livemode}` is `false` in test-mode sessions).

## Reviews

Nodes in a step are reviewed together as a step. `report-work` tells you
whether to continue the step or await review. Auto-confirmed (informational)
nodes skip human review — continue immediately. Before awaiting review, run
the node's verification command when one is given, keep useful servers
running, and share concrete actions and expected results for the developer.
