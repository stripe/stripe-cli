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

Verbs are generic; the migration is a topic argument. doctor only
diagnoses: every finding's verdict message says exactly what to change,
but the code edit itself is manual (by you or the human). Topics:
  dpm                payment_method_types removal readiness (judged
                     against account facts — see step 1's verdict_class)
  elements           WHICH Payment Element migration applies? Run this first
                     when unsure: .triage detects each from-state with file
                     evidence — Charges-era (prerequisite migration), legacy
                     Card Element (-> pe or ewcs), PaymentMethod server
                     handoff (-> ct, mandatory: from-state is unsupported),
                     and already-migrated markers
  pe                 legacy per-PM Elements -> Payment Element (advise;
                     signals: confirm<PM>Payment/Setup family, card Element;
                     expected events payment_intent.succeeded/processing/
                     payment_failed; note: Stripe prefers ewcs for most)
  ct                 PaymentMethod handoff -> Confirmation Tokens (advise;
                     anchor mandate_data@PI create/confirm; signals:
                     createPaymentMethod, paymentMethodId handoff)
  ewcs               Elements with Checkout Sessions readiness report (advise;
                     recommendation-class — no version gate; the doctor report
                     carries the machine-checkable prerequisites: .findings are
                     the create calls to migrate, .webhook_handlers diffs the
                     four expected checkout.session.* events, .frontend_warnings
                     lists from-state client tokens with their replacements,
                     .manifest_checks enforces @stripe/stripe-js>=8 and
                     @stripe/react-stripe-js>=5; the actual rewrite is agent/
                     human work per the docs link)
  flex               flexible payment features beta->GA (advise; account-gated,
                     not version-gated: anchors the incremental-auth rename and
                     final_capture semantics; the other features are param
                     ADDITIONS a presence scanner cannot see — follow the doc)
  tax-percent        tax_percent removed 2020-08-27 (advise: needs TaxRate objects)
  collection-method  billing -> collection_method rename, 2019-10-17 (advise)
  prorate            prorate -> proration_behavior, 2020-08-27 (advise)
  source-types       allowed_source_types -> payment_method_types, 2019-02-11 (advise)
Advise topics: doctor detects (verdict_class ADVISE, works offline) and the
message names the replacement — apply the rename by hand per the docs, then
re-run doctor for exit 0. For source-types, run topic dpm afterwards: the
renamed param is dpm's target.
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
                     delayed methods)
  .frontend_warnings legacy Card Element signals — dashboard-managed methods
                     cannot render there; server-side removal alone strands
                     them. Surface these to the human.
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
- All account access is read-only GETs with the CLI's stored test-mode
  credentials; nothing is ever written to the Stripe account.
`

// ---------- hidden: parse-dump ----------
