package doctor

// The agent playbook: `stripe guide` documents the JSON schemas and
// exit-code contract as the machine interface.

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewGuideCmd builds the `stripe guide` agent-playbook command.
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

Verbs are generic; the migration is a topic argument. Topics:
  dpm                remove payment_method_types (fixable: doctor -> fix -> doctor)
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
message names the replacement; fix exits 2 by design — apply the rename per
the docs, then re-run doctor for exit 0. For source-types, run topic dpm
afterwards: the renamed param is dpm's target.
Every command supports --json (single JSON object on stdout; logs on stderr)
and --yes (auto-approve confirmations). Exit codes: 0 clean/verified,
1 findings-present/not-verified, 2 operational error.

## 1. Diagnose
    stripe doctor dpm <dir> --json
Exit 0 → nothing to migrate; stop. Exit 1 → .findings[] each with
file/line/col, via (resolution mechanism), value, and verdict_class:
  CANDIDATE  safe removal target, account preconditions met
  SKIP       deliberate restriction — do not remove
  REVIEW     needs human judgment (dynamic value / static multi)
  CAUTION/BLOCKED  account API version predates 2023-08-16 — removal
             without automatic_payment_methods[enabled]=true drops
             methods. fix handles this: it forks by version and
             REPLACES the parameter with the companion there (see
             step 2's .companion). Surface the verdict to the human
             but the code change remains safe to preview.
  UNKNOWN    account facts unavailable (.degraded says why) — treat
             as REVIEW.
.account carries the evidence (event_api_versions, dashboard_configured).
Only proceed to step 2 for CANDIDATE findings.

Also in the doctor report, credentials or not:
  .webhook_handlers  does YOUR code mention the three delayed-notification
                     event types (all_present should be true before enabling
                     delayed methods)
  .frontend_warnings legacy Card Element signals — dashboard-managed methods
                     cannot render there; server-side removal alone strands
                     them. Surface these to the human.

## 2. Preview the remediation
    stripe fix dpm <dir> --json
Dry-run. Per file: edits[] (byte spans + label); reparse must be "clean".
.companion reports the version fork: mode "insert" means removals of
payment_method_types on PaymentIntents/SetupIntents become REPLACEMENTS
that splice in automatic_payment_methods[enabled]=true (required below
the 2023-08-16 cutoff, harmless above it); mode "omit" means account
traffic is all at/after the cutoff and plain removal preserves behavior.
.companion.reason carries the account evidence; --offline forces the
insert branch (the only choice correct at any version). Checkout
Sessions/Payment Links never get the insert (no such parameter there).
Server-side confirmation: a create with confirm:true and no return_url
would 400 at runtime once automatic_payment_methods is in effect —
which is BOTH branches of the fork (inserted below the cutoff,
default-on above it), so confirm sites always get explicit handling:
  --return-url <url> given → companion + return_url (keeps redirect
    methods; the URL is validated — http(s), no quotes/spaces)
  old hardcoded list named redirect-based methods (ideal, bancontact,
    giropay, sofort, p24, ...) and no return_url → the site is GATED
    (.skipped intent "confirm-redirect"): pinning would silently drop
    a method the merchant used; removal alone would 400. ASK THE HUMAN
    for a return_url and rerun — that is the intended resolution.
  otherwise → companion + allow_redirects:"never";
    .companion.pinned_sites lists exactly which file:line got the pin,
    and each companion edit carries .files[].edits[].variant
    (plain | never-pin | return_url) — branch on that, not on labels.
NOTE: .companion.mode "omit" + inserts>0 is a VALID state: confirm
sites force an explicit companion even when plain sites need none
(APM is default-on at that version); a note explains it.
--apply with --json requires --yes (no interactive consent in JSON).
Java builders are judged per BUILDER, not per method-add: a builder
adding card+ideal is one site — gates apply to the whole list.
.companion.account names the account whose facts decided the fork. A
Stripe-Version pinned in CODE below the cutoff overrides an omit
verdict (the events census only reflects the account default).
Also in doctor's .account: enabled_but_unavailable lists methods
toggled ON whose capability is inactive — they will NOT render; treat
them as missing when judging method coverage.
Connect: for direct charges (Stripe-Account header in the user's code),
pass --stripe-account acct_... so the CONNECTED account's configuration
and traffic govern the fork and the doctor's verdicts.
The gate skips dynamic values and deliberate single-method restrictions
(.skipped, with reasons) — that is intentional; --all overrides, but only
after a human reviews each skipped finding.

## 3. Apply
    stripe fix dpm <dir> --apply --yes --json
Writes only reparse-clean files (.files[].written=true). A per-file write
failure is recorded in .files[].error and does not abort the rest.

## 4. Confirm the code change
    stripe doctor dpm <dir> --json
Exit 0 proves the parameter is gone.

## 5. Confirm runtime behavior (recommended)
    stripe doctor dpm --live --json
Adds .live_drill (webhook round-trip via stripe listen/trigger); require
.live_drill.verified == true.

## Notes
- All account access is read-only GETs with the CLI's stored test-mode
  credentials; nothing is ever written to the Stripe account.
`

// ---------- hidden: parse-dump ----------
