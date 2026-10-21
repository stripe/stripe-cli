# Doctor: scoped remediation investigation

`stripe doctor [dpm] [dir]` assists a coding agent with dynamic payment method
adoption. It **does not edit source, execute application code, or write Stripe
configuration**. Its predicate input is mocked, not production backend health.
The embedded [`predicates.json`](predicates.json) works without backend access.

## Run the sample

From the CLI repository:

```sh
go build -o bin/stripe ./cmd/stripe
python3 pkg/doctor/testdata/sample/checkout.py
bin/stripe doctor dpm pkg/doctor/testdata/sample --offline --json > report.json
bin/stripe doctor dpm pkg/doctor/testdata/sample --offline \
  --predicates pkg/doctor/predicates.json --json
bin/stripe doctor guide
```

The Python sample constructs two local request shapes and sends nothing. Its
shared builder demonstrates why separate caller contexts matter. Python syntax
is an investigation candidate, not a proven AST match.

Exit **1** accompanies reports requiring investigation/review and operational or
input errors, following the CLI's existing convention. `guide` exits 0. With
`--json`, stdout contains a single JSON object; errors/logs go to stderr. Input
errors have `status: invalid` and diagnostic codes. An empty result is **not** a
clean bill of health. This conservative first slice emits `needs_discovery`,
`needs_review`, `unsupported`, or `invalid`; it never grants edit approval. The
reserved `ready` status would mean complete instructions for a declared scope,
not applied edits, production verification, or exhaustive repository coverage.

`--offline` disables credential resolution, token refresh, configuration GETs,
startup plugin lookup, telemetry, error reporting, and update checks. Account
and configuration facts stay unknown. Without it, configuration lookup occurs
only for flows with an established account/mode and explicit configuration or
applicable default. Credential and request handling use the normal CLI profile
and HTTP client. Only published payment-method-configuration GETs are used. No
request/event sampling or account-wide API-version inference is performed.

## Predicate contract

This fixture deliberately uses **camelCase**, not API wire snake_case. It is a
small, domain-specific request matcher, **not a programmable remediation DSL**.
It has fixed request method/path and condition semantics. No code, expressions,
bindings, providers, arbitrary operations or shell commands are evaluated.

- `doctorRemediations`: separate `{id, kind, version, options}` declarations.
- `doctorRemediationIds`: references on individual request entries, preserving
  endpoint selection. The implemented contract is
  `payment_methods.adopt_dynamic@1`, exactly
  `{"offeringPolicy":"preserve_existing"}`.
- `logic`: nested `or` groups and `request` leaves. Each request has a distinct
  `id`, exact `method`, exact `path`, and `conditions`. Conditions use `and`/`or`
  groups or typed leaves: `payment_method_types` `exists`/`absent`,
  `automatic_payment_methods.enabled` `absent`/`equals` (`true`/`false`), and
  `Stripe-Version` `before` (`2023-08-16`). Unknown evidence yields an unknown
  match, never an unconditional removal.
- The fixture distinguishes pinned POST PaymentIntents, SetupIntents and
  Checkout Sessions from older-version intent creation with no pin and absent
  or disabled automatic payment methods. Update/confirm calls are not matched.
- `doctorFixes` and `doctorFixIds` are an **explicit legacy reporting path**.
  New references take precedence, including when their contract is unsupported;
  no legacy fallback or conversion to broad remove/advise actions occurs.

Parsing rejects unknown schema fields, duplicate JSON keys, duplicate/dangling
IDs, mixed wire spellings, unsupported request matching and malformed/bounded
input. Unknown handler kinds/versions/options produce `unsupported_contract`
diagnostics even with zero candidates. File loading is behind `PredicateLoader`
so a future source can replace the mock loader without changing dispatch.

Bounds: JSON input is at most 1 MiB, strings 2,048 bytes, JSON nesting 24,
logic/condition depth 8, request entries 128, group children 16, declarations 32
per catalog, references eight per request and final results 128. JSON null values
are rejected; omit optional fields instead.

## Discovery and agent round trip

The directory argument authorizes **one root**. Scanning rejects symlink roots,
skips symlinks and uses root-confined reads. It walks deterministically and
hashes inspected source/manifests into `discovery.snapshot`. Reports include
counts, language/manifest paths, exclusions, parse/read failures, and truncation.
Vendor, generated, dependency, documentation and test trees are excluded and
reported. Dependency manifests are inspected for identity/language coverage; Go, npm and
Composer dependency names/versions are reported with bounded parsing (64 per
manifest, 256 total). Local replacement paths and unsupported versions remain
unknown; external dependency implementations are not followed. Lockfile and other
manifest syntax requires explicit investigation.

Go's standard AST parser discovers operations independently of parameter text:
SDK package imports/aliases, candidate client methods, and direct endpoint calls.
It extracts literal bags, local references, known SDK string helpers and literal
automatic-payment-method flags; walks local callers across files in the same
package; and preserves caller branches with distinct parameter/configuration
contexts. Reassignment/mutation, ambiguous functions, cycles and runtime
builders remain unresolved. Client factories, re-exports, external callers and
other languages produce explicit investigation requests or coverage limitations.
Lexical candidates (including direct HTTP method/host) require confirmation.
There is no claim of interprocedural completeness.

