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
topic: dpm (payment_method_types removal readiness, judged against
account facts — see step 1's verdict_class). Other migrations previously
lived here as detect-only "advise" topics; they were removed to keep this
tool to one steel thread and may return later.
Every command supports --json (single JSON object on stdout; logs on
stderr). Exit codes: 0 clean/verified, 1 findings-present/not-verified,
2 operational error.

## 1. Diagnose
    stripe doctor dpm <dir> --json
Exit 0 → nothing to migrate; stop. Exit 1 → .findings[] each with
file/line/col, via (resolution mechanism), value, and verdict_class:
  CANDIDATE  safe to remove payment_method_types by hand, account
             preconditions met (verify frontend is Payment Element first)
  SKIP       deliberate restriction — do not remove
  REVIEW     needs human judgment (dynamic value / static multi)
  CAUTION/BLOCKED  account API version predates 2023-08-16 — removing
             payment_method_types by itself drops methods there. Add
             automatic_payment_methods[enabled]=true in its place
             instead (create-only; never on update/confirm). For
             server-side confirmation (confirm:true) sites with no
             return_url, pass allow_redirects:'never' too, unless the
             removed list named redirect-based methods (ideal,
             bancontact, giropay, sofort, p24, ...) the merchant
             actually used — then get a return_url from the human
             first. Surface the verdict message to the human; it names
             the exact condition and the exact replacement.
  CAUTION/BLOCKED  (Dashboard-config variant) the governing payment-method
             configuration has no methods on, or is missing methods this
             code hardcodes. The verdict message includes a ready-to-run
             stripe payment_method_configurations update/create command
             (one --<method>[display_preference][preference]=on flag per
             missing method) — run it as-is rather than re-deriving the
             flags.
  UNKNOWN    account facts unavailable (.degraded says why) — treat
             as REVIEW.
.account carries the evidence (event_api_versions, dashboard_configured).
Java builders are judged per BUILDER, not per method-add: a builder
adding card+ideal is one site. Connect: for direct charges (Stripe-Account
header in the user's code), pass --stripe-account acct_... so the
CONNECTED account's configuration and traffic govern the verdicts.

Also in the doctor report, credentials or not:
  .webhook_handlers  does YOUR code mention the three delayed-notification
                     event types (all_present should be true before enabling
                     delayed methods); each event in .events[] carries a
                     .note explaining what breaks if that handler is missing
                     — surface it to the human when .present is false.
Frontend signals (e.g. legacy Card Element detection) aren't implemented
yet — deferred pending a redesign.
Also in doctor's .account: enabled_but_unavailable lists methods toggled
ON whose capability is inactive — they will NOT render; treat them as
missing when judging method coverage.

## 2. Make the code change, then confirm it
    stripe doctor dpm <dir> --json
After editing per the verdict messages above, re-run the same command.
Exit 0 proves payment_method_types is gone (or safely replaced).

## 3. Confirm runtime behavior (recommended)
    stripe doctor dpm --live --json
Adds .live_drill (webhook round-trip via stripe listen/trigger); require
.live_drill.verified == true.

## Notes
- All account access, and the dpm rule itself (fetched live from the
  not-yet-public doctor_rules API), are read-only GETs with the CLI's
  stored credentials, in whichever mode (test or live) the active context
  is actually logged into — no --live flag needed. .account.livemode says
  which. --offline skips both and uses the embedded rule. Nothing is ever
  written to the Stripe account.
`

// ---------- hidden: parse-dump ----------
