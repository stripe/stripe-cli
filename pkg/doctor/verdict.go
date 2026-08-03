package doctor

// Verdicts: account facts + finding intent -> a judgment per finding.

import (
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

// verdict combines one finding's intent and value with the account facts.
func verdict(intent, value string, f *accountFacts) string {
	if f.NoEvents {
		// No traffic sampled: fabricating a version claim would be worse
		// than admitting ignorance.
		return "REVIEW: no recent API traffic to infer the account's effective version — confirm it is >= " + dpmCutoff + " before removing"
	}
	if !f.VersionsOK {
		if f.VersionsMixed {
			return "CAUTION: some recent traffic predates " + dpmCutoff + " — removal there silently drops methods unless automatic_payment_methods[enabled]=true is added (`fix` inserts it on this account)"
		}
		return "BLOCKED: recent traffic runs before " + dpmCutoff + " — removal alone would drop methods; `fix` replaces the parameter with automatic_payment_methods[enabled]=true here"
	}
	if !f.ConfiguredOK {
		return "BLOCKED: no active Dashboard payment-method configuration — configure methods before removing"
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
			return "CAUTION: " + strings.Join(missing, ", ") + " hardcoded here but OFF in the Dashboard config — enable in the Dashboard first or customers lose them on removal"
		}
		if len(inactive) > 0 {
			return "CAUTION: " + strings.Join(inactive, ", ") + " toggled ON but not available (capability inactive) — it will not render after removal; activate the capability first"
		}
	}
	switch intent {
	case "default-shaped":
		return "CANDIDATE: static card/link list, account preconditions met — removal enables Dashboard-managed methods (verify frontend is Payment Element)"
	case "deliberate":
		return "SKIP: single-method integration, restriction looks intentional — consider excluded_payment_method_types instead"
	case "dynamic":
		return "REVIEW: value computed at runtime — understand the routing logic before changing"
	default:
		return "REVIEW: static multi-method list — confirm whether the restriction is still wanted"
	}
}

// buildDoctorReport combines scan findings with account facts. The rule
// decides the judgment style: the dpm pack gets full account verdicts;
// advise packs get ADVISE verdicts carrying the rule's remediation message
// (their account precondition is the version window, shown as context).
func buildDoctorReport(findings []Finding, cfg *config.Config, stripeAccount string, rule Rule) (*DoctorReport, error) {
	key, err := loadTestKey(cfg)
	if err != nil {
		return nil, err
	}
	facts, err := fetchAccountFacts(key, stripeAccount)
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
