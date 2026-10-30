# Adding a remediation

A migration `Rule` (`pkg/doctor/internal/catalog`) names its own topic via
`catalog.Rule.RemediationTopic` — e.g. `"dpm"`. That
bare string is the *only* thing doctor's engine knows about remediations;
it has no opinion on what a topic's remediation actually does, how it
renders, or what evidence it needs. Everything else is yours to write in
your own subpackage here, under `pkg/doctor/remediation/<your-package>`.

This doc is the integration guide and function reference. For a full
worked example, read `pkg/doctor/remediation/dpm` (implements
the `"dpm"` topic) alongside this doc. For two small, runnable examples,
see `pkg/doctor/remediation/example` (`go test -run Example -v`) — they
compile and their output is checked by `go test`, so they can't drift out
of date the way a markdown sample could.

## Before you start: register a topic

Your remediation only runs if a `catalog.Rule` names your topic. Add
your topic string to `catalog.Topics` (`pkg/doctor/internal/catalog/rules.go`)
so the CLI recognizes it, and make sure the doctor_rules API has a rule
configured under that same topic string — `ParseAPIRule` sets
`Rule.RemediationTopic` to the rule's own `topic` (the API has no
separate field for this), and `judge.FetchRule` fetches it live; there's
no local `Rule` literal to author in this repo. The rule is what doctor's
scanner/judge engine use to find and generically classify matches in
source code; your remediation is a separate, later step that runs
against the findings the rule produced.

## The two steps

