# pkg/doctor — the migration doctor

`stripe doctor`, `stripe fix`, and `stripe guide`: rule-based detection and
remediation of Stripe API migrations, judged against live account facts.

```bash
stripe doctor dpm .           # diagnose: findings + account verdicts (read-only)
stripe fix dpm . --apply      # remediate: span-verified edits, disclosure before consent
stripe guide                  # the agent playbook (--json + exit-code contract)
stripe doctor elements .      # triage: WHICH Payment Element migration applies
```

## Architecture

Three layers, deliberately separated:

1. **Scanner** (`scan.go`, `langs.go`, `resolve.go`, `span.go`) — cgo-free
   tree-sitter over 7 languages (Ruby, Python, PHP, JS/TS, Go, Java, C#).
   Rules are data (`rule.go`, `packs.go`): a parameter plus the API operations
   it belongs to, derived from the vendored OpenAPI spec. Resolution follows
   variables, typed builders, and target-typed declarations to the Stripe call
   site; matching alone never produces advice.
2. **Doctor** (`account.go`, `verdict.go`, `signals.go`, `drill.go`) — judges
   findings against read-only account facts: the API versions recent traffic
   actually runs at, the governing payment-method configuration (toggle AND
   capability availability), plus pack-declared code signals (webhook
   handlers, frontend tokens, package floors) and the `elements` triage.
   Degrades to scan-only without credentials, honestly labeled.
3. **Fix** (`fix.go`, `companion.go`) — gate-checked, span-verified edits.
   Decisions are per CALL SITE (a Java builder's whole method list is one
   unit). The dpm rule's companion forks by version evidence: below the
   2023-08-16 cutoff (or unknowable) removals become replacements inserting
   `automatic_payment_methods[enabled]=true`; server-side-confirmation sites
   additionally pin `allow_redirects:"never"`, take `--return-url`, or gate
   entirely when redirect-capable methods are at stake. Only reparse-clean
   files are written, and `--apply` renders the full disclosure before
   asking consent.

## Contracts

- `--json`: one JSON object on stdout, logs on stderr (`reports.go` is the
  schema). Exit codes: 0 clean/verified, 1 findings/not-verified, 2 error.
  `--apply --json` requires `--yes`.
- Account access is read-only GETs with test-mode keys only; live keys are
  refused. `--stripe-account` scopes Connect direct-charge diagnosis to the
  connected account.
- Binary size: the tree-sitter grammars embed all languages by default.
  Release builds should pass
  `-tags "grammar_subset grammar_subset_go grammar_subset_python grammar_subset_ruby grammar_subset_php grammar_subset_javascript grammar_subset_typescript grammar_subset_tsx grammar_subset_java grammar_subset_c_sharp"`
  (+10 MB instead of +27 MB over the base CLI).

## Adding a migration

Add a `Pack` to `packs.go` — a rule (parameters + operations + action), and
optionally signals (expected webhook events, frontend tokens, manifest
floors) and triage branches. The verbs never change.
