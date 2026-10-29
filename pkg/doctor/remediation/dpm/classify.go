package dpm

import (
	"regexp"
	"strings"
)

var quotedMethodRe = regexp.MustCompile(`["']([a-z0-9_]+)["']`)

// percentArrayRe matches Ruby's unquoted %w[...]/%i[...] array literal
// (e.g. "%w[card ideal bancontact]") — its members have no quote
// characters for quotedMethodRe to match.
var percentArrayRe = regexp.MustCompile(`^%[wi]\s*[\[({<]([a-z0-9_\s]*)[\])}>]$`)

// quotedMethods extracts the static method names from a finding's value
// expression, e.g. "['card', 'ideal']" or "%w[card ideal]" -> ["card", "ideal"].
func quotedMethods(value string) []string {
	v := strings.TrimSpace(value)
	if m := percentArrayRe.FindStringSubmatch(v); m != nil {
		return strings.Fields(m[1])
	}
	var out []string
	for _, m := range quotedMethodRe.FindAllStringSubmatch(v, -1) {
		out = append(out, m[1])
	}
	return out
}

var identRe = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

// classifyIntent buckets a finding's value expression:
//
//	default-shaped — static list of only card/link
//	deliberate     — static single non-card method (['oxxo'])
//	static         — static multi-method list beyond card/link
//	dynamic        — computed at runtime (identifiers, conditionals)
func classifyIntent(value string) string {
	v := strings.TrimSpace(value)
	// Stripping "%w[card ideal]"'s brackets/whitespace below would leave
	// "wcardideal", which identRe misreads as an identifier reference.
	// Handle %w/%i literals before that check runs.
	if percentArrayRe.MatchString(v) {
		return bucketMethods(quotedMethods(v))
	}
	quoted := regexp.MustCompile(`["']([a-z0-9_]+)["']`).FindAllStringSubmatch(v, -1)
	stripped := regexp.MustCompile(`["'][a-z0-9_]*["']|stripe\.StringSlice|\[\]string|Arrays\.asList|List\.of|[\[\]{},()\s]|new|List|string`).ReplaceAllString(v, "")
	if identRe.MatchString(stripped) || strings.ContainsAny(v, "?:") {
		return "dynamic"
	}
	if len(quoted) == 0 {
		return "dynamic"
	}
	return bucketMethods(quotedMethods(v))
}

// bucketMethods classifies an extracted list of method names into
// classifyIntent's static buckets.
func bucketMethods(methods []string) string {
	onlyCardLink := true
	for _, m := range methods {
		if m != "card" && m != "link" {
			onlyCardLink = false
		}
	}
	if onlyCardLink {
		return "default-shaped"
	}
	if len(methods) == 1 {
		return "deliberate"
	}
	return "static"
}

// intentVerdict turns a classifyIntent bucket into a per-finding verdict.
func intentVerdict(intent string) string {
	switch intent {
	case "default-shaped":
		return "CANDIDATE: hardcoded to card/link only — removing this lets the Dashboard manage payment methods instead (confirm the frontend uses Payment Element)"
	case "deliberate":
		return "SKIP: restricted to a single payment method, which looks intentional — consider excluded_payment_method_types instead"
	case "dynamic":
		return "REVIEW: this value is computed at runtime — understand the routing logic before changing it"
	default:
		return "REVIEW: hardcoded list of multiple payment methods — if this restriction is intentional, use automatic_payment_methods[enabled]=true with excluded_payment_method_types naming whatever should stay off instead of a fixed allow-list; if not, it's a CANDIDATE for removal"
	}
}

// versionFix is the account-wide fix the account's recent API-version
// evidence dictates, applying uniformly regardless of any one finding's
// code shape. A zero-value versionFix (empty Class) means the API
// version doesn't change the fix.
//
// A pre-cutoff version is CAUTION, not BLOCKED: the fix is known and
// actionable (add automatic_payment_methods[enabled]=true), not a dead
// end. Only NoEvents stays REVIEW, since there's no fix to name yet.
type versionFix struct {
	Class       string
	Reason      string
	Instruction string
}

func findingsVersionFix(vf versionFacts) versionFix {
	switch {
	case vf.NoEvents:
		return versionFix{
			Class:       "REVIEW",
			Reason:      "Couldn't find any recent API traffic for this account, so its API version is unknown.",
			Instruction: "Confirm it's at/after " + dpmCutoff + " before removing payment_method_types — on an older version, removing it by itself silently drops payment methods.",
		}
	case vf.VersionsMixed:
		return versionFix{
			Class:       "CAUTION",
			Reason:      "This account's recent traffic mixes API versions from before and after " + dpmCutoff + ".",
			Instruction: "Plain removal is only safe for the newer requests; on the older ones, add automatic_payment_methods[enabled]=true instead, which keeps them working on every version.",
		}
	case !vf.VersionsOK:
		return versionFix{
			Class:       "CAUTION",
			Reason:      "Recent API traffic for this account uses versions before " + dpmCutoff + ". On those versions, you must explicitly enable automatic payment methods. Removing payment_method_types alone is not enough.",
			Instruction: "In each flow, confirm the API version, then replace payment_method_types with automatic_payment_methods[enabled]=true.",
		}
	default:
		return versionFix{}
	}
}
