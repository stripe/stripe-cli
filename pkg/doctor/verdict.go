package doctor

// Verdicts: account facts + finding intent -> a judgment per finding.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stripe/stripe-cli/pkg/config"
)

var identRe = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

// classifyIntent inspects a finding's value expression and buckets it:
//
//	default-shaped — static list of only card/link: the classic pre-DPM
//	                 hardcode and the migration's actual target
//	deliberate     — static single non-card method (['oxxo']): per-method
//	                 integration, almost certainly intentional
//	static         — static multi-method list beyond card/link

// dynamic        — computed at runtime (identifiers, conditionals)
func classifyIntent(value string) string {
	v := strings.TrimSpace(value)
	quoted := regexp.MustCompile(`["']([a-z0-9_]+)["']`).FindAllStringSubmatch(v, -1)
	stripped := regexp.MustCompile(`["'][a-z0-9_]*["']|stripe\.StringSlice|\[\]string|Arrays\.asList|List\.of|[\[\]{},()\s]|new|List|string`).ReplaceAllString(v, "")
	if identRe.MatchString(stripped) || strings.ContainsAny(v, "?:") {
		return "dynamic"
	}
	if len(quoted) == 0 {
		return "dynamic"
	}
	onlyCardLink := true
	for _, q := range quoted {
		if q[1] != "card" && q[1] != "link" {
			onlyCardLink = false
		}
	}
	if onlyCardLink {
		return "default-shaped"
	}
	if len(quoted) == 1 {
		return "deliberate"
	}
	return "static"
}

// quotedMethods extracts the static method names from a finding's value.
func quotedMethods(value string) []string {
	var out []string
	for _, m := range regexp.MustCompile(`["']([a-z0-9_]+)["']`).FindAllStringSubmatch(value, -1) {
		out = append(out, m[1])
	}
	return out
}

// pmConfigCommand builds the exact `stripe payment_method_configurations`
// command that would turn the given methods on, so a verdict can name the
// fix instead of just pointing at the Dashboard. configID "" means no
// active configuration exists yet, so the command creates one instead of
// updating. Each method name doubles as the API field name
// (payment_method_configurations' per-method params are snake_case, same
// as the SDK's payment_method_types strings), so no translation is needed.
func pmConfigCommand(configID string, methods []string) string {
	var b strings.Builder
	b.WriteString("stripe payment_method_configurations ")
	if configID == "" {
		b.WriteString("create")
	} else {
		b.WriteString("update " + configID)
	}
	for _, m := range methods {
		fmt.Fprintf(&b, " --%s[display_preference][preference]=on", m)
	}
	return b.String()
}

// verdict combines one finding's intent and value with the account facts.
func verdict(intent, value string, f *accountFacts) string {
	if f.NoEvents {
		// No traffic sampled: fabricating a version claim would be worse
		// than admitting ignorance.
		return "REVIEW: couldn't find any recent API traffic for this account, so its API version is unknown. Confirm it's at/after " + dpmCutoff + " before removing payment_method_types — on an older version, removing it by itself silently drops payment methods."
	}
	if !f.VersionsOK {
		if f.VersionsMixed {
			return "CAUTION: this account's recent traffic mixes API versions from before and after " + dpmCutoff + ". Plain removal is only safe for the newer requests; on the older ones, removing payment_method_types by itself would silently drop payment methods — add automatic_payment_methods[enabled]=true instead, which keeps them working on every version."
		}
		return "BLOCKED: this account's recent traffic is entirely on API versions from before " + dpmCutoff + ". Plain removal only becomes safe once this account's traffic moves to " + dpmCutoff + " or later — until then, removing payment_method_types by itself silently drops payment methods. Add automatic_payment_methods[enabled]=true in its place instead of just deleting it."
	}
	if !f.ConfiguredOK {
		msg := "BLOCKED: this account has no payment methods turned on in an active Dashboard configuration."
		if methods := quotedMethods(value); len(methods) > 0 {
			msg += " Turn on the methods this code expects first — `" + pmConfigCommand(f.ActiveConfigID, methods) + "` — "
		} else {
			msg += " Turn on the payment methods you need first, e.g. `" + pmConfigCommand(f.ActiveConfigID, nil) + " --card[display_preference][preference]=on` — "
		}
		return msg + "before removing payment_method_types from your code."
	}
	// The doc's loudest warning: migration only keeps methods the Dashboard
	// has ON *and available* (toggle + active capability — both required to
	// render). Diff the hardcoded list against the governing config.
	if intent == "default-shaped" || intent == "static" {
		var missing, inactive []string
		enabled := map[string]bool{}
		for _, m := range f.EnabledMethods {
			enabled[m] = true
		}
		unavailable := map[string]bool{}
		for _, m := range f.UnavailableMethods {
			unavailable[m] = true
		}
		for _, m := range quotedMethods(value) {
			switch {
			case !enabled[m]:
				missing = append(missing, m)
			case unavailable[m]:
				inactive = append(inactive, m)
			}
		}
		if len(missing) > 0 {
			return "CAUTION: " + strings.Join(missing, ", ") + " — your code hardcodes these payment methods, but they're turned OFF in the Dashboard. Turn them on first — `" + pmConfigCommand(f.ActiveConfigID, missing) + "` — or customers will stop seeing them once this code is removed."
		}
		if len(inactive) > 0 {
			return "CAUTION: " + strings.Join(inactive, ", ") + " — your code hardcodes these payment methods, and they're turned ON in the Dashboard, but Stripe hasn't activated the capability for this account yet (not something payment_method_configurations can toggle — check Settings > Payment methods, or contact support), so they don't render for customers today either way. Activate them first."
		}
	}
	switch intent {
	case "default-shaped":
		return "CANDIDATE: hardcoded to card/link only, and the account is ready — removing this lets the Dashboard manage payment methods instead (confirm the frontend uses Payment Element)"
	case "deliberate":
		return "SKIP: restricted to a single payment method, which looks intentional — consider excluded_payment_method_types instead"
	case "dynamic":
		return "REVIEW: this value is computed at runtime — understand the routing logic before changing it"
	default:
		return "REVIEW: hardcoded list of multiple payment methods — confirm whether the restriction is still wanted"
	}
}