1. **Write a type satisfying `Implementation`**, with an exported
   constructor — see [Implementation](#implementation) below.
   ```go
   func New() remediation.Implementation { return implementation{} }
   ```
2. **Add it to the `remediation.Set` built in `pkg/cmd/root.go`**, next
   to the existing `dpm` entry, keyed by your topic string:
   ```go
   rootCmd.AddCommand(doctor.NewDoctorCmd(&Config, remediation.Set{
   	dpm.Topic:     dpm.New(),
   	"my_topic":    mytopic.New(),
   }))
   ```

That's it — doctor's engine (`cmd.go`) looks your topic up in the `Set`
by its bare string and renders whatever `Result` your `Run` returns. It
never calls into your package directly otherwise, and never needs to
import it either — only `root.go` does.

## `Implementation`

```go
type Implementation interface {
	Run(ctx Context) (Result, error)
}
```

`Run` should never mutate the Stripe account or the scanned directory —
doctor only diagnoses; remediation is always manual, guided by what you
return. A non-nil `error` is treated as an operational failure (surfaced
to the user and exits non-zero) — if you have *some* evidence but
couldn't gather all of it (e.g. no credentials), prefer returning a
`Result` with `Status: StatusUnknown` and a `nil` error over returning an
error, so partial evidence doctor already has isn't thrown away.

### `Context`

Everything your `Run` gets handed:

| Field           | Type                | What it's for                                                                              |
| --------------- | -------------------- | -------------------------------------------------------------------------------------------- |
| `Dir`           | `string`             | The scanned directory — pass to your own code-signal scans (see `codesignals` below).        |
| `Rule`          | `catalog.Rule`       | The rule that matched (message, docs, action text, etc.), in case you need it.               |
| `Findings`      | `[]catalog.Finding`  | Already computed by doctor's normal scan — reuse it; don't re-scan.                          |
| `Config`        | `*config.Config`     | Pass to `stripeapi.LoadCredentials` to fetch read-only account evidence.                     |
| `StripeAccount` | `string`             | The `--stripe-account` flag's value, for Connect-scoped lookups — pass through to `stripeapi`. |

## `Result`

What your `Run` returns. Most fields are optional — set only what your
topic actually has something to say about.

| Field             | Type                     | Required? | Purpose                                                                                                   |
| ----------------- | ------------------------ | --------- | ----------------------------------------------------------------------------------------------------------- |
| `Topic`           | `string`                 | yes       | Your topic string (same as your `remediation.Set` key).                                                    |
| `Status`          | `Status`                 | yes       | Top-line verdict — see [Status](#status).                                                                  |
| `Message`         | `string`                 | yes       | One sentence naming why. Shown up front, and used as the JSON `"message"`.                                |
| `Title`           | `string`                 | no        | Plain-English name for the migration, for the text action-plan renderer. Falls back to the bare topic string. |
| `Summary`         | `[]SummaryRow`           | no        | The short summary table shown up front — see [`SummaryRow`](#summaryrow).                                 |
| `Evidence`        | `map[string]any`         | no        | Free-form supporting data. doctor doesn't interpret it — just passes it through to `--json` for debugging. |
| `FindingVerdicts` | `map[string]string`      | no        | Per-finding verdict overrides — see [Overriding per-finding verdicts](#overriding-per-finding-verdicts).   |
| `Notes`           | `[]string`               | no        | Standing action items independent of `Status` (e.g. "go check this Dashboard setting").                    |
| `Findings`        | `[]Finding`              | no        | The numbered, human-facing action plan — see [`Finding`](#finding).                                        |
| `Report`          | `any`                    | no        | Your own fully custom `--json` document, replacing doctor's generic schema entirely. See [`Report`](#report). |

A `Result` that sets only `Topic`/`Status`/`Message` still renders fine —
doctor falls back to its generic summary/details layout. See `Example` in
`pkg/doctor/remediation/example` for exactly that minimal shape.

### `Status`

```go
const (
	StatusReady   Status = "ready"   // safe to apply as-is
	StatusCaution Status = "caution" // apply, but Message names something to fix first
	StatusBlocked Status = "blocked" // not safe yet; Message names the blocker
	StatusReview  Status = "review"  // needs human judgment, not auto-decidable
	StatusUnknown Status = "unknown" // couldn't gather enough evidence (e.g. no credentials)
)
```

`JSONStatus(s Status) string` maps these to the vocabulary used by
`FindingsJSON` and any custom `Report`: `ready`, `action_required`,
`blocked`, `review_required`, `unknown`.

### `SummaryRow`

```go
type SummaryRow struct {
	Category string
	Value    string
}
```

One line in the summary table shown at the top of the text action-plan
renderer — doctor doesn't hardcode what the categories should be, or how
many rows there are; it just aligns `Category` into a column and prints
`Value` next to it. dpm, for example, builds three rows itself
(`"Code"`, `"Dashboard"`, `"Webhooks"`), computing each `Value` from its
own evidence — but another topic could report two rows, five rows, or
rows with entirely different categories, however its own remediation is
shaped. A `nil`/empty `Result.Summary` means no table is shown at all.

### Overriding per-finding verdicts

Doctor's generic `judge` package has no topic-specific knowledge — every
scanned finding gets a generic `REVIEW` verdict by default. If your topic
can do better (e.g. classify a finding as safe to apply, or as an
account-wide fact), set `Result.FindingVerdicts`:

```go
FindingVerdicts map[string]string
```

A map from `FindingKey(file, line)` to a `"CLASS: message"` string (the
same shape `judge.Evaluate` itself produces, e.g. `"CANDIDATE: ..."`).
Set the same value for every key to apply one account-wide fact uniformly
(e.g. "recent traffic is all on an old API version"), or a different
value per key to classify each finding on its own code shape. A finding
with no entry keeps whatever generic verdict `judge.Evaluate` already
gave it.

```go
func FindingKey(file string, line int) string
```

Builds the map key from a `catalog.Finding`'s `File`/`Line` — use this
rather than formatting the string yourself, so your keys always match
what doctor's engine looks up.

### `Finding`

The single thing you write to drive **both** doctor's text action-plan
renderer and (via `FindingsJSON`) your own `--json` contract — one
heading, a few body paragraphs, optional line items, describing one
discrete, reportable fact (a flagged code location, a webhook event, or
similar).

```go
type Finding struct {
	Heading string
	Body    []string

	ID                       string
	Status                   Status // falls back to the enclosing Result's Status when empty
	Implication              string
	ActionDescription        string
	Items                    []Item
	RequiresVerification     []string
	SuggestedParameter       *Param
	SafeToApplyWithoutReview bool
	Evidence                 any
}
```

| Field                      | Text renderer                                        | `FindingsJSON`                                                                    |
| --------------------------- | ----------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `Heading`                  | The numbered section's title.                         | *(no equivalent — `ID` covers it)*                                                |
| `Body`                     | Each entry is a paragraph, word-wrapped.              | `Body[0]` → `"implication"`, `Body[1]` → `"recommended_action.description"` (fallback — see below) |
| `ID`                       | *(unused)*                                            | The finding's `"id"`. **A `Finding` with `ID == ""` is skipped by `FindingsJSON`.** |
| `Status`                   | *(unused by the text renderer)*                       | The finding's own `"status"`, mapped through `JSONStatus`. Falls back to the `Result`'s `Status` when empty. |
| `Implication`              | *(unused)*                                            | Overrides the `Body[0]` fallback, when the text and JSON prose should say different things. |
| `ActionDescription`        | *(unused)*                                            | Overrides the `Body[1]` fallback, same reasoning.                                 |
| `Items`                    | Each is a sub-heading + body line.                    | Feeds the default evidence envelope `{"items": [...]}` via each `Item.Fields` — ignored if `Evidence` is set. |
| `RequiresVerification`     | Rendered as a "Before applying this, confirm:" checklist. | `"recommended_action.requires_verification"`                                 |
| `SuggestedParameter`       | *(unused)*                                            | `"recommended_action.suggested_parameter"`                                       |
| `SafeToApplyWithoutReview` | *(unused)*                                            | `"recommended_action.safe_to_apply_without_review"`                              |
| `Evidence`                 | *(unused)*                                            | Overrides the default `{"items": [...]}` envelope entirely, for a bespoke shape (e.g. `{"locations": [...], "api_version_observation": {...}}`). |

**A `Finding` with `Heading == ""` is skipped by the text renderer** — a
way to opt a section out of text (e.g. no webhook signals were scanned at
all) without touching JSON. Symmetrically, **a `Finding` with `ID == ""`
is skipped by `FindingsJSON`** — for prose that's tied to `Result.Status`
rather than one independently reportable fact (dpm's own "verify routing"
reminder is like this: see `dashboardFinding` in
`dpm/main.go`).

A body paragraph containing a literal `"\n"` is split into independently
word-wrapped segments with no blank line between them — useful for a
lead-in sentence followed by a bare URL, which (having no spaces) is left
intact on its own line regardless of width.

### `Item`

```go
type Item struct {
	Heading string
	Body    string
	Fields  any
}
```

One line item under a `Finding` — a flagged code location, a webhook
event, or similar. `Heading`/`Body` are the text sub-heading and body
line (e.g. `"lib/checkout.rb:11"` / `"Current list: ['card']"`). `Fields`,
when set, feeds `FindingsJSON`'s default evidence envelope — ignored if
the enclosing `Finding.Evidence` overrides that envelope.

### `Param`

```go
type Param struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}
```

Names a suggested parameter/value pair, for `Finding.SuggestedParameter`
— e.g. `{Name: "automatic_payment_methods[enabled]", Value: true}`.

### `FindingsJSON`

```go
func FindingsJSON(findings []Finding, fallback Status) []FindingJSON
```

Converts `Finding`s to their generic `--json` shape (`FindingJSON`), for
a topic building its own `Result.Report` to embed under a `"findings"`
key. Skips any `Finding` with `ID == ""`. `fallback` supplies the status
for any included `Finding` whose own `Status` is empty — pass the
enclosing `Result.Status`.

```go
type FindingJSON struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	Evidence          any    `json:"evidence,omitempty"`
	Implication       string `json:"implication,omitempty"`
	RecommendedAction any    `json:"recommended_action,omitempty"`
}
```

Calling this instead of hand-authoring your JSON findings array a second
time is the point: the structured facts (`ID`, `Items`,
`RequiresVerification`, `SuggestedParameter`, `Evidence`) only have to be
written once, so they can't drift between your text and JSON output the
way two independently-authored descriptions of the same fact can.

### `Report`

```go
Report any `json:"-"`
```

When set, this is emitted as the **entire** `--json` body in place of
doctor's generic schema (hence `json:"-"` — it isn't nested inside
`remediation_result`). Use this when you want full control over your
`--json` contract's shape — your own `schema_version`, top-level
`status`/`summary` fields, and so on — rather than doctor's generic
`topic`/`message`/`findings`/`summary`/`stats` shape. A `Result` that
leaves `Report` `nil` is rendered via the generic schema as before. See
`pkg/doctor/remediation/dpm/jsonreport.go` for a complete
worked example (schema key `"dynamic_payment_methods"`).

## Helper packages

Your `Run` almost certainly needs to fetch its own evidence. These are
the packages doctor's engine itself uses, and they're yours to use too —
none of them know anything about your topic specifically.

### `pkg/doctor/internal/stripeapi` — read-only Stripe API calls

```go
func LoadCredentials(cfg *config.Config) (stripe.Credentials, error)
func NewClient(creds stripe.Credentials) (*stripe.Client, error)
func Get(client *stripe.Client, stripeAccount, path, query string, headers map[string]string, out any) error
```

`LoadCredentials` resolves **live-mode credentials only** (the CLI's
normal login/API-key resolution) — if the active context is actually in
test mode, it errors rather than silently resolving test-mode
credentials. A test-mode account's facts aren't a meaningful stand-in for
the real account state these calls exist to check. Typical use:

```go
creds, err := stripeapi.LoadCredentials(ctx.Config)
if err != nil { /* return Result{Status: StatusUnknown, ...}, nil */ }
client, err := stripeapi.NewClient(creds)
...
var resp someShape
err = stripeapi.Get(client, ctx.StripeAccount, "/v1/events", "limit=20", nil, &resp)
```

### `pkg/doctor/internal/codesignals` — your own code-signal scans

```go
type PackSignals struct {
	WebhookEvents []WebhookEvent
}
type WebhookEvent struct {
	Event string
	Note  string // shown to the human if this event's handler isn't found
}

func ScanSignals(root string, sig *PackSignals) *HandlerSignals
```

Declare the webhook events (or other signals) your migrated integration
expects, and scan for them — doctor's engine doesn't wire in any
generically; "code signals" aren't a rule-level concept. `ScanSignals`
returns:

```go
type HandlerSignals struct {
	Events     []EventSignal
	AllPresent bool
}
type EventSignal struct {
	Event   string
	Note    string
	Files   []string // which scanned files mention it, if Present
	Present bool
}
```

These are honest substring checks, reported as signals — never verdicts.

## Minimal example

The simplest possible `Implementation` — no live evidence, no custom
rendering:

```go
package mytopic

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

const Topic = "my_topic"

func New() remediation.Implementation { return implementation{} }

type implementation struct{}

func (implementation) Run(ctx remediation.Context) (remediation.Result, error) {
	return remediation.Result{
		Topic:   Topic,
		Status:  remediation.StatusReview,
		Message: fmt.Sprintf("found %d location(s) that need manual review", len(ctx.Findings)),
	}, nil
}
```

Then add `mytopic.Topic: mytopic.New()` to the `remediation.Set` in
`pkg/cmd/root.go`. That's a fully working remediation — it renders via
doctor's generic layout in both text and `--json` modes.

See `pkg/doctor/remediation/example`'s `Example` (this exact shape) and
`Example_actionPlan` (adding `Title`/`Findings`, and converting to JSON
via `FindingsJSON`) for runnable versions of both.

## Testing

Model your tests on `dpm/dynamic_test.go` — it tests the
building blocks (version/config-fact derivation, finding classification,
`assess`) directly as plain functions, rather than through the
`Implementation` interface, so they don't need real Stripe credentials.
Run the package's own `Example` functions with
`go test -run Example -v ./pkg/doctor/remediation/...` to confirm the
documented shapes still compile and produce the documented output.
