# pkg/doctor — the migration doctor

`stripe doctor`: rule-based detection of Stripe API migrations, judged
against the migration rule itself (fetched live). Diagnosis is the root
action; the agent playbook is nested beneath it. doctor never edits code —
each finding's verdict message names the exact change to make by hand.

```bash
stripe doctor dpm .              # diagnose: findings + verdicts (read-only)
stripe doctor guide              # the agent playbook (--json + exit-code contract)
```

`dpm` is the only topic today — the steel thread. Other migrations
previously lived here as detect-only "advise" topics and were removed to
keep this tool scoped to one working path; they may return later (see git
history).

## Architecture

Several packages, each a compiler-enforced boundary around one concern.
Only the top-level `doctor` package (`cmd.go`, `render.go`, `reports.go`)
depends on all of them; none of them depend on it or on each other except
where listed.

Packages 1-5 below live under `internal/` — doctor's own generic engine,
not meant to be imported by anything outside this tree (Go enforces
that). `remediation` and `remediation/dpm` are deliberately outside
`internal/`: that's the plugin boundary other teams add their own topic
to, by following the same pattern dpm does — see `remediation/README.md`.

1. **`internal/catalog`** — pure data: `Rule`/`ParamMatch`/`Finding`, the `Topics`
   set of known topic names (today: just `dpm`, used for CLI arg
   validation and typo-hints — not rule data), and `ParseAPIRule`, which
   parses one rule in the shape the doctor_rules API serves it in,
   setting `Rule.RemediationTopic` (the lookup key into
   `pkg/doctor/remediation`) to the rule's own `topic` — the API has no
   separate field for this. The rule data itself is never kept here —
   `cmd.go` fetches it live via `judge.FetchRule` on every run. No
   code-signals concept lives here anymore — see `remediation/dpm`
   below. Depends on nothing else in this module.
2. **`internal/scan`** — the AST-based matching engine: cgo-free tree-sitter over 7
   languages (Ruby, Python, PHP, JS/TS, Go, Java, C#). Resolution follows
   variables, typed builders, and target-typed declarations to the Stripe
   call site; matching alone never produces advice. Depends on `catalog` for
   the `Rule`/`Finding` shapes.
3. **`internal/codesignals`** — substring code-signal scanning: a caller declares
   its own `PackSignals` (expected webhook events) and calls
   `ScanSignals`; honest about its strength — these are signals, never
   verdicts. Depends only on `scan`, to know which file extensions it can
   read — not on `catalog`, since code signals aren't a rule-level
   concept. (Frontend tokens and package-version-floor checks aren't
   supported yet.)
4. **`internal/stripeapi`** — the minimal read-only Stripe API client shared by
   `judge` (fetching the rule) and remediation implementations (fetching
   whatever account evidence their topic needs). Resolves LIVE MODE
   credentials only — a test-mode account's facts aren't a meaningful
   stand-in for the real account state these calls exist to check. Knows
   nothing about doctor's engine.
5. **`internal/judge`** — judges findings against the migration rule alone, purely
   by intent (CANDIDATE/SKIP/REVIEW, or ADVISE for a detect-only rule): no
   live account facts and no remediation-level guidance are part of this
   judgment anymore — that's entirely a remediation's own concern now
   (see `remediation/dpm`). Also owns `FetchRule` (the live rule fetch,
   via `stripeapi`), which `cmd.go` calls on every run to get the
   `Rule` it judges against. Depends on `catalog` for the `Rule`/`Finding`
   shapes.
6. **`remediation`** — the plugin boundary for `Rule.RemediationTopic`
   (the rule's own topic string, e.g. `"dpm"`): it maps directly
   to a Go `Implementation` via a `Register`/`Lookup` registry, so teams
   that own a topic can implement its remediation without doctor's engine
   knowing anything about them. There's no "fixes" field or per-remediation
   `Kind`/`Options` data on the rule describing what to do — a topic
   defines all of that directly in its own Go code. See the package doc
   comment for the three-step guide to plugging one in, and
   **`remediation/dpm`** for dpm's worked example, which
   hardcodes its own Dashboard-link guidance and webhook signals, and
   reassembles recent API versions, the Dashboard's payment-method
   configuration, and the scanned findings into one readiness `Result`.
7. **`internal/webhookcheck`** — the live webhook round-trip check behind
   `doctor --verify-webhooks`. Fully standalone.

## Contracts

- `--json`: one JSON object on stdout, logs on stderr (`reports.go` is the
  schema). Exit codes: 0 once the report completes — findings present or
  an unverified webhook check are reported outcomes, not failures, so
  check the report's contents rather than the exit code for those; 2
  means doctor couldn't produce a report at all (bad arguments, a scan
  error, a webhook-check operational failure).
- `remediation_result` is what `cmd.go`'s `runRemediation` actually
  computed by looking up the rule's `RemediationTopic` in the `remediation`
  registry — `unknown` status if the topic has no registered
  `Implementation` yet, or if gathering its evidence failed (e.g. no
  credentials); never silently dropped.
- Any read-only GET against the live Stripe API (today: `stripeapi`'s
  callers) requires the active context to actually be in live mode — a
  test-mode context makes the call fail. For `judge.FetchRule` (needed
  just to get the rule to diagnose against) that's a hard failure: the
  whole run exits 2 with no report. For a remediation's own evidence
  gathering it's soft: that topic's result comes back `status: unknown`
  naming the mismatch, rather than failing the whole run.
  `--stripe-account` scopes these lookups to a Connect connected
  account.
- The doctor_rules endpoints aren't publicly shipped yet, so
  `judge.FetchRule` pins `Stripe-Version: 2026-10-12-16.internal` on
  that call only — remove once the API is GA.
- Binary size: `make build` and the release configs always pass the
  `grammar_subset` build tags (see GRAMMAR_TAGS in the Makefile), embedding
  only the grammars for the languages the doctor supports (+10 MB over the
  base CLI). A bare `go build` without tags still works but embeds every
  gotreesitter grammar (+27 MB) — use `make build`.

## Adding a migration

Add the topic name to `catalog.Topics`, and make sure the doctor_rules
API has that topic's rule (parameters + operations + action) configured
server-side — `judge.FetchRule` fetches it live, there's no local rule
data to author in this repo. The verbs never change. Code signals
(expected webhook events, etc.) aren't part of this registry: a
remediation that wants them declares and scans for its own, below.

## Adding a remediation for a topic

See `pkg/doctor/remediation/README.md` for the full integration guide and
function reference; short version: implement `remediation.Implementation`,
`Register` it (under your rule's own topic string) from an `init()` in
your own subpackage (model it on `remediation/dpm`), and
blank-import that subpackage from `cmd.go`.