// buildDoctorReport combines scan findings with account facts. The rule
// decides the judgment style: the dpm pack gets full account verdicts;
// advise packs get ADVISE verdicts carrying the rule's remediation message
// (their account precondition is the version window, shown as context).
func buildDoctorReport(findings []Finding, cfg *config.Config, stripeAccount string, rule Rule) (*DoctorReport, error) {
	creds, err := loadTestCredentials(cfg)
	if err != nil {
		return nil, err
	}
	facts, err := fetchAccountFacts(creds, stripeAccount)
	if err != nil {
		return nil, err
	}

	r := &DoctorReport{
		Command: "doctor",
		Account: AccountSummary{
			ID:             facts.AccountID,
			Name:           facts.DisplayName,
			EventVersions:  facts.EventVersions,
			VersionsOK:     facts.VersionsOK,
			VersionsMixed:  facts.VersionsMixed,
			Cutoff:         dpmCutoff,
			Configs:        facts.ConfigCount,
			ActiveConfig:   facts.ActiveConfig,
			ActiveConfigID: facts.ActiveConfigID,
			MethodsOn:      facts.MethodsOn,
			MethodsOff:     facts.MethodsOff,
			EnabledMethods: facts.EnabledMethods,
			Unavailable:    facts.UnavailableMethods,
			NoRecentEvents: facts.NoEvents,
			ConfiguredOK:   facts.ConfiguredOK,
		},
		Summary: map[string]int{},
	}
	for _, f := range findings {
		intent := classifyIntent(f.Value)
		var v string
		if rule.Action == "advise" {
			v = "ADVISE: " + rule.Message
			if rule.IntroducedIn != "" {
				v += " (API " + rule.IntroducedIn + ")"
			}
		} else {
			v = verdict(intent, f.Value, facts)
		}
		class := v
		if i := strings.Index(v, ":"); i > 0 {
			class = v[:i]
		}
		r.Findings = append(r.Findings, DoctorFinding{Finding: f, Intent: intent, Verdict: v, Class: class})
		r.Summary[class]++
	}
	return r, nil
}

// ---------- code signals (substring-level, no credentials needed) ----------

// PackSignals is the pack-declared signal set: which webhook event types the
// migrated integration is expected to handle, which legacy client-side tokens

func scanOnlyReport(topic string, rule Rule, findings []Finding, stats ScanStats, why string) *DoctorReport {
	rep := &DoctorReport{Command: "doctor", Topic: topic, Degraded: why, Summary: map[string]int{}, Stats: stats}
	for _, f := range findings {
		intent := classifyIntent(f.Value)
		v, class := "UNKNOWN: account facts unavailable — "+why, "UNKNOWN"
		if rule.Action == "advise" {
			// An advise verdict is a version fact, not an account fact — it
			// needs no credentials.
			v, class = "ADVISE: "+rule.Message, "ADVISE"
			if rule.IntroducedIn != "" {
				v += " (API " + rule.IntroducedIn + ")"
			}
		}
		rep.Findings = append(rep.Findings, DoctorFinding{Finding: f, Intent: intent, Verdict: v, Class: class})
		rep.Summary[class]++
	}
	return rep
}
