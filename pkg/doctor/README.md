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

Two layers, deliberately separated:

1. **Scanner** (`scan.go`, `langs.go`, `resolve.go`) — cgo-free tree-sitter
   over 7 languages (Ruby, Python, PHP, JS/TS, Go, Java, C#). Rules are data
   (`rule.go`, `packs.go`): a parameter plus the API operations it belongs
   to, derived from the vendored OpenAPI spec. Resolution follows
   variables, typed builders, and target-typed declarations to the Stripe call
   site; matching alone never produces advice.
2. **Doctor** (`account.go`, `verdict.go`, `signals.go`, `drill.go`) — judges
   findings against read-only account facts: the API versions recent traffic
   actually runs at, the governing payment-method configuration (toggle AND
   capability availability), plus pack-declared code signals (webhook
   handlers, frontend tokens, package floors) and the `elements` triage.
   Degrades to scan-only without credentials, honestly labeled.

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

Add a `Pack` to `packs.go` — a rule (parameters + operations + action), and
optionally signals (expected webhook events, frontend tokens, manifest
floors) and triage branches. The verbs never change.
