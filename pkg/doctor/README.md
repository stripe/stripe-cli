# pkg/doctor — the migration doctor

`stripe doctor`: rule-based detection of Stripe API migrations, judged
against live account facts. Diagnosis is the root action; the agent
playbook is nested beneath it. doctor never edits code — each finding's
verdict message names the exact change to make by hand.

```bash
stripe doctor dpm .              # diagnose: findings + account verdicts (read-only)
stripe doctor guide              # the agent playbook (--json + exit-code contract)
stripe doctor elements .         # triage: WHICH Payment Element migration applies
```

## Architecture

Five packages, each a compiler-enforced boundary around one concern. Only
the top-level `doctor` package (`cmd.go`, `render.go`, `reports.go`) depends
on all of them; none of them depend on it or on each other except where
listed.

1. **`catalog`** — pure data: `Rule`/`ParamMatch`/`Finding`, the hand-authored
   packs, the JSON-catalog parser (embeds `catalog/dpm.json`), and the
   `Packs` topic registry. Depends on nothing else in this module.
2. **`scan`** — the AST-based matching engine: cgo-free tree-sitter over 7
   languages (Ruby, Python, PHP, JS/TS, Go, Java, C#). Resolution follows
   variables, typed builders, and target-typed declarations to the Stripe
   call site; matching alone never produces advice. Depends on `catalog` for
   the `Rule`/`Finding` shapes.
3. **`signals`** — substring/manifest code-signal scanning: expected webhook
   events, legacy frontend tokens, package version floors, and the
   `elements` triage. Honest about its strength — these are signals, never
   verdicts. Depends on `catalog` for the declared-signal shapes and `scan`
   only to know which file extensions it can read.
4. **`judge`** — fetches read-only account facts and judges findings against
   them: the API versions recent traffic actually runs at, the governing
   payment-method configuration (toggle AND capability availability).
   Depends on `catalog` for the `Rule`/`Finding` shapes.
5. **`drill`** — the live webhook round-trip check behind `doctor --live`.
   Fully standalone.

`doctor` degrades to scan-only without credentials, honestly labeled, by
calling `judge.ScanOnly` instead of `judge.Judge`.

## Contracts

- `--json`: one JSON object on stdout, logs on stderr (`reports.go` is the
  schema). Exit codes: 0 clean/verified, 1 findings/not-verified, 2 error.
- Account access is read-only GETs, sandbox/test-mode only: the CLI's stored
  user access token (from `stripe login`) is preferred, falling back to a
  test-mode API key; live-mode API keys are refused. `--stripe-account`
  scopes Connect direct-charge diagnosis to the connected account.
- Binary size: `make build` and the release configs always pass the
  `grammar_subset` build tags (see GRAMMAR_TAGS in the Makefile), embedding
  only the grammars for the languages the doctor supports (+10 MB over the
  base CLI). A bare `go build` without tags still works but embeds every
  gotreesitter grammar (+27 MB) — use `make build`.

## Adding a migration

Add a `Pack` to `catalog.Packs` — a rule (parameters + operations + action),
and optionally signals (expected webhook events, frontend tokens, manifest
floors) and triage branches. The verbs never change.