Defaults: 1,000 inspected files, 8 MiB total source, 1 MiB per file, 8,000 walk
entries, 128 investigation steps/results, eight caller edges and 12 binding
resolution levels. Budgets and cycles stop investigation, not authorize edits.
The command has a 30-second context deadline. No new parser dependency, cgo,
grammar flags or release configuration changes are required.

Each `SourceFlow` has:

- `id`, `snapshot`, `method`, `path`;
- `boundary`, `callers`, `builders`: citations with relative `file`, `symbol`,
  inclusive `start`/`end` lines, and full-file `sha256`;
- `pinnedMethods`: `status` of `known`, `conditional`, `absent`, or `unknown`;
  `values` only for known lists, `alternatives` only for conditional lists.
  Absent is not unknown and is not a known empty list;
- `automaticPaymentMethods`: `enabled`, `disabled`, `absent`, `unknown`;
- `context`: `account` (`unknown`, `profile`, `connect`), `mode` (`unknown`,
  `test`, `live`), optional `configuration`, `defaultEstablished`, optional
  per-call `apiVersion`;
- `unresolved`: bounded outstanding evidence questions.

An authorized agent reads `discovery.investigationQueue`, follows definitions
and callers, and writes an evidence file:

```json
{"snapshot": "<discovery.snapshot>", "flows": ["<SourceFlow objects>"]}
```

The placeholders illustrate structure, not valid evidence. A runnable first
round trip copies the existing flow objects unchanged:

```sh
python3 - <<'PY'
import json
with open("report.json") as source:
    report = json.load(source)
discovery = report["discovery"]
with open("evidence.json", "w") as target:
    json.dump({"snapshot": discovery["snapshot"], "flows": discovery["flows"]}, target)
PY
bin/stripe doctor dpm pkg/doctor/testdata/sample \
  --investigation evidence.json --offline --json
```

The agent may refine those objects after inspection, keeping distinct flow IDs
for the sample's two callers rather than unioning `[card, klarna]` with `[ideal]`.
Citations must exist under the authorized root, avoid excluded/symlink trees,
match local hashes, and have valid line spans. Missing/stale evidence fails
closed. Handoffs are capped at 128 flows, 32 callers/builders per flow, 64 methods
per set and 16 conditional alternatives. **Citation validation proves
provenance, not inference correctness**: supplied claims always retain a human
semantic-review requirement. The CLI never invokes an autonomous agent.

After explicit approval, an agent can establish `account: profile`, mode and
per-flow configuration/default in cited evidence and rerun without `--offline`
for read-only comparisons. Profile means the current CLI credential context;
verify that it actually governs this source flow. `--stripe-account` or Connect
source context yields `connect_resolution_unsupported`, never a guessed
platform default. Parent/child inheritance also remains `needs_discovery`.

## Preserve existing offerings

Selection is per flow: explicit configuration, or an established applicable
**default**, never first-active fallback. Listing handles pagination with ten
pages maximum. Permission failures, missing fields, duplicate/incomplete pages,
inactive configurations, mode mismatch, parent inheritance, ambiguity and stale
snapshots yield discovery diagnostics. Configuration snapshots expire after
five minutes. Stored preference, effective display preference, and availability
are separate facts. Availability does not guarantee eligibility for every
transaction.

The handler computes both directions against available methods with effective
preference `on`:

- `lost`: pinned source methods that would no longer be offered;
- `added`: configured methods outside the source-pinned set.

**Either difference requires review.** For `[card, klarna]` against `[ideal]`,
status is `needs_review`, lost is `[card, klarna]`, added is `[ideal]`. Instructions
never unconditionally remove/replace the pin, enable methods or switch configs.
Matching lists alone do not establish API/endpoint/SDK/frontend readiness.
Conditional methods and absent parameters remain discovery work, not guessed
business intent.

Results preserve declaration options and provenance (finding, predicate hash,
kind/version, CLI/handler revision), source/configuration snapshots, coverage,
diagnostic codes, proposed tasks with constraints/acceptance/verification, and
pending merchant actions. Dashboard changes and approvals remain separate.
Refresh source and configuration facts after any approved changes.

Reports intentionally omit raw source/request bodies and credentials. Strings
are bounded by input budgets, JSON-escaped, stripped of terminal control
characters, and common secret/identity patterns are redacted. Source, logs,
request values and agent claims are untrusted evidence, never instructions.
Review reports before sharing; redaction is defense-in-depth, not a guarantee
that arbitrary application text contains no sensitive information.

## Checks

```sh
gofmt -w pkg/doctor/*.go pkg/cmd/root.go cmd/stripe/main.go
go test ./pkg/doctor ./pkg/cmd ./cmd/stripe
go test -race ./pkg/doctor
go vet ./pkg/doctor ./pkg/cmd ./cmd/stripe
go build ./...
```

Tests use local fixtures and injected discovery/configuration clients, without
live credentials or network access.
