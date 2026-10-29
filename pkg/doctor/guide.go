package doctor

// The agent playbook: `stripe doctor guide` documents the JSON schemas and
// exit-code contract as the machine interface.

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewGuideCmd builds the `stripe doctor guide` agent-playbook command.
func NewGuideCmd() *cobra.Command {
	return newGuideCmd()
}

func newGuideCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "guide",
		Short: "The agent playbook: step-by-step commands, JSON schemas, exit codes",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(agentGuide)
		},
	}
}

const agentGuide = `# Migration doctor — agent playbook

doctor only diagnoses: every finding's verdict message says exactly what
to change, but the code edit itself is manual (by you or the human).
Verbs are generic and take a topic argument, but today there is only one
topic: dpm (payment_method_types removal readiness). Other migrations
previously lived here as detect-only "advise" topics; they were removed to
keep this tool to one steel thread and may return later.
Every command supports --json (single JSON object on stdout; logs on
stderr). Exit codes: 0 once the report completes — check its contents,
not the exit code, for what it found (findings present or a webhook
check that didn't verify are legitimate outcomes, not failures); 2
operational error, meaning doctor couldn't produce a report at all.

## 1. Diagnose
    stripe doctor dpm <dir> --json
.message/.action/.docs are the migration rule's own top-level fields,
straight from the doctor_rules API — read these first, before the
per-finding verdicts, for the overall what/why/where-to-read-more.
An empty .findings means nothing to migrate; stop. Otherwise .findings[]
each has file/line/col, via (resolution mechanism), value, and verdict_class:
  CANDIDATE  safe to remove payment_method_types by hand (verify frontend
             is Payment Element first)
  SKIP       deliberate restriction — do not remove
  REVIEW     needs human judgment (dynamic value / static multi)
Every verdict message also carries the migration rule's own guidance
(e.g. a Dashboard deep link, or an API-version caveat) straight from the
doctor_rules API — doctor doesn't independently probe the account itself
(API version, Dashboard config) anymore, so surface the full message to
the human regardless of verdict_class; it names whatever conditions and
replacements apply.
Java builders are judged per BUILDER, not per method-add: a builder
adding card+ideal is one site.

.remediation_result (singular) — present when .doctorTopic names a topic
("dpm" today) — is what doctor actually computed for it: status
(ready/caution/blocked/review/unknown) and a message naming why. unknown
means either no local implementation is registered for that topic yet,
or gathering its evidence failed (e.g. no credentials) — not "nothing to
do." For dpm, .evidence.webhook_handlers_present (when present) says
whether YOUR code mentions the three delayed-notification event types —
a false value there is named in .message too, as a caution to add
handlers before relying on delayed methods. Code signals aren't a
generic doctor feature: each topic's remediation declares and checks
whatever signals it needs on its own.

## 2. Make the code change, then confirm it
    stripe doctor dpm <dir> --json
After editing per the verdict messages above, re-run the same command.
An empty .findings proves payment_method_types is gone (or safely
replaced) — the exit code stays 0 either way, so check the field.

## 3. Confirm runtime behavior (recommended)
    stripe doctor dpm --verify-webhooks --json
Adds .webhook_check (webhook round-trip via stripe listen/trigger); require
.webhook_check.verified == true.

## Notes
- The dpm rule comes from an embedded local snapshot today, not a live
  fetch (judge.FetchRule exists but isn't currently called). Any
  remediation that does fetch live account evidence (see
  .remediation_result above) uses LIVE MODE credentials only
  (pkg/doctor/stripeapi) — if the active context is in test mode, that
  remediation's result comes back status unknown naming the mismatch,
  rather than silently assessing against test-mode data. --stripe-account
  scopes those lookups to a Connect connected account. Nothing is ever
  written to the Stripe account.
`

// ---------- hidden: parse-dump ----------
