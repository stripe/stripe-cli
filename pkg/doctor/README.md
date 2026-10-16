# pkg/doctor — the migration doctor

`stripe doctor`: rule-based detection of Stripe API migrations, judged
against live account facts. Diagnosis is the root action; the agent
playbook is nested beneath it. doctor never edits code — each finding's
verdict message names the exact change to make by hand.

```bash
stripe doctor dpm .              # diagnose: findings + account verdicts (read-only)
stripe doctor guide              # the agent playbook (--json + exit-code contract)
```

`dpm` is the only topic today — the steel thread. Other migrations
previously lived here as detect-only "advise" topics and were removed to
keep this tool scoped to one working path; they may return later (see git
history).

## Architecture

Five packages, each a compiler-enforced boundary around one concern. Only
the top-level `doctor` package (`cmd.go`, `render.go`, `reports.go`) depends
on all of them; none of them depend on it or on each other except where
listed.

1. **`catalog`** — pure data: `Rule`/`ParamMatch`/`Finding`, the `Packs`
   topic registry (today: just `dpm`), and `ParseAPIRule`, which parses one
   rule in the shape the live doctor_rules API serves it in. `DpmRule` is
   parsed from an embedded snapshot (`catalog/dpm.json`) at startup — the
   offline/fallback source; `judge.FetchDPMRule` fetches the live version.
   `DpmSignals` (`dpm_signals.go`) is hardcoded Go, patched onto whichever
   Rule wins, since the API has no signals field yet. Depends on nothing
   else in this module.
2. **`scan`** — the AST-based matching engine: cgo-free tree-sitter over 7
   languages (Ruby, Python, PHP, JS/TS, Go, Java, C#). Resolution follows
   variables, typed builders, and target-typed declarations to the Stripe
   call site; matching alone never produces advice. Depends on `catalog` for
   the `Rule`/`Finding` shapes.
3. **`signals`** — substring code-signal scanning: expected webhook events.
   Honest about its strength — these are signals, never verdicts. Depends
   on `catalog` for the declared-signal shapes and `scan` only to know
   which file extensions it can read. (Frontend tokens, package-version-
   floor checks, and the which-migration triage aren't supported pending
   the removed non-dpm topics returning; see git history.)
4. **`judge`** — fetches read-only account facts and judges findings against
   them: the API versions recent traffic actually runs at, the governing
   payment-method configuration (toggle AND capability availability). Also
   owns `FetchDPMRule` (the live rule fetch), since it already has the
   credential/HTTP plumbing. Depends on `catalog` for the `Rule`/`Finding`
   shapes.
5. **`drill`** — the live webhook round-trip check behind `doctor --live`.
   Fully standalone.

`doctor` degrades to scan-only without credentials, honestly labeled, by
calling `judge.ScanOnly` instead of `judge.Judge`.

## Contracts

- `--json`: one JSON object on stdout, logs on stderr (`reports.go` is the
  schema). Exit codes: 0 clean/verified, 1 findings/not-verified, 2 error.
- Account access, and the live dpm rule fetch, are read-only GETs: the
  CLI's stored user access token (from `stripe login`) is preferred,
  falling back to an API key. Calls run in whichever mode (test or live)
  the active context is actually logged into — no `--live` flag needed;
  `judge.AccountSummary.Livemode` (JSON: `account.livemode`) says which.
  `--offline` skips both and uses the embedded rule. `--stripe-account`
  scopes Connect direct-charge diagnosis to the connected account.
- The doctor_rules endpoints aren't publicly shipped yet, so
  `judge.FetchDPMRule` pins `Stripe-Version: 2026-10-12-16.internal` on
  that call only — remove once the API is GA.
- Binary size: `make build` and the release configs always pass the
  `grammar_subset` build tags (see GRAMMAR_TAGS in the Makefile), embedding
  only the grammars for the languages the doctor supports (+10 MB over the
  base CLI). A bare `go build` without tags still works but embeds every
  gotreesitter grammar (+27 MB) — use `make build`.

## Adding a migration

Add a `Pack` to `catalog.Packs` — a rule (parameters + operations + action)
and optionally signals (expected webhook events). The verbs never change.
